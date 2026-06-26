package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/go-ego/gse"
	"github.com/tidwall/buntdb"
)

const (
	gseTokenizerName = "gse"
	gseAnalyzerName  = "gse"
	bleveBatchSize   = 500
)

// gseTokenizer 基于 gse 的 Bleve 分词器
type gseTokenizer struct {
	seg *gse.Segmenter
}

func (t *gseTokenizer) Tokenize(input []byte) analysis.TokenStream {
	segments := t.seg.Segment(input)

	tokens := make(analysis.TokenStream, 0, len(segments))
	pos := 1

	for _, seg := range segments {
		text := seg.Token().Text()
		if len(strings.TrimSpace(text)) == 0 {
			continue
		}
		tokens = append(tokens, &analysis.Token{
			Term:     []byte(text),
			Start:    seg.Start(),
			End:      seg.End(),
			Position: pos,
			Type:     analysis.Ideographic,
		})
		pos++
	}

	return tokens
}

// TODO: sync.Once 保证 gse 初始化只执行一次。如果首次初始化因临时原因失败
// （如词典文件被占用），后续所有调用都会返回相同错误，搜索功能永久不可用。
// 可改用 sync.OnceValues（Go 1.21+）配合重试计数器，或在 NewIndexManager
// 中显式初始化并暴露重试入口，避免应用必须重启才能恢复。
var (
	gseOnce      sync.Once
	gseSingleton *gse.Segmenter
	gseInitErr   error
)

func gseTokenizerConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.Tokenizer, error) {
	gseOnce.Do(func() {
		seg, err := gse.New()
		if err != nil {
			gseInitErr = fmt.Errorf("init gse segmenter: %w", err)
			return
		}
		gseSingleton = &seg
	})
	if gseInitErr != nil {
		return nil, gseInitErr
	}
	return &gseTokenizer{seg: gseSingleton}, nil
}

func init() {
	registry.RegisterTokenizer(gseTokenizerName, gseTokenizerConstructor)
}

// buildIndexMapping 构建 Bleve 索引映射：gse analyzer + 各实体文档结构
func buildIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// 注册 gse analyzer
	indexMapping.AddCustomAnalyzer(gseAnalyzerName, map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": gseTokenizerName,
	})
	indexMapping.DefaultAnalyzer = gseAnalyzerName

	// 站点文档映射
	siteMapping := bleve.NewDocumentMapping()
	siteMapping.AddFieldMappingsAt("_type", keywordField())
	siteMapping.AddFieldMappingsAt("title", textField())
	siteMapping.AddFieldMappingsAt("description", textField())
	siteMapping.AddFieldMappingsAt("domain", keywordField())
	siteMapping.AddFieldMappingsAt("domain_text", textField())
	siteMapping.AddFieldMappingsAt("tags", keywordField())
	siteMapping.AddFieldMappingsAt("updated_at", datetimeField())
	indexMapping.AddDocumentMapping("site", siteMapping)

	// 书签文档映射
	bmMapping := bleve.NewDocumentMapping()
	bmMapping.AddFieldMappingsAt("_type", keywordField())
	bmMapping.AddFieldMappingsAt("title", textField())
	bmMapping.AddFieldMappingsAt("description", textField())
	bmMapping.AddFieldMappingsAt("domain", keywordField())
	bmMapping.AddFieldMappingsAt("domain_text", textField())
	bmMapping.AddFieldMappingsAt("tags", keywordField())
	bmMapping.AddFieldMappingsAt("url", keywordField())
	bmMapping.AddFieldMappingsAt("updated_at", datetimeField())
	indexMapping.AddDocumentMapping("bookmark", bmMapping)

	// 笔记文档映射
	noteMapping := bleve.NewDocumentMapping()
	noteMapping.AddFieldMappingsAt("_type", keywordField())
	noteMapping.AddFieldMappingsAt("title", textField())
	noteMapping.AddFieldMappingsAt("body", textField())
	noteMapping.AddFieldMappingsAt("updated_at", datetimeField())
	indexMapping.AddDocumentMapping("note", noteMapping)

	// 媒体文档映射
	mediaMapping := bleve.NewDocumentMapping()
	mediaMapping.AddFieldMappingsAt("_type", keywordField())
	mediaMapping.AddFieldMappingsAt("folder_id", keywordField())
	mediaMapping.AddFieldMappingsAt("media_type", keywordField())
	mediaMapping.AddFieldMappingsAt("filename", textField())
	mediaMapping.AddFieldMappingsAt("tags", keywordField())
	mediaMapping.AddFieldMappingsAt("updated_at", datetimeField())
	indexMapping.AddDocumentMapping("media", mediaMapping)

	return indexMapping
}

