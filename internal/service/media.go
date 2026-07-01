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

// isSubPath 判断 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}
