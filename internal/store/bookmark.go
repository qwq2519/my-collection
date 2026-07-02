package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// bmBleveFields 构建书签的 Bleve 索引字段映射
func bmBleveFields(bm *model.Bookmark) map[string]interface{} {
	return map[string]interface{}{
		"_type":       "bookmark",
		"title":       bm.Title,
		"description": bm.Description,
		"domain":      bm.Domain,
		"domain_text": bm.Domain,
		"tags":        bm.Tags,
		"url":         bm.URL,
		"updated_at":  bm.UpdatedAt,
	}
}

// getBookmarkTx 在已有事务中按 ID 读取书签（key: bm:{id}）
func getBookmarkTx(tx *buntdb.Tx, id string) (*model.Bookmark, error) {
	val, err := tx.Get("bm:" + id)
	if err == buntdb.ErrNotFound {
		return nil, fmt.Errorf("bookmark not found")
	}
	if err != nil {
		return nil, err
	}
	var bm model.Bookmark
	if err := json.Unmarshal([]byte(val), &bm); err != nil {
		return nil, fmt.Errorf("unmarshal bookmark: %w", err)
	}
	return &bm, nil
}

// setBookmarkTx 在已有事务中写入书签（key: bm:{id}）
func setBookmarkTx(tx *buntdb.Tx, bm *model.Bookmark) error {
	bm.EnsureSlices()
	val, err := json.Marshal(bm)
	if err != nil {
		return fmt.Errorf("marshal bookmark: %w", err)
	}
	_, _, err = tx.Set("bm:"+bm.ID, string(val), nil)
	return err
}

// checkBookmarkDupTx 通过 idx:bm_site 取出同站点书签，归一化比对 URL 判断重复。
// TODO: 当前为 O(N) 全扫描——遍历站点下所有书签逐条归一化比对。
// 优化方案：在 Bookmark 上持久化 NormalizedURL 字段并建 BuntDB 索引，改为 O(1) 查找。
func checkBookmarkDupTx(tx *buntdb.Tx, siteID, normalizedURL string) error {
	pivot, _ := json.Marshal(map[string]string{"site_id": siteID})
	var dup bool
	tx.AscendEqual("idx:bm_site", string(pivot), func(key, value string) bool {
		var bm model.Bookmark
		if err := json.Unmarshal([]byte(value), &bm); err != nil {
			return true
		}
		if norm, err := util.NormalizeURL(bm.URL); err == nil && norm == normalizedURL {
			dup = true
			return false
		}
		return true
	})
	if dup {
		return fmt.Errorf("URL already bookmarked")
	}
	return nil
}

// CreateBookmark 创建书签。同一事务内完成：URL 去重、写入书签、
// 递增站点 bookmark_count、更新站点 updated_at。
// BuntDB 提交后同步更新书签和站点的 Bleve 索引。
func (s *Store) CreateBookmark(req model.CreateBookmarkReq) (*model.Bookmark, error) {
	normalizedURL, err := util.NormalizeURL(req.URL)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	bm := &model.Bookmark{
		ID:          uuid.New().String(),
		URL:         req.URL,
		Domain:      req.Domain,
		SiteID:      req.SiteID,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
		Status:      model.BookmarkStatusAlive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	bm.EnsureSlices()

	var site model.Site
	err = s.db.Update(func(tx *buntdb.Tx) error {
		siteObj, err := getSiteTx(tx, req.SiteID)
		if err != nil {
			return err
		}

		if err := checkBookmarkDupTx(tx, req.SiteID, normalizedURL); err != nil {
			return err
		}

		if err := setBookmarkTx(tx, bm); err != nil {
			return err
		}

		siteObj.BookmarkCount++
		siteObj.UpdatedAt = now
		site = *siteObj
		return setSiteTx(tx, siteObj)
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("bm:"+bm.ID, bmBleveFields(bm))
	s.IndexDoc("site:"+site.ID, siteBleveFields(&site))
	slog.Info("bookmark created", "id", bm.ID, "url", bm.URL, "site_id", bm.SiteID)
	return bm, nil
}

// GetBookmark 按 ID 查询书签
func (s *Store) GetBookmark(id string) (*model.Bookmark, error) {
	var bm model.Bookmark
	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("bm:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("bookmark not found")
		}
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &bm)
	})
	if err != nil {
		return nil, err
	}
	return &bm, nil
}