// textField 分词文本字段（使用 gse 中文分词）
func textField() *mapping.FieldMapping {
	f := bleve.NewTextFieldMapping()
	f.Analyzer = gseAnalyzerName
	return f
}

// keywordField 精确匹配字段（不分词）
func keywordField() *mapping.FieldMapping {
	f := bleve.NewKeywordFieldMapping()
	return f
}

// datetimeField 时间字段
func datetimeField() *mapping.FieldMapping {
	f := bleve.NewDateTimeFieldMapping()
	return f
}

// BleveDoc 用于批量索引的文档结构
type BleveDoc struct {
	ID     string
	Fields map[string]interface{}
}

// IndexDoc 索引单个文档到 Bleve（写操作后调用）。
// 自动注入 _type 字段，调用方无需手动设置。
// 失败时记录到脏队列，不阻塞主流程。
func (s *Store) IndexDoc(id string, docType string, fields map[string]interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc := make(map[string]interface{}, len(fields)+1)
	for k, v := range fields {
		doc[k] = v
	}
	doc["_type"] = docType

	if err := s.idx.IndexDoc(id, doc); err != nil {
		slog.Warn("bleve index failed", "id", id, "err", err)
		s.addDirtyItem(id, docType)
		return err
	}
	s.removeDirtyItem(id)
	return nil
}

// DeleteDoc 从 Bleve 索引中删除文档。失败时记录到脏队列。
func (s *Store) DeleteDoc(id string, docType string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.idx.DeleteDoc(id); err != nil {
		slog.Warn("bleve delete failed", "id", id, "err", err)
		s.addDirtyItem(id, docType)
		return err
	}
	s.removeDirtyItem(id)
	return nil
}

// Search 执行 Bleve 搜索查询。
func (s *Store) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.idx.Search(req)
}

// RebuildIndexByType 按文档类型重建索引（如 "site"、"bookmark"、"note"）。
//
// TODO: 当前将所有删除+新增放入单个 batch，万级文档时内存压力大。
// 后续应复用 bleveBatchSize 分批策略（先分批删除旧文档，再分批写入新文档），
// RebuildMediaFolderIndex 同理。
func (s *Store) RebuildIndexByType(docType string, docs []BleveDoc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldIDs, err := s.searchDocIDs("_type", docType)
	if err != nil {
		return fmt.Errorf("search existing %s docs: %w", docType, err)
	}

	batch, err := s.idx.NewBatch()
	if err != nil {
		return err
	}
	for _, id := range oldIDs {
		batch.Delete(id)
	}
	for _, doc := range docs {
		batch.Index(doc.ID, doc.Fields)
	}

	if err := s.idx.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("rebuild %s index: %w", docType, err)
	}

	s.ClearDirtyByType(docType)
	slog.Info("index rebuilt by type", "type", docType, "deleted", len(oldIDs), "indexed", len(docs))
	return nil
}

// RebuildMediaFolderIndex 按媒体文件夹重建索引。仅影响指定文件夹的文档。
func (s *Store) RebuildMediaFolderIndex(folderID string, docs []BleveDoc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldIDs, err := s.searchDocIDs("folder_id", folderID)
	if err != nil {
		return fmt.Errorf("search media docs for folder %s: %w", folderID, err)
	}

	batch, err := s.idx.NewBatch()
	if err != nil {
		return err
	}
	for _, id := range oldIDs {
		batch.Delete(id)
	}
	for _, doc := range docs {
		batch.Index(doc.ID, doc.Fields)
	}

	if err := s.idx.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("rebuild media folder %s index: %w", folderID, err)
	}

	for _, id := range oldIDs {
		s.removeDirtyItem(id)
	}
	for _, doc := range docs {
		s.removeDirtyItem(doc.ID)
	}

	slog.Info("media folder index rebuilt", "folder_id", folderID, "deleted", len(oldIDs), "indexed", len(docs))
	return nil
}

// RebuildDocs 局部重建：删除指定 ID 的旧文档，写入新文档。
func (s *Store) RebuildDocs(deleteIDs []string, newDocs []BleveDoc) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	batch, err := s.idx.NewBatch()
	if err != nil {
		return err
	}
	for _, id := range deleteIDs {
		batch.Delete(id)
	}
	for _, doc := range newDocs {
		batch.Index(doc.ID, doc.Fields)
	}

	if err := s.idx.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("rebuild docs batch: %w", err)
	}

	for _, id := range deleteIDs {
		s.removeDirtyItem(id)
	}
	for _, doc := range newDocs {
		s.removeDirtyItem(doc.ID)
	}

	slog.Info("bleve docs rebuilt", "deleted", len(deleteIDs), "indexed", len(newDocs))
	return nil
}

