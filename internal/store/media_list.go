package store

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
)

// ListMediaFiles 分页查询媒体文件。无搜索/标签时从 media_meta.json 读取，
// 有搜索或标签时走 Bleve 查询。
func (s *Store) ListMediaFiles(req model.MediaListReq) (*model.MediaListResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 40
	}

	if req.Search == "" && len(req.Tags) == 0 {
		return s.listMediaFromDisk(req)
	}
	return s.listMediaFromBleve(req)
}

// listMediaFromDisk 从 media_meta.json 加载全量文件，内存排序后分页返回
func (s *Store) listMediaFromDisk(req model.MediaListReq) (*model.MediaListResult, error) {
	var folderIDs []string
	if req.FolderID != "" {
		folderIDs = []string{req.FolderID}
	} else {
		folders, err := s.ListFolders()
		if err != nil {
			return nil, err
		}
		for _, f := range folders {
			folderIDs = append(folderIDs, f.ID)
		}
	}

	var items []model.MediaFileItem
	for _, fid := range folderIDs {
		meta, err := s.ReadMediaMeta(fid)
		if err != nil {
			slog.Warn("skip unreadable media meta", "folder_id", fid, "err", err)
			continue
		}
		for relPath, file := range meta.Files {
			if req.MediaType != "" && file.MediaType != req.MediaType {
				continue
			}
			items = append(items, mediaFileToItem(fid, relPath, file))
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})

	total := len(items)
	start := (req.Page - 1) * req.PageSize
	if start >= total {
		return &model.MediaListResult{Items: []model.MediaFileItem{}, Total: total, HasMore: false}, nil
	}
	end := start + req.PageSize
	if end > total {
		end = total
	}
	return &model.MediaListResult{
		Items:   items[start:end],
		Total:   total,
		HasMore: end < total,
	}, nil
}

// splitMediaID 解析 Bleve 文档 ID（格式: {folderID}/{relPath}）
func splitMediaID(id string) (folderID, relPath string, ok bool) {
	idx := strings.IndexByte(id, '/')
	if idx < 0 {
		return "", "", false
	}
	return id[:idx], id[idx+1:], true
}

// listMediaFromBleve 有搜索/标签时走 Bleve 查询，结果从 media_meta.json 补全字段
func (s *Store) listMediaFromBleve(req model.MediaListReq) (*model.MediaListResult, error) {
	typeQ := bleve.NewTermQuery("media")
	typeQ.SetField("_type")
	conjunction := bleve.NewConjunctionQuery(typeQ)

	if req.FolderID != "" {
		fQ := bleve.NewTermQuery(req.FolderID)
		fQ.SetField("folder_id")
		conjunction.AddQuery(fQ)
	}
	if req.MediaType != "" {
		mtQ := bleve.NewTermQuery(req.MediaType)
		mtQ.SetField("media_type")
		conjunction.AddQuery(mtQ)
	}
	if req.Search != "" {
		fnQ := bleve.NewMatchQuery(req.Search)
		fnQ.SetField("filename")
		descQ := bleve.NewMatchQuery(req.Search)
		descQ.SetField("description")
		conjunction.AddQuery(bleve.NewDisjunctionQuery(fnQ, descQ))
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
		return nil, fmt.Errorf("search media: %w", err)
	}

	metaCache := make(map[string]*model.MediaMeta)
	items := make([]model.MediaFileItem, 0, len(result.Hits))

	for _, hit := range result.Hits {
		folderID, relPath, ok := splitMediaID(hit.ID)
		if !ok {
			continue
		}
		meta, cached := metaCache[folderID]
		if !cached {
			meta, err = s.ReadMediaMeta(folderID)
			if err != nil {
				slog.Warn("skip media meta in search", "folder_id", folderID, "err", err)
				continue
			}
			metaCache[folderID] = meta
		}
		file, exists := meta.Files[relPath]
		if !exists {
			continue
		}
		items = append(items, mediaFileToItem(folderID, relPath, file))
	}

	total := int(result.Total)
	return &model.MediaListResult{
		Items:   items,
		Total:   total,
		HasMore: (req.Page-1)*req.PageSize+len(items) < total,
	}, nil
}
