package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
)

// gseTokenizer 基于 gse 的 Bleve 分词器
type gseTokenizer struct {
	seg *gse.Segmenter
}

func (t *gseTokenizer) Tokenize(input []byte) analysis.TokenStream {
	text := string(input)
	segments := t.seg.Cut(text, true)

	tokens := make(analysis.TokenStream, 0, len(segments))
	pos := 1
	byteOffset := 0

	for _, seg := range segments {
		start := byteOffset
		end := start + len(seg)
		byteOffset = end

		tokens = append(tokens, &analysis.Token{
			Term:     []byte(seg),
			Start:    start,
			End:      end,
			Position: pos,
			Type:     analysis.Ideographic,
		})
		pos++
	}

	return tokens
}

func gseTokenizerConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.Tokenizer, error) {
	seg, err := gse.New()
	if err != nil {
		return nil, fmt.Errorf("init gse segmenter: %w", err)
	}
	return &gseTokenizer{seg: &seg}, nil
}

func init() {
	registry.RegisterTokenizer(gseTokenizerName, gseTokenizerConstructor)
}

// openBleve 打开已有 Bleve 索引，或新建一个
func openBleve(persistDir string) (bleve.Index, error) {
	indexPath := filepath.Join(persistDir, "search.bleve")

	idx, err := bleve.Open(indexPath)
	if err == nil {
		return idx, nil
	}

	if !os.IsNotExist(err) && err != bleve.Error(1) {
		return nil, fmt.Errorf("open index: %w", err)
	}

	slog.Info("creating new bleve index", "path", indexPath)
	m := buildIndexMapping()
	idx, err = bleve.New(indexPath, m)
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	return idx, nil
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
// 失败时记录到脏队列，不阻塞主流程。
func (s *Store) IndexDoc(id string, docType string, fields map[string]interface{}) error {
	if err := s.index.Index(id, fields); err != nil {
		slog.Warn("bleve index failed", "id", id, "err", err)
		s.addDirtyItem(id, docType)
		return err
	}
	return nil
}

// DeleteDoc 从 Bleve 索引中删除文档。失败时记录到脏队列。
func (s *Store) DeleteDoc(id string, docType string) error {
	if err := s.index.Delete(id); err != nil {
		slog.Warn("bleve delete failed", "id", id, "err", err)
		s.addDirtyItem(id, docType)
		return err
	}
	return nil
}

// RebuildIndex 全量重建 Bleve 索引（设置页"重建所有索引"）
func (s *Store) RebuildIndex(docs []BleveDoc) error {
	s.backupMu.Lock()
	defer s.backupMu.Unlock()

	indexPath := filepath.Join(s.persistDir, "search.bleve")

	if s.index != nil {
		s.index.Close()
	}

	os.RemoveAll(indexPath)

	m := buildIndexMapping()
	idx, err := bleve.New(indexPath, m)
	if err != nil {
		return fmt.Errorf("create new index: %w", err)
	}

	if err := batchIndex(idx, docs); err != nil {
		idx.Close()
		return err
	}

	s.index = idx
	s.ClearAllDirtyItems()

	slog.Info("bleve index fully rebuilt", "docs", len(docs))
	return nil
}

// RebuildDocs 局部重建：删除指定 ID 的旧文档，写入新文档。
// 用于按模块或按文件夹粒度重建。
func (s *Store) RebuildDocs(deleteIDs []string, newDocs []BleveDoc) error {
	s.backupMu.RLock()
	defer s.backupMu.RUnlock()

	batch := s.index.NewBatch()
	for _, id := range deleteIDs {
		batch.Delete(id)
	}
	for _, doc := range newDocs {
		batch.Index(doc.ID, doc.Fields)
	}

	if err := s.index.Batch(batch); err != nil {
		return fmt.Errorf("rebuild docs batch: %w", err)
	}

	// 从脏队列中移除已重建的项
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
	s.db.Update(func(tx *buntdb.Tx) error {
		val := fmt.Sprintf(`{"doc_id":"%s","doc_type":"%s","failed_at":"%s"}`,
			docID, docType, time.Now().Format(time.RFC3339))
		tx.Set("dirty:"+docID, val, nil)
		return nil
	})
}

// removeDirtyItem 从脏队列移除一条
func (s *Store) removeDirtyItem(docID string) {
	s.db.Update(func(tx *buntdb.Tx) error {
		tx.Delete("dirty:" + docID)
		return nil
	})
}

// GetDirtyItems 获取脏队列中所有条目
func (s *Store) GetDirtyItems() []model.DirtyItem {
	var items []model.DirtyItem
	s.db.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys("dirty:*", func(key, value string) bool {
			var item model.DirtyItem
			if err := json.Unmarshal([]byte(value), &item); err == nil {
				items = append(items, item)
			}
			return true
		})
		return nil
	})
	return items
}

// GetDirtyIndexStatus 获取索引状态摘要（按模块统计脏文档数）
func (s *Store) GetDirtyIndexStatus() model.DirtyIndexStatus {
	status := model.DirtyIndexStatus{}
	s.db.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys("dirty:*", func(key, value string) bool {
			var item model.DirtyItem
			if err := json.Unmarshal([]byte(value), &item); err == nil {
				switch item.DocType {
				case "site", "bookmark":
					status.URLCount++
				case "note":
					status.NoteCount++
				case "media":
					status.MediaCount++
				}
			}
			return true
		})
		return nil
	})
	status.HasDirty = (status.URLCount + status.NoteCount + status.MediaCount) > 0
	return status
}

// ClearAllDirtyItems 清空脏队列（全量重建后调用）
func (s *Store) ClearAllDirtyItems() {
	s.db.Update(func(tx *buntdb.Tx) error {
		var keys []string
		tx.AscendKeys("dirty:*", func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
		for _, k := range keys {
			tx.Delete(k)
		}
		return nil
	})
}

// ClearDirtyByType 清除指定类型的脏记录（模块级重建后调用）
func (s *Store) ClearDirtyByType(docType string) {
	s.db.Update(func(tx *buntdb.Tx) error {
		var keys []string
		tx.AscendKeys("dirty:*", func(key, value string) bool {
			var item model.DirtyItem
			if err := json.Unmarshal([]byte(value), &item); err == nil && item.DocType == docType {
				keys = append(keys, key)
			}
			return true
		})
		for _, k := range keys {
			tx.Delete(k)
		}
		return nil
	})
}

// --- 辅助函数 ---

func batchIndex(idx bleve.Index, docs []BleveDoc) error {
	batch := idx.NewBatch()
	for i, doc := range docs {
		batch.Index(doc.ID, doc.Fields)
		if (i+1)%500 == 0 {
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

// Index 返回底层 Bleve 索引实例（供 service 层执行搜索查询）
func (s *Store) Index() bleve.Index {
	return s.index
}
