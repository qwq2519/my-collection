package store

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/tidwall/buntdb"
)

// Store 数据访问层主结构体，持有 BuntDB 和 IndexManager 实例。
//
// 并发策略（单锁）：
//   - mu (RWMutex)：全局操作锁。普通读写操作取 RLock（允许并发），
//     导出备份和模块级索引重建取 Lock（独占，阻塞所有读写）。
//   - IndexManager 自身不持有锁，并发安全由 Store.mu 统一保证。
type Store struct {
	mu sync.RWMutex

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
