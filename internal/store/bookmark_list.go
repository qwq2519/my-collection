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
