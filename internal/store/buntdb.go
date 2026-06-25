package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/buntdb"
)

// openBuntDB 打开或创建 BuntDB 数据库文件
func openBuntDB(persistDir string) (*buntdb.DB, error) {
	if err := os.MkdirAll(persistDir, 0755); err != nil {
		return nil, fmt.Errorf("create persist dir: %w", err)
	}

	dbPath := filepath.Join(persistDir, "main.db")
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dbPath, err)
	}

	return db, nil
}

// registerIndexes 注册 BuntDB 自定义索引，用于加速非主键查询
func (s *Store) registerIndexes() error {
	indexes := []struct {
		name    string
		pattern string
		path    string
	}{
		// 按域名查找站点（添加书签时自动归组）
		{"idx:site_domain", "site:*", "domain"},
		// 按站点列出书签
		{"idx:bm_site", "bm:*", "site_id"},
	}

	for _, idx := range indexes {
		err := s.db.CreateIndex(idx.name, idx.pattern, buntdb.IndexJSON(idx.path))
		if err != nil && err != buntdb.ErrIndexExists {
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
	}

	return nil
}
