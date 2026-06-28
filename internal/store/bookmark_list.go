package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"collections/internal/model"

	"github.com/tidwall/buntdb"
)

// ListBookmarks 按站点分页查询书签列表，走 BuntDB idx:bm_site 索引。
// TODO: 当前全量加载站点下所有书签到内存后排序再分页，数据量大时浪费内存。
// 优化方案：建 site_id+updated_at 复合索引，让 BuntDB 按序扫描直接跳过 + 截断。
func (s *Store) ListBookmarks(req model.BookmarkListReq) (*model.BookmarkListResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	var bookmarks []model.Bookmark

	pivot, _ := json.Marshal(map[string]string{"site_id": req.SiteID})
	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendEqual("idx:bm_site", string(pivot), func(key, value string) bool {
			var bm model.Bookmark
			if err := json.Unmarshal([]byte(value), &bm); err != nil {
				slog.Warn("skip corrupted bookmark", "key", key, "err", err)
				return true
			}
			bookmarks = append(bookmarks, bm)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list bookmarks: %w", err)
	}

	sort.Slice(bookmarks, func(i, j int) bool {
		return bookmarks[i].UpdatedAt.After(bookmarks[j].UpdatedAt)
	})

	total := len(bookmarks)
	start := (req.Page - 1) * req.PageSize
	if start >= total {
		return &model.BookmarkListResult{Items: []model.Bookmark{}, Total: total, HasMore: false}, nil
	}
	end := start + req.PageSize
	if end > total {
		end = total
	}
	return &model.BookmarkListResult{
		Items:   bookmarks[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}
