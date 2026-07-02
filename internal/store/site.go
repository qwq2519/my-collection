package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// siteBleveFields 构建站点的 Bleve 索引字段映射
func siteBleveFields(site *model.Site) map[string]interface{} {
	return map[string]interface{}{
		"_type":       "site",
		"title":       site.Title,
		"description": site.Description,
		"domain":      site.Domain,
		"url":         site.URL,
		"domain_text": site.Domain,
		"tags":        site.Tags,
		"updated_at":  site.UpdatedAt,
	}
}

// getSiteTx 在已有事务中读取站点（bookmark.go 等同包文件复用）
func getSiteTx(tx *buntdb.Tx, id string) (*model.Site, error) {
	val, err := tx.Get("site:" + id)
	if err != nil {
		return nil, err
	}
	var site model.Site
	if err := json.Unmarshal([]byte(val), &site); err != nil {
		return nil, fmt.Errorf("unmarshal site: %w", err)
	}
	return &site, nil
}

// setSiteTx 在已有事务中写入站点（bookmark.go 等同包文件复用）
func setSiteTx(tx *buntdb.Tx, site *model.Site) error {
	site.EnsureSlices()
	val, err := json.Marshal(site)
	if err != nil {
		return fmt.Errorf("marshal site: %w", err)
	}
	_, _, err = tx.Set("site:"+site.ID, string(val), nil)
	return err
}

// CreateSite 创建站点，事务内校验域名唯一性。
// BuntDB 写入后同步更新 Bleve 索引。
func (s *Store) CreateSite(req model.CreateSiteReq) (*model.Site, error) {
	now := time.Now()
	site := &model.Site{
		ID:            uuid.New().String(),
		Title:         req.Title,
		URL:           req.URL,
		Domain:        req.Domain,
		Icon:          req.Icon,
		Description:   req.Description,
		Tags:          req.Tags,
		Attachments:   req.Attachments,
		BookmarkCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	site.EnsureSlices()

	err := s.db.Update(func(tx *buntdb.Tx) error {
		pivot, _ := json.Marshal(map[string]string{"domain": req.Domain})
		var exists bool
		tx.AscendEqual("idx:site_domain", string(pivot), func(key, value string) bool {
			exists = true
			return false
		})
		if exists {
			return fmt.Errorf("site already exists for domain %q", req.Domain)
		}
		return setSiteTx(tx, site)
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("site:"+site.ID, siteBleveFields(site))
	slog.Info("site created", "id", site.ID, "domain", site.Domain)
	return site, nil
}

// GetSite 按 ID 查询站点
func (s *Store) GetSite(id string) (*model.Site, error) {
	var site model.Site
	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("site:" + id)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &site)
	})
	if err != nil {
		return nil, err
	}
	return &site, nil
}

// GetSiteByDomain 通过域名查找站点，利用 idx:site_domain 索引。
// 未找到返回 (nil, nil)。
func (s *Store) GetSiteByDomain(domain string) (*model.Site, error) {
	var site *model.Site

	pivot, err := json.Marshal(map[string]string{"domain": domain})
	if err != nil {
		return nil, fmt.Errorf("marshal pivot: %w", err)
	}

	err = s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendEqual("idx:site_domain", string(pivot), func(key, value string) bool {
			var s model.Site
			if err := json.Unmarshal([]byte(value), &s); err == nil {
				site = &s
			}
			return false
		})
	})
	if err != nil {
		return nil, fmt.Errorf("lookup site by domain: %w", err)
	}

	return site, nil
}

// UpdateSite 部分更新站点（仅修改非 nil 字段），更新 updated_at。
// BuntDB 事务提交后同步重建 Bleve 索引。
func (s *Store) UpdateSite(req model.UpdateSiteReq) (*model.Site, error) {
	var site model.Site

	err := s.db.Update(func(tx *buntdb.Tx) error {
		existing, err := getSiteTx(tx, req.ID)
		if err != nil {
			return err
		}
		site = *existing

		if req.Title != nil {
			site.Title = *req.Title
		}
		if req.Description != nil {
			site.Description = *req.Description
		}
		if req.Tags != nil {
			site.Tags = *req.Tags
		}
		if req.Icon != nil {
			site.Icon = *req.Icon
		}
		if req.Attachments != nil {
			site.Attachments = *req.Attachments
		}

		site.UpdatedAt = time.Now()
		return setSiteTx(tx, &site)
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("site:"+site.ID, siteBleveFields(&site))
	return &site, nil
}

// DeleteSite 删除站点。站点下仍有书签时拒绝删除。
// BuntDB 删除后从 Bleve 移除对应文档。
func (s *Store) DeleteSite(id string) error {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		site, err := getSiteTx(tx, id)
		if err != nil {
			return err
		}
		if site.BookmarkCount > 0 {
			return fmt.Errorf("site still has %d bookmarks, delete them first", site.BookmarkCount)
		}
		_, err = tx.Delete("site:" + id)
		return err
	})
	if err != nil {
		return err
	}

	s.DeleteDoc("site:"+id, "site")
	slog.Info("site deleted", "id", id)
	return nil
}
