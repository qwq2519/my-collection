package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

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
			return fmt.Errorf("域名 %q 对应的站点已存在", req.Domain)
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

// UpdateSite 部分更新站点（仅修改非 nil 字段），更新 updated_at 并重建索引。
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
		if req.URL != nil {
			site.URL = *req.URL
		}
		if req.Domain != nil && *req.Domain != site.Domain {
			pivot, _ := json.Marshal(map[string]string{"domain": *req.Domain})
			var exists bool
			tx.AscendEqual("idx:site_domain", string(pivot), func(key, value string) bool {
				exists = true
				return false
			})
			if exists {
				return fmt.Errorf("域名 %q 对应的站点已存在", *req.Domain)
			}
			site.Domain = *req.Domain
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
func (s *Store) DeleteSite(id string) error {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		site, err := getSiteTx(tx, id)
		if err != nil {
			return err
		}
		if site.BookmarkCount > 0 {
			return fmt.Errorf("站点下仍有 %d 条书签，请先删除所有书签", site.BookmarkCount)
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

// ListSites 分页查询站点列表。无搜索/标签时走 BuntDB 全量扫描，否则走 Bleve。
func (s *Store) ListSites(req model.SiteListReq) (*model.SiteListResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	if req.Search == "" && len(req.Tags) == 0 {
		return s.listSitesFromDB(req)
	}
	return s.listSitesFromBleve(req)
}

func (s *Store) listSitesFromDB(req model.SiteListReq) (*model.SiteListResult, error) {
	// TODO: 页满后仍继续遍历所有记录以统计 total，数据量增大后考虑维护独立 count key
	skip := (req.Page - 1) * req.PageSize
	total := 0
	sites := make([]model.Site, 0, req.PageSize)

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend("idx:site_updated", func(key, value string) bool {
			total++
			if total <= skip {
				return true
			}
			if len(sites) >= req.PageSize {
				return true
			}
			var site model.Site
			if err := json.Unmarshal([]byte(value), &site); err != nil {
				slog.Warn("skip corrupted site", "key", key, "err", err)
				return true
			}
			sites = append(sites, site)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}

	return &model.SiteListResult{
		Items:   sites,
		Total:   total,
		HasMore: skip+len(sites) < total,
	}, nil
}

func (s *Store) listSitesFromBleve(req model.SiteListReq) (*model.SiteListResult, error) {
	typeQ := bleve.NewTermQuery("site")
	typeQ.SetField("_type")
	conjunction := bleve.NewConjunctionQuery(typeQ)

	if req.Search != "" {
		titleQ := bleve.NewMatchQuery(req.Search)
		titleQ.SetField("title")
		descQ := bleve.NewMatchQuery(req.Search)
		descQ.SetField("description")
		domainQ := bleve.NewMatchQuery(req.Search)
		domainQ.SetField("domain_text")
		conjunction.AddQuery(bleve.NewDisjunctionQuery(titleQ, descQ, domainQ))
	}

	if len(req.Tags) > 0 {
		tagOr := bleve.NewDisjunctionQuery()
		for _, tag := range req.Tags {
			tagQ := bleve.NewTermQuery(tag)
			tagQ.SetField("tags")
			tagOr.AddQuery(tagQ)
		}
		conjunction.AddQuery(tagOr)
	}

	searchReq := bleve.NewSearchRequest(conjunction)
	searchReq.SortBy([]string{"-updated_at"})
	searchReq.From = (req.Page - 1) * req.PageSize
	searchReq.Size = req.PageSize
	searchReq.Fields = []string{}

	result, err := s.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search sites: %w", err)
	}

	sites := make([]model.Site, 0, len(result.Hits))
	err = s.db.View(func(tx *buntdb.Tx) error {
		for _, hit := range result.Hits {
			val, err := tx.Get(hit.ID)
			if err != nil {
				slog.Warn("site in index but not in db", "id", hit.ID, "err", err)
				continue
			}
			var site model.Site
			if err := json.Unmarshal([]byte(val), &site); err != nil {
				slog.Warn("corrupted site data", "id", hit.ID, "err", err)
				continue
			}
			sites = append(sites, site)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetch sites: %w", err)
	}

	total := int(result.Total)
	return &model.SiteListResult{
		Items:   sites,
		Total:   total,
		HasMore: (req.Page-1)*req.PageSize+len(sites) < total,
	}, nil
}
