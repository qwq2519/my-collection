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
	IndexClosed     IndexState = iota // 已关闭或未初始化
	IndexOpen                         // 正常可用
	IndexRebuilding                   // 重建中，读写暂不可用
	IndexError                        // 打开/创建失败
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
// 内部持有 RWMutex：普通读写取 RLock（允许并发），重建取 Lock（独占）。
type IndexManager struct {
	mu sync.RWMutex

	index      bleve.Index
	state      IndexState
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
	if err := initGse(); err != nil {
		return nil, err
	}

	m := &IndexManager{
		persistDir: persistDir,
		state:      IndexClosed,
	}
	idx, err := openBleve(persistDir)
	if err != nil {
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.ensureOpen(); err != nil {
		return err
	}
	return m.index.Index(id, fields)
}

// DeleteDoc 从索引删除文档
func (m *IndexManager) DeleteDoc(id string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.ensureOpen(); err != nil {
		return err
	}
	return m.index.Delete(id)
}

// Search 执行搜索查询
func (m *IndexManager) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.ensureOpen(); err != nil {
		return nil, err
	}
	return m.index.Search(req)
}

// rebuild 全量重建索引：备份旧索引 → 创建新索引 → 批量写入 → 删除备份。
// 失败时自动恢复旧索引，避免搜索功能完全不可用。
//
// NOTE: 目前未启用全量重建，保留此函数供后续实现全量重建功能时使用。
//
//nolint:unused
func (m *IndexManager) rebuild(docs []BleveDoc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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
		}
		return fmt.Errorf("create new index: %w", err)
	}

	if err := batchIndex(idx, docs); err != nil {
		if closeErr := idx.Close(); closeErr != nil {
			slog.Warn("close failed index after batch error", "err", closeErr)
		}
		restoreBackup()
		if m.state != IndexOpen {
			m.state = IndexError
		}
		return err
	}

	m.index = idx
	m.state = IndexOpen

	if hasBackup {
		_ = os.RemoveAll(backupPath)
	}

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
