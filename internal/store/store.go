package store

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/tidwall/buntdb"
)

// Store 数据访问层主结构体，持有 BuntDB 和 IndexManager 实例
type Store struct {
	db  *buntdb.DB
	Idx *IndexManager

	// backupMu 备份协调锁：普通写操作取 RLock（允许并发），备份操作取 Lock（独占）
	backupMu sync.RWMutex

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
	s.Idx = idx

	status := s.GetDirtyIndexStatus()
	if status.HasDirty {
		slog.Warn("dirty index items found", "url", status.URLCount, "note", status.NoteCount, "media", status.MediaCount)
	}

	return s, nil
}

// Close 关闭 BuntDB 和 Bleve
func (s *Store) Close() error {
	var firstErr error

	if s.Idx != nil {
		if err := s.Idx.Close(); err != nil && firstErr == nil {
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