// UpdateBookmark 部分更新书签（URL/domain/site_id 不可修改），
// 同事务内更新所属站点的 updated_at。
// BuntDB 提交后同步重建书签和站点的 Bleve 索引。
func (s *Store) UpdateBookmark(req model.UpdateBookmarkReq) (*model.Bookmark, error) {
	var bm model.Bookmark
	var site model.Site

	err := s.db.Update(func(tx *buntdb.Tx) error {
		existing, err := getBookmarkTx(tx, req.ID)
		if err != nil {
			return err
		}
		bm = *existing

		if req.Title != nil {
			bm.Title = *req.Title
		}
		if req.Description != nil {
			bm.Description = *req.Description
		}
		if req.Tags != nil {
			bm.Tags = *req.Tags
		}
		if req.Attachments != nil {
			bm.Attachments = *req.Attachments
		}

		bm.UpdatedAt = time.Now()
		if err := setBookmarkTx(tx, &bm); err != nil {
			return err
		}

		siteObj, err := getSiteTx(tx, bm.SiteID)
		if err != nil {
			return err
		}
		siteObj.UpdatedAt = bm.UpdatedAt
		site = *siteObj
		return setSiteTx(tx, siteObj)
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("bm:"+bm.ID, bmBleveFields(&bm))
	s.IndexDoc("site:"+site.ID, siteBleveFields(&site))
	return &bm, nil
}

// DeleteBookmark 删除书签，同事务内递减站点 bookmark_count 并更新 updated_at。
// BuntDB 提交后从 Bleve 删除书签文档并重建站点索引。
func (s *Store) DeleteBookmark(id string) error {
	var site model.Site

	err := s.db.Update(func(tx *buntdb.Tx) error {
		bm, err := getBookmarkTx(tx, id)
		if err != nil {
			return err
		}

		if _, err = tx.Delete("bm:" + id); err != nil {
			return err
		}

		siteObj, err := getSiteTx(tx, bm.SiteID)
		if err != nil {
			return err
		}
		siteObj.BookmarkCount--
		if siteObj.BookmarkCount < 0 {
			slog.Warn("bookmark_count went negative, clamped to 0", "site_id", bm.SiteID, "raw", siteObj.BookmarkCount)
			siteObj.BookmarkCount = 0
		}
		siteObj.UpdatedAt = time.Now()
		site = *siteObj
		return setSiteTx(tx, siteObj)
	})
	if err != nil {
		return err
	}

	s.DeleteDoc("bm:"+id, "bookmark")
	s.IndexDoc("site:"+site.ID, siteBleveFields(&site))
	slog.Info("bookmark deleted", "id", id)
	return nil
}

// BatchDeleteBookmarks 批量删除同一站点下的书签，更新该站点的 bookmark_count。
// BuntDB 提交后逐条从 Bleve 删除书签文档，并在有实际删除时重建站点索引。
func (s *Store) BatchDeleteBookmarks(siteID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	deleted := 0
	var site model.Site

	err := s.db.Update(func(tx *buntdb.Tx) error {
		for _, id := range ids {
			if _, err := tx.Delete("bm:" + id); err != nil {
				if err == buntdb.ErrNotFound {
					slog.Warn("skip missing bookmark in batch delete", "id", id)
					continue
				}
				return fmt.Errorf("delete bookmark %s: %w", id, err)
			}
			deleted++
		}

		if deleted == 0 {
			return nil
		}
		siteObj, err := getSiteTx(tx, siteID)
		if err != nil {
			return fmt.Errorf("get site %s: %w", siteID, err)
		}
		siteObj.BookmarkCount -= deleted
		if siteObj.BookmarkCount < 0 {
			slog.Warn("bookmark_count went negative, clamped to 0", "site_id", siteID, "raw", siteObj.BookmarkCount)
			siteObj.BookmarkCount = 0
		}
		siteObj.UpdatedAt = time.Now()
		site = *siteObj
		return setSiteTx(tx, siteObj)
	})
	if err != nil {
		return err
	}

	for _, id := range ids {
		s.DeleteDoc("bm:"+id, "bookmark")
	}
	if deleted > 0 {
		s.IndexDoc("site:"+site.ID, siteBleveFields(&site))
	}

	slog.Info("bookmarks batch deleted", "count", deleted, "site_id", siteID)
	return nil
}
