package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"github.com/blevesearch/bleve/v2"
	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

func (s *Store) mediaFolderDir(folderID string) string {
	return filepath.Join(s.persistDir, "media-folders", folderID)
}

func mediaFileToItem(folderID, relPath string, file model.MediaFile) model.MediaFileItem {
	tags := file.Tags
	if tags == nil {
		tags = []string{}
	}
	return model.MediaFileItem{
		FolderID:  folderID,
		RelPath:   relPath,
		MediaType: file.MediaType,
		Tags:      tags,
		Thumbnail: file.Thumbnail,
		Preview:   file.Preview,
		FileSize:  file.FileSize,
		Width:     file.Width,
		Height:    file.Height,
		Duration:  file.Duration,
		UpdatedAt: file.UpdatedAt,
	}
}

// --- 文件夹注册表 CRUD（BuntDB） ---

// CreateFolder 注册媒体文件夹并创建 persist 目录
func (s *Store) CreateFolder(path, name string) (*model.MediaFolder, error) {
	folder := &model.MediaFolder{
		ID:        uuid.New().String(),
		Path:      path,
		Name:      name,
		FileCount: 0,
		AddedAt:   time.Now(),
	}
	val, _ := json.Marshal(folder)

	err := s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("folder:"+folder.ID, string(val), nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	dir := s.mediaFolderDir(folder.ID)
	if err := os.MkdirAll(filepath.Join(dir, "thumbnails"), 0755); err != nil {
		return nil, fmt.Errorf("create media folder dir: %w", err)
	}

	slog.Info("folder created", "id", folder.ID, "path", path)
	return folder, nil
}

// GetFolder 按 ID 查询文件夹
func (s *Store) GetFolder(id string) (*model.MediaFolder, error) {
	var folder model.MediaFolder
	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("folder:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("文件夹不存在")
		}
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &folder)
	})
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// UpdateFolder 更新文件夹注册表（路径、名称、file_count、last_scan_at 等）
func (s *Store) UpdateFolder(folder *model.MediaFolder) error {
	val, err := json.Marshal(folder)
	if err != nil {
		return fmt.Errorf("marshal folder: %w", err)
	}
	return s.db.Update(func(tx *buntdb.Tx) error {
		if _, err := tx.Get("folder:" + folder.ID); err == buntdb.ErrNotFound {
			return fmt.Errorf("文件夹不存在")
		}
		_, _, err := tx.Set("folder:"+folder.ID, string(val), nil)
		return err
	})
}

// DeleteFolder 从 BuntDB 中删除文件夹注册表条目
func (s *Store) DeleteFolder(id string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete("folder:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("文件夹不存在")
		}
		return err
	})
}

// ListFolders 列出所有已注册的媒体文件夹
func (s *Store) ListFolders() ([]model.MediaFolder, error) {
	var folders []model.MediaFolder
	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("folder:*", func(key, value string) bool {
			var f model.MediaFolder
			if err := json.Unmarshal([]byte(value), &f); err != nil {
				slog.Warn("skip corrupted folder", "key", key, "err", err)
				return true
			}
			folders = append(folders, f)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	return folders, nil
}

// RemoveMediaFolderDir 删除 persist/media-folders/{id}/ 整个目录
func (s *Store) RemoveMediaFolderDir(folderID string) error {
	return os.RemoveAll(s.mediaFolderDir(folderID))
}

// --- media_meta.json / tree_hash.json 读写 ---

// ReadMediaMeta 读取文件夹的媒体元数据，文件不存在时返回空 meta
func (s *Store) ReadMediaMeta(folderID string) (*model.MediaMeta, error) {
	path := filepath.Join(s.mediaFolderDir(folderID), "media_meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.MediaMeta{FolderID: folderID, Files: make(map[string]model.MediaFile)}, nil
		}
		return nil, fmt.Errorf("read media_meta.json: %w", err)
	}
	var meta model.MediaMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal media_meta.json: %w", err)
	}
	if meta.Files == nil {
		meta.Files = make(map[string]model.MediaFile)
	}
	return &meta, nil
}

// WriteMediaMeta 原子写入文件夹的媒体元数据
func (s *Store) WriteMediaMeta(folderID string, meta *model.MediaMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal media_meta: %w", err)
	}
	path := filepath.Join(s.mediaFolderDir(folderID), "media_meta.json")
	return util.AtomicWrite(path, data, 0644)
}

// ReadTreeHash 读取 Merkle Tree 快照，文件不存在时返回 (nil, nil)
func (s *Store) ReadTreeHash(folderID string) (*model.TreeHashFile, error) {
	path := filepath.Join(s.mediaFolderDir(folderID), "tree_hash.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tree_hash.json: %w", err)
	}
	var th model.TreeHashFile
	if err := json.Unmarshal(data, &th); err != nil {
		return nil, fmt.Errorf("unmarshal tree_hash.json: %w", err)
	}
	return &th, nil
}

// WriteTreeHash 原子写入 Merkle Tree 快照
func (s *Store) WriteTreeHash(folderID string, th *model.TreeHashFile) error {
	data, err := json.MarshalIndent(th, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tree_hash: %w", err)
	}
	path := filepath.Join(s.mediaFolderDir(folderID), "tree_hash.json")
	return util.AtomicWrite(path, data, 0644)
}

// --- 媒体文件列表查询 ---

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

func splitMediaID(id string) (folderID, relPath string, ok bool) {
	idx := strings.IndexByte(id, '/')
	if idx < 0 {
		return "", "", false
	}
	return id[:idx], id[idx+1:], true
}

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
		conjunction.AddQuery(fnQ)
	}
	for _, tag := range req.Tags {
		tagQ := bleve.NewTermQuery(tag)
		tagQ.SetField("tags")
		conjunction.AddQuery(tagQ)
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
