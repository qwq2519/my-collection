package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
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

var (
	gseOnce    sync.Once
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

	fields["_type"] = docType

	if err := s.idx.IndexDoc(id, fields); err != nil {
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

// Search 执行 Bleve 搜索查询，持 mu.RLock 保证与重建操作互斥。
func (s *Store) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.idx.Search(req)
}

// RebuildIndex 全量重建 Bleve 索引（设置页"重建所有索引"）。
// 取 mu.Lock 独占，阻塞所有并发的读写操作。
func (s *Store) RebuildIndex(docs []BleveDoc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.idx.Rebuild(docs); err != nil {
		return err
	}
	s.ClearAllDirtyItems()
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
		if err := json.Unmarshal([]byte(value), &item); err == nil {
			fn(key, item)
		}
		return true
	})
}

// HasDirtyItems 判断是否存在脏记录（前缀扫描 dirty:*）
func (s *Store) HasDirtyItems() bool {
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

