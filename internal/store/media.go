package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// mediaFolderDir 返回媒体文件夹的 persist 目录路径：persist/media-folders/{id}
func (s *Store) mediaFolderDir(folderID string) string {
	return filepath.Join(s.persistDir, "media-folders", folderID)
}

// mediaFileToItem 将 MediaFile 转换为带 FolderID/RelPath 的列表项，确保 Tags 非 nil
func mediaFileToItem(folderID, relPath string, file model.MediaFile) model.MediaFileItem {
	if file.Tags == nil {
		file.Tags = []string{}
	}
	return model.MediaFileItem{
		MediaFile: file,
		FolderID:  folderID,
		RelPath:   relPath,
	}
}

// --- 文件夹注册表 CRUD（BuntDB） ---

// CreateFolder 注册媒体文件夹（BuntDB key: folder:{id}）并创建 persist 目录结构
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
			return fmt.Errorf("folder not found")
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
		_, err := tx.Get("folder:" + folder.ID)
		if err != nil {
			return err
		}
		_, _, err = tx.Set("folder:"+folder.ID, string(val), nil)
		return err
	})
}

// DeleteFolder 从 BuntDB 中删除文件夹注册表条目
func (s *Store) DeleteFolder(id string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete("folder:" + id)
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

// ReadMediaMeta 读取文件夹的媒体元数据
func (s *Store) ReadMediaMeta(folderID string) (*model.MediaMeta, error) {
	path := filepath.Join(s.mediaFolderDir(folderID), "media_meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read media_meta.json failed: %w", err)
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

// ReadTreeHash 读取 Merkle Tree 快照。
// 文件不存在时返回 error（可通过 errors.Is(err, os.ErrNotExist) 判断）。
func (s *Store) ReadTreeHash(folderID string) (*model.TreeHashFile, error) {
	path := filepath.Join(s.mediaFolderDir(folderID), "tree_hash.json")
	data, err := os.ReadFile(path)
	if err != nil {
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
