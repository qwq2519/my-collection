package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/blevesearch/bleve/v2"
)

// IndexState 索引生命周期状态
type IndexState int

const (
	IndexClosed    IndexState = iota // 已关闭或未初始化
	IndexOpen                       // 正常可用
	IndexRebuilding                 // 重建中，读写暂不可用
	IndexError                      // 打开/创建失败
)

func (s IndexState) String() string {
	switch s {
	case IndexClosed:
		return "closed"
	case IndexOpen:
		return "open"
	case IndexRebuilding:
		return "rebuilding"
	case IndexError:
		return "error"
	default:
		return "unknown"
	}
}

// IndexManager 管理 Bleve 索引的生命周期，提供线程安全的访问。
// 外部不直接持有 bleve.Index 引用，所有操作通过 IndexManager 间接完成，
// 避免重建期间持有过期引用导致的竞态问题。
type IndexManager struct {
	mu         sync.RWMutex
	index      bleve.Index
	state      IndexState
	lastErr    error
	persistDir string
}

// openBleve 打开已有 Bleve 索引，或新建一个
func openBleve(persistDir string) (bleve.Index, error) {
	indexPath := filepath.Join(persistDir, "search.bleve")

	idx, err := bleve.Open(indexPath)
	if err == nil {
		return idx, nil
	}

	if !os.IsNotExist(err) && err != bleve.ErrorIndexPathDoesNotExist {
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

// NewIndexManager 打开或创建 Bleve 索引
func NewIndexManager(persistDir string) (*IndexManager, error) {
	m := &IndexManager{
		persistDir: persistDir,
		state:      IndexClosed,
	}
	idx, err := openBleve(persistDir)
	if err != nil {
		m.state = IndexError
		m.lastErr = err
		return nil, err
	}
	m.index = idx
	m.state = IndexOpen
	return m, nil
}

// State 返回当前索引状态（线程安全）
func (m *IndexManager) State() IndexState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// IndexDoc 索引单个文档。索引不可用时返回 error。
func (m *IndexManager) IndexDoc(id string, fields map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != IndexOpen {
		return fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return m.index.Index(id, fields)
}

// DeleteDoc 从索引删除文档。索引不可用时返回 error。
func (m *IndexManager) DeleteDoc(id string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != IndexOpen {
		return fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return m.index.Delete(id)
}

// Search 执行搜索查询。索引不可用时返回 error。
func (m *IndexManager) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != IndexOpen {
		return nil, fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return m.index.Search(req)
}

// NewBatch 创建新的批量操作。索引不可用时返回 error。
func (m *IndexManager) NewBatch() (*bleve.Batch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != IndexOpen {
		return nil, fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return m.index.NewBatch(), nil
}

// ExecuteBatch 执行批量操作。索引不可用时返回 error。
func (m *IndexManager) ExecuteBatch(batch *bleve.Batch) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state != IndexOpen {
		return fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return m.index.Batch(batch)
}

// Rebuild 全量重建索引：关闭旧索引 → 删除文件 → 创建新索引 → 批量写入。
// 重建期间持有写锁，所有并发的 IndexDoc/Search 等操作会阻塞等待。
func (m *IndexManager) Rebuild(docs []BleveDoc) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = IndexRebuilding
	indexPath := filepath.Join(m.persistDir, "search.bleve")

	if m.index != nil {
		m.index.Close()
		m.index = nil
	}

	if err := os.RemoveAll(indexPath); err != nil {
		m.state = IndexError
		m.lastErr = err
		return fmt.Errorf("remove old index: %w", err)
	}

	im := buildIndexMapping()
	idx, err := bleve.New(indexPath, im)
	if err != nil {
		m.state = IndexError
		m.lastErr = err
		return fmt.Errorf("create new index: %w", err)
	}

	if err := batchIndex(idx, docs); err != nil {
		idx.Close()
		m.state = IndexError
		m.lastErr = err
		return err
	}

	m.index = idx
	m.state = IndexOpen
	m.lastErr = nil

	slog.Info("bleve index fully rebuilt", "docs", len(docs))
	return nil
}

// Close 关闭索引
func (m *IndexManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.index != nil {
		err := m.index.Close()
		m.index = nil
		m.state = IndexClosed
		return err
	}
	m.state = IndexClosed
	return nil
}