// searchDocIDs 按 keyword 字段精确匹配查找文档 ID 列表（内部方法，调用方须持锁）
func (s *Store) searchDocIDs(field, value string) ([]string, error) {
	query := bleve.NewTermQuery(value)
	query.SetField(field)
	req := bleve.NewSearchRequest(query)
	req.Size = math.MaxInt32
	req.Fields = []string{}

	result, err := s.idx.Search(req)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		ids = append(ids, hit.ID)
	}
	return ids, nil
}

// --- 脏队列管理 ---

// addDirtyItem 记录一条索引失败的文档到脏队列
func (s *Store) addDirtyItem(docID, docType string) {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		item := model.DirtyItem{
			DocID:    docID,
			DocType:  docType,
			FailedAt: time.Now(),
		}
		val, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, _, err = tx.Set("dirty:"+docID, string(val), nil)
		return err
	})
	if err != nil {
		slog.Warn("failed to add dirty item", "doc_id", docID, "err", err)
	}
}

// removeDirtyItem 从脏队列移除一条记录
func (s *Store) removeDirtyItem(docID string) {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete("dirty:" + docID)
		if err == buntdb.ErrNotFound {
			return nil
		}
		return err
	})
	if err != nil {
		slog.Warn("failed to remove dirty item", "doc_id", docID, "err", err)
	}
}

// foreachDirtyItem 遍历脏队列中所有条目，对每条调用 fn。
// 统一遍历逻辑，避免多处重复 AscendKeys + Unmarshal。
func (s *Store) foreachDirtyItem(tx *buntdb.Tx, fn func(key string, item model.DirtyItem)) {
	tx.AscendKeys("dirty:*", func(key, value string) bool {
		var item model.DirtyItem
		if err := json.Unmarshal([]byte(value), &item); err != nil {
			slog.Warn("skip corrupted dirty item", "key", key, "err", err)
			return true
		}
		fn(key, item)
		return true
	})
}

// HasDirtyItems 判断是否存在脏记录（前缀扫描 dirty:*）
func (s *Store) HasDirtyItems() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found bool
	s.db.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys("dirty:*", func(key, value string) bool {
			found = true
			return false
		})
		return nil
	})
	return found
}

// GetDirtyItems 获取脏队列中所有条目
func (s *Store) GetDirtyItems() []model.DirtyItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []model.DirtyItem
	s.db.View(func(tx *buntdb.Tx) error {
		s.foreachDirtyItem(tx, func(_ string, item model.DirtyItem) {
			items = append(items, item)
		})
		return nil
	})
	return items
}

// GetDirtyIndexStatus 获取索引状态摘要（按模块统计脏文档数）
func (s *Store) GetDirtyIndexStatus() model.DirtyIndexStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := model.DirtyIndexStatus{}
	s.db.View(func(tx *buntdb.Tx) error {
		s.foreachDirtyItem(tx, func(_ string, item model.DirtyItem) {
			switch item.DocType {
			case "site", "bookmark":
				status.URLCount++
			case "note":
				status.NoteCount++
			case "media":
				status.MediaCount++
			}
		})
		return nil
	})
	status.HasDirty = (status.URLCount + status.NoteCount + status.MediaCount) > 0
	return status
}

// ClearAllDirtyItems 清空脏队列（全量重建后调用）
func (s *Store) ClearAllDirtyItems() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.db.Update(func(tx *buntdb.Tx) error {
		var keys []string
		s.foreachDirtyItem(tx, func(key string, _ model.DirtyItem) {
			keys = append(keys, key)
		})
		for _, k := range keys {
			if _, err := tx.Delete(k); err != nil && err != buntdb.ErrNotFound {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Warn("failed to clear all dirty items", "err", err)
	}
}

// ClearDirtyByType 清除指定类型的脏记录（模块级重建后调用）
func (s *Store) ClearDirtyByType(docType string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.db.Update(func(tx *buntdb.Tx) error {
		var keys []string
		s.foreachDirtyItem(tx, func(key string, item model.DirtyItem) {
			if item.DocType == docType {
				keys = append(keys, key)
			}
		})
		for _, k := range keys {
			if _, err := tx.Delete(k); err != nil && err != buntdb.ErrNotFound {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Warn("failed to clear dirty items by type", "doc_type", docType, "err", err)
	}
}

// --- 辅助函数 ---

func batchIndex(idx bleve.Index, docs []BleveDoc) error {
	batch := idx.NewBatch()
	for i, doc := range docs {
		batch.Index(doc.ID, doc.Fields)
		if (i+1)%bleveBatchSize == 0 {
			if err := idx.Batch(batch); err != nil {
				return fmt.Errorf("batch index at %d: %w", i, err)
			}
			batch = idx.NewBatch()
		}
	}
	if batch.Size() > 0 {
		if err := idx.Batch(batch); err != nil {
			return fmt.Errorf("batch index final: %w", err)
		}
	}
	return nil
}

