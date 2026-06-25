package store

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/tidwall/buntdb"
)

// Store 数据访问层主结构体，持有 BuntDB 和 IndexManager 实例。
//
// 并发策略（双锁分离）：
//   - backupMu (RWMutex)：备份协调锁。写操作取 RLock（允许并发写入），
//     备份/全量重建取 Lock（独占，等待所有写入完成后执行）。
//     读操作不取 backupMu，备份期间搜索不受影响。
//   - IndexManager.mu (RWMutex)：索引状态锁。索引读写取 RLock，
//     重建/关闭取 Lock。独立于 backupMu，保证索引操作的并发安全。
type Store struct {
	backupMu sync.RWMutex

	db  *buntdb.DB
	idx *IndexManager

	persistDir string
}

// New 初始化 Store：打开 BuntDB、注册自定义索引、打开或创建 Bleve 索引
func New(persistDir string) (*Store, error) {
	s := &Store{persistDir: persistDir}

	db, err := openBuntDB(persistDir)
	if err != nil {
		return nil, fmt.Errorf("open buntdb: %w", err)
	}
	s.db = db

	if err := s.registerIndexes(); err != nil {
		db.Close()
		return nil, fmt.Errorf("register indexes: %w", err)
	}

	idx, err := NewIndexManager(persistDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open bleve: %w", err)
	}
	s.idx = idx

	if s.HasDirtyItems() {
		slog.Warn("dirty index detected, rebuild recommended")
	}

	return s, nil
}

// Close 关闭 BuntDB 和 Bleve
func (s *Store) Close() error {
	var firstErr error

	if s.idx != nil {
		if err := s.idx.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// PersistDir 返回持久化目录路径
func (s *Store) PersistDir() string {
	return s.persistDir
}
