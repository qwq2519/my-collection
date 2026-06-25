package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
)

// IndexState 索引生命周期状态
type IndexState int

const (
	IndexClosed     IndexState = iota // 已关闭或未初始化
	IndexOpen                        // 正常可用
	IndexRebuilding                  // 重建中，读写暂不可用
	IndexError                       // 打开/创建失败
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

// IndexManager 管理 Bleve 索引的生命周期和状态。
// 自身不持有锁，并发安全由 Store.mu 统一保证。
type IndexManager struct {
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

// State 返回当前索引状态
func (m *IndexManager) State() IndexState {
	return m.state
}

func (m *IndexManager) ensureOpen() error {
	if m.state != IndexOpen {
		return fmt.Errorf("bleve index unavailable (state: %s)", m.state)
	}
	return nil
}

// IndexDoc 索引单个文档
func (m *IndexManager) IndexDoc(id string, fields map[string]interface{}) error {
	if err := m.ensureOpen(); err != nil {
		return err
	}
	return m.index.Index(id, fields)
}

// DeleteDoc 从索引删除文档
func (m *IndexManager) DeleteDoc(id string) error {
	if err := m.ensureOpen(); err != nil {
		return err
	}
	return m.index.Delete(id)
}

// Search 执行搜索查询
func (m *IndexManager) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	if err := m.ensureOpen(); err != nil {
		return nil, err
	}
	return m.index.Search(req)
}

// NewBatch 创建新的批量操作
func (m *IndexManager) NewBatch() (*bleve.Batch, error) {
	if err := m.ensureOpen(); err != nil {
		return nil, err
	}
	return m.index.NewBatch(), nil
}

// ExecuteBatch 执行批量操作
func (m *IndexManager) ExecuteBatch(batch *bleve.Batch) error {
	if err := m.ensureOpen(); err != nil {
		return err
	}
	return m.index.Batch(batch)
}

// Rebuild 全量重建索引：备份旧索引 → 创建新索引 → 批量写入 → 删除备份。
// 失败时自动恢复旧索引，避免搜索功能完全不可用。
// 调用方须持有 Store.mu.Lock。
func (m *IndexManager) Rebuild(docs []BleveDoc) error {
	m.state = IndexRebuilding
	indexPath := filepath.Join(m.persistDir, "search.bleve")
	backupPath := indexPath + ".bak"

	if m.index != nil {
		if err := m.index.Close(); err != nil {
			slog.Warn("close old index before rebuild", "err", err)
		}
		m.index = nil
	}

	_ = os.RemoveAll(backupPath)
	hasBackup := false
	if _, err := os.Stat(indexPath); err == nil {
		if err := os.Rename(indexPath, backupPath); err != nil {
			m.state = IndexError
			m.lastErr = err
			return fmt.Errorf("backup old index: %w", err)
		}
		hasBackup = true
	}

	restoreBackup := func() {
		if !hasBackup {
			return
		}
		_ = os.RemoveAll(indexPath)
		if err := os.Rename(backupPath, indexPath); err != nil {
			slog.Error("failed to restore index backup", "err", err)
			return
		}
		if idx, err := bleve.Open(indexPath); err == nil {
			m.index = idx
			m.state = IndexOpen
			m.lastErr = nil
			slog.Info("restored old index after rebuild failure")
		} else {
			slog.Error("failed to reopen restored index", "err", err)
		}
	}

	im := buildIndexMapping()
	idx, err := bleve.New(indexPath, im)
	if err != nil {
		restoreBackup()
		if m.state != IndexOpen {
			m.state = IndexError
			m.lastErr = err
		}
		return fmt.Errorf("create new index: %w", err)
	}

	if err := batchIndex(idx, docs); err != nil {
		idx.Close()
		restoreBackup()
		if m.state != IndexOpen {
			m.state = IndexError
			m.lastErr = err
		}
		return err
	}

	m.index = idx
	m.state = IndexOpen
	m.lastErr = nil

	if hasBackup {
		_ = os.RemoveAll(backupPath)
	}

	slog.Info("bleve index fully rebuilt", "docs", len(docs))
	return nil
}

// Close 关闭索引
func (m *IndexManager) Close() error {
	if m.index != nil {
		err := m.index.Close()
		m.index = nil
		m.state = IndexClosed
		return err
	}
	m.state = IndexClosed
	return nil
}
