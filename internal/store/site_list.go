package store

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
	"github.com/tidwall/buntdb"
)

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
