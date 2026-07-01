package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"collections/internal/model"
	"collections/internal/store"
)

// MediaService 媒体文件夹管理、扫描、缩略图、标签的业务逻辑层。
// 公开方法即前端可调用接口（通过 Wails 绑定）。
type MediaService struct {
	Store *store.Store
}

// ────────────────────── Folder Management ──────────────────────

// ListFolders 列出所有已注册的媒体文件夹
func (m *MediaService) ListFolders() (_ []model.MediaFolder, err error) {
	defer logError(&err)
	return m.Store.ListFolders()
}

// AddFolder 添加媒体文件夹。校验路径存在、可读、不与已有文件夹嵌套，
// 创建注册表记录和 persist 目录。Name 为空时取路径末段目录名。
func (m *MediaService) AddFolder(req model.AddFolderReq) (_ *model.MediaFolder, err error) {
	defer logError(&err)

	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("folder path required")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist")
		}
		return nil, fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	if err := m.checkFolderNesting(absPath, ""); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = filepath.Base(absPath)
	}

	folder, err := m.Store.CreateFolder(absPath, name)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	slog.Info("media folder added", "id", folder.ID, "path", absPath, "name", name)
	return folder, nil
}

// checkFolderNesting 校验新路径不与已有文件夹存在父子嵌套关系。
// excludeID 非空时跳过该文件夹（用于 UpdateFolderPath 排除自身）。
func (m *MediaService) checkFolderNesting(newPath, excludeID string) error {
	folders, err := m.Store.ListFolders()
	if err != nil {
		return fmt.Errorf("check nesting: %w", err)
	}

	cleanNew := filepath.Clean(newPath)
	for _, f := range folders {
		if f.ID == excludeID {
			continue
		}
		cleanExisting := filepath.Clean(f.Path)

		if cleanNew == cleanExisting {
			return fmt.Errorf("folder already registered: %s", f.Name)
		}
		if isSubPath(cleanNew, cleanExisting) {
			return fmt.Errorf("path nests with existing folder %q (%s)", f.Name, f.Path)
		}
		if isSubPath(cleanExisting, cleanNew) {
			return fmt.Errorf("path nests with existing folder %q (%s)", f.Name, f.Path)
		}
	}
	return nil
}

// RemoveFolder 移除媒体文件夹。遍历该文件夹下所有文件的标签，
// 批量递减 media_tag count，删除 BuntDB 注册表、Bleve 索引和 persist 目录。
func (m *MediaService) RemoveFolder(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("folder ID required")
	}

	folder, err := m.Store.GetFolder(id)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}

	m.cleanFolderTags(folder.ID)

	if err := m.Store.DeleteFolder(id); err != nil {
		return fmt.Errorf("delete folder registry: %w", err)
	}

	m.deleteFolderIndex(folder.ID)

	if err := m.Store.RemoveMediaFolderDir(id); err != nil {
		slog.Warn("failed to remove media folder dir", "id", id, "err", err)
	}

	slog.Info("media folder removed", "id", id, "path", folder.Path)
	return nil
}

// cleanFolderTags 汇总文件夹下所有文件的标签，批量递减 media_tag count
func (m *MediaService) cleanFolderTags(folderID string) {
	meta, err := m.Store.ReadMediaMeta(folderID)
	if err != nil {
		slog.Warn("skip tag cleanup: cannot read media meta", "folder_id", folderID, "err", err)
		return
	}

	deltas := make(map[string]int)
	for _, file := range meta.Files {
		for _, tag := range file.Tags {
			deltas[tag]--
		}
	}

	if len(deltas) > 0 {
		if err := m.Store.BatchAdjustTagCounts("media_tag", deltas); err != nil {
			slog.Warn("failed to adjust media tag counts on folder removal", "folder_id", folderID, "err", err)
		}
	}
}

// deleteFolderIndex 从 Bleve 中删除该文件夹下所有媒体文档
func (m *MediaService) deleteFolderIndex(folderID string) {
	meta, err := m.Store.ReadMediaMeta(folderID)
	if err != nil {
		return
	}

	for relPath := range meta.Files {
		docID := folderID + "/" + relPath
		if err := m.Store.DeleteDoc(docID, "media"); err != nil {
			slog.Warn("failed to delete media index", "doc_id", docID, "err", err)
		}
	}
}

// UpdateFolderPath 更新文件夹路径映射（文件夹物理位置变化后调用）。
// 校验新路径存在、可读、不与其他文件夹嵌套。
func (m *MediaService) UpdateFolderPath(req model.UpdateFolderPathReq) (_ *model.MediaFolder, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("folder ID required")
	}

	newPath := strings.TrimSpace(req.Path)
	if newPath == "" {
		return nil, fmt.Errorf("folder path required")
	}

	absPath, err := filepath.Abs(newPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist")
		}
		return nil, fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	if err := m.checkFolderNesting(absPath, req.ID); err != nil {
		return nil, err
	}

	folder, err := m.Store.GetFolder(req.ID)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	folder.Path = absPath
	if err := m.Store.UpdateFolder(folder); err != nil {
		return nil, fmt.Errorf("update folder path: %w", err)
	}

	slog.Info("media folder path updated", "id", req.ID, "path", absPath)
	return folder, nil
}

// isSubPath 判断 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}
