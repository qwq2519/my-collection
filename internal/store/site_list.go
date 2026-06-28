package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
	"github.com/tidwall/buntdb"
)

// ListSites 分页查询站点列表，按 updated_at 降序。纯列表，不含搜索。
func (s *Store) ListSites(req model.SiteListReq) (*model.SiteListResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	var sites []model.Site
	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend("idx:site_updated", func(key, value string) bool {
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

	total := len(sites)
	start := (req.Page - 1) * req.PageSize
	if start >= total {
		return &model.SiteListResult{Items: []model.Site{}, Total: total, HasMore: false}, nil
	}
	end := start + req.PageSize
	if end > total {
		end = total
	}
	return &model.SiteListResult{
		Items:   sites[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}

// SearchURL 同时搜索站点和书签，将书签结果按所属站点分组返回。
// 站点自身命中或其下书签命中均会出现在结果中。
func (s *Store) SearchURL(req model.SearchURLReq) (*model.SearchURLResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.Search == "" && len(req.Tags) == 0 {
		return nil, fmt.Errorf("请提供搜索关键词或标签")
	}

	siteTypeQ := bleve.NewTermQuery("site")
	siteTypeQ.SetField("_type")
	bmTypeQ := bleve.NewTermQuery("bookmark")
	bmTypeQ.SetField("_type")
	typeQ := bleve.NewDisjunctionQuery(siteTypeQ, bmTypeQ)

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
		for _, tag := range req.Tags {
			tagQ := bleve.NewTermQuery(tag)
			tagQ.SetField("tags")
			conjunction.AddQuery(tagQ)
		}
	}

	searchReq := bleve.NewSearchRequest(conjunction)
	// TODO: 硬编码 10000 全量取出后内存分组排序，数据量大时有性能隐患。
	// 优化方案：结合 Bleve 分页 + 按需加载，或引入游标式搜索。
	searchReq.Size = 10000
	searchReq.Fields = []string{}

	result, err := s.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search url: %w", err)
	}

	var items []model.SiteWithBookmarks
	err = s.db.View(func(tx *buntdb.Tx) error {
		siteHitSet := make(map[string]bool)
		bmBySite := make(map[string][]string)

		for _, hit := range result.Hits {
			if strings.HasPrefix(hit.ID, "site:") {
				siteID := strings.TrimPrefix(hit.ID, "site:")
				siteHitSet[siteID] = true
			} else if strings.HasPrefix(hit.ID, "bm:") {
				bmID := strings.TrimPrefix(hit.ID, "bm:")
				bm, err := getBookmarkTx(tx, bmID)
				if err != nil {
					continue
				}
				bmBySite[bm.SiteID] = append(bmBySite[bm.SiteID], bmID)
			}
		}

		allSiteIDs := make(map[string]bool)
		for id := range siteHitSet {
			allSiteIDs[id] = true
		}
		for siteID := range bmBySite {
			allSiteIDs[siteID] = true
		}

		for siteID := range allSiteIDs {
			site, err := getSiteTx(tx, siteID)
			if err != nil {
				slog.Warn("skip missing site in search", "site_id", siteID)
				continue
			}
			item := model.SiteWithBookmarks{
				Site:      *site,
				Bookmarks: make([]model.Bookmark, 0),
			}
			if bmIDs, ok := bmBySite[siteID]; ok {
				for _, bmID := range bmIDs {
					bm, err := getBookmarkTx(tx, bmID)
					if err != nil {
						continue
					}
					item.Bookmarks = append(item.Bookmarks, *bm)
				}
			}
			items = append(items, item)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search url: group and fetch: %w", err)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Site.UpdatedAt.After(items[j].Site.UpdatedAt)
	})

	total := len(items)
	start := (req.Page - 1) * req.PageSize
	if start >= total {
		return &model.SearchURLResult{
			Items: []model.SiteWithBookmarks{}, Total: total, HasMore: false,
		}, nil
	}
	end := start + req.PageSize
	if end > total {
		end = total
	}

	return &model.SearchURLResult{
		Items:   items[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}
