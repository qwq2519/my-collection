package service

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"
)

// MediaService 媒体文件夹管理、扫描、缩略图、标签的业务逻辑层。
// 公开方法即前端可调用接口（通过 Wails 绑定）。
type MediaService struct {
	Store      *store.Store
	ffmpegPath string // 懒检测，首次扫描时缓存
}

// detectFFmpeg 返回 ffmpeg 可执行文件路径，不可用时返回空串
func (m *MediaService) detectFFmpeg() string {
	if m.ffmpegPath != "" {
		return m.ffmpegPath
	}
	status := util.DetectFFmpeg(m.Store.PersistDir())
	if status.Available {
		m.ffmpegPath = status.BinPath
	}
	return m.ffmpegPath
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

// ────────────────────── Scan ──────────────────────

// ScanFolder 扫描单个媒体文件夹，检测文件变化并处理。
// 流程：构建 Merkle Tree → diff → 处理新增/删除/修改 → 更新 meta + Bleve + tree_hash → 更新注册表。
func (m *MediaService) ScanFolder(id string) (_ *model.ScanComplete, err error) {
	defer logError(&err)
	if id == "" {
		return nil, fmt.Errorf("folder ID required")
	}

	folder, err := m.Store.GetFolder(id)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	var cachedRoot *model.TreeNode
	if cached, err := m.Store.ReadTreeHash(id); err == nil {
		cachedRoot = cached.Root
	}

	newRoot, err := buildTree(folder.Path, cachedRoot)
	if err != nil {
		return nil, fmt.Errorf("scan filesystem: %w", err)
	}

	diff := diffTrees(cachedRoot, newRoot)

	meta, err := m.Store.ReadMediaMeta(id)
	if err != nil {
		meta = &model.MediaMeta{
			SchemaVersion: 1,
			FolderID:      id,
			Files:         make(map[string]model.MediaFile),
		}
	}

	thumbDir := filepath.Join(m.Store.PersistDir(), "media-folders", id, "thumbnails")
	ffmpeg := m.detectFFmpeg()

	var newDocs []store.BleveDoc
	var deleteIDs []string
	var tagDeltas map[string]int

	if len(diff.Added) > 0 {
		docs := m.processAdded(id, folder.Path, thumbDir, ffmpeg, diff.Added, meta)
		newDocs = append(newDocs, docs...)
	}

	if len(diff.Removed) > 0 {
		tagDeltas, deleteIDs = m.processRemoved(id, thumbDir, diff.Removed, meta)
	}

	if len(diff.Modified) > 0 {
		docs := m.processModified(id, folder.Path, thumbDir, ffmpeg, diff.Modified, meta)
		newDocs = append(newDocs, docs...)
	}

	if err := m.Store.WriteMediaMeta(id, meta); err != nil {
		return nil, fmt.Errorf("write media meta: %w", err)
	}

	treeHash := &model.TreeHashFile{
		SchemaVersion: 1,
		FolderID:      id,
		Root:          newRoot,
	}
	if err := m.Store.WriteTreeHash(id, treeHash); err != nil {
		return nil, fmt.Errorf("write tree hash: %w", err)
	}

	if len(deleteIDs) > 0 || len(newDocs) > 0 {
		if err := m.Store.RebuildDocs(deleteIDs, newDocs); err != nil {
			slog.Warn("bleve batch update failed during scan", "folder_id", id, "err", err)
		}
	}

	if len(tagDeltas) > 0 {
		if err := m.Store.BatchAdjustTagCounts("media_tag", tagDeltas); err != nil {
			slog.Warn("failed to adjust media tag counts after scan", "folder_id", id, "err", err)
		}
	}

	folder.FileCount = len(meta.Files)
	folder.LastScanAt = time.Now()
	if err := m.Store.UpdateFolder(folder); err != nil {
		slog.Warn("failed to update folder after scan", "folder_id", id, "err", err)
	}

	result := &model.ScanComplete{
		FolderID: id,
		Added:    len(diff.Added),
		Removed:  len(diff.Removed),
		Modified: len(diff.Modified),
	}
	slog.Info("folder scan complete",
		"folder_id", id, "added", result.Added, "removed", result.Removed, "modified", result.Modified)
	return result, nil
}

// ScanAllFolders 依次扫描所有已注册媒体文件夹
func (m *MediaService) ScanAllFolders() (_ []model.ScanComplete, err error) {
	defer logError(&err)
	folders, err := m.Store.ListFolders()
	if err != nil {
		return nil, err
	}

	var results []model.ScanComplete
	for _, f := range folders {
		result, err := m.ScanFolder(f.ID)
		if err != nil {
			slog.Warn("scan folder failed, skipping", "folder_id", f.ID, "err", err)
			continue
		}
		results = append(results, *result)
	}
	return results, nil
}

// ────────────────────── Scan Helpers ──────────────────────

// processFile 处理单个文件：生成缩略图/预览、提取元数据。
// existing 非 nil 时保留用户数据（tags/description/updated_at）。
func (m *MediaService) processFile(folderID, folderPath, thumbDir, ffmpeg, relPath string, existing *model.MediaFile) (model.MediaFile, error) {
	srcPath := filepath.Join(folderPath, filepath.FromSlash(relPath))
	ext := strings.ToLower(filepath.Ext(relPath))
	mediaType, _ := model.LookupMediaType(ext)

	fileInfo, err := os.Stat(srcPath)
	if err != nil {
		return model.MediaFile{}, fmt.Errorf("stat file: %w", err)
	}

	now := time.Now()
	thumbName, previewName := thumbNames(folderID, relPath)
	thumbPath := filepath.Join(thumbDir, thumbName)
	previewPath := filepath.Join(thumbDir, previewName)

	file := model.MediaFile{
		MediaType: mediaType,
		Tags:      []string{},
		Thumbnail: thumbName,
		ScannedAt: now,
		UpdatedAt: now,
		FileSize:  fileInfo.Size(),
	}

	if existing != nil {
		file.Tags = existing.Tags
		file.Description = existing.Description
		file.UpdatedAt = existing.UpdatedAt
	}

	switch mediaType {
	case model.MediaTypeImage:
		w, h, err := generateImageThumbnail(srcPath, thumbPath)
		if err != nil {
			slog.Warn("image thumbnail failed", "path", relPath, "err", err)
			file.Thumbnail = ""
		} else {
			file.Width = &w
			file.Height = &h
		}
		if ext == ".gif" && ffmpeg != "" {
			if err := generateAnimatedPreview(ffmpeg, srcPath, previewPath); err == nil {
				file.Preview = previewName
			}
		}

	case model.MediaTypeVideo:
		if ffmpeg != "" {
			if err := generateVideoThumbnail(ffmpeg, srcPath, thumbPath); err != nil {
				slog.Warn("video thumbnail failed", "path", relPath, "err", err)
				file.Thumbnail = ""
			}
			if err := generateAnimatedPreview(ffmpeg, srcPath, previewPath); err == nil {
				file.Preview = previewName
			}
			w, h, dur, err := probeVideoMeta(ffmpeg, srcPath)
			if err == nil {
				if w > 0 {
					file.Width = &w
					file.Height = &h
				}
				if dur > 0 {
					file.Duration = &dur
				}
			}
		} else {
			file.Thumbnail = ""
		}

	case model.MediaTypeAudio:
		file.Thumbnail = ""
		if ffmpeg != "" {
			_, _, dur, err := probeVideoMeta(ffmpeg, srcPath)
			if err == nil && dur > 0 {
				file.Duration = &dur
			}
		}
	}

	return file, nil
}

func (m *MediaService) processAdded(folderID, folderPath, thumbDir, ffmpeg string, relPaths []string, meta *model.MediaMeta) []store.BleveDoc {
	var docs []store.BleveDoc
	for _, relPath := range relPaths {
		file, err := m.processFile(folderID, folderPath, thumbDir, ffmpeg, relPath, nil)
		if err != nil {
			slog.Warn("skip added file", "path", relPath, "err", err)
			continue
		}
		meta.Files[relPath] = file
		docs = append(docs, store.BleveDoc{
			ID:     folderID + "/" + relPath,
			Fields: mediaBleveFields(folderID, relPath, file),
		})
	}
	return docs
}

func (m *MediaService) processRemoved(folderID, thumbDir string, relPaths []string, meta *model.MediaMeta) (tagDeltas map[string]int, deleteIDs []string) {
	tagDeltas = make(map[string]int)
	for _, relPath := range relPaths {
		file, exists := meta.Files[relPath]
		if !exists {
			continue
		}
		for _, tag := range file.Tags {
			tagDeltas[tag]--
		}
		removeThumbFiles(thumbDir, file.Thumbnail, file.Preview)
		delete(meta.Files, relPath)
		deleteIDs = append(deleteIDs, folderID+"/"+relPath)
	}
	return tagDeltas, deleteIDs
}

func (m *MediaService) processModified(folderID, folderPath, thumbDir, ffmpeg string, relPaths []string, meta *model.MediaMeta) []store.BleveDoc {
	var docs []store.BleveDoc
	for _, relPath := range relPaths {
		existing, ok := meta.Files[relPath]
		var existingPtr *model.MediaFile
		if ok {
			existingPtr = &existing
		}
		file, err := m.processFile(folderID, folderPath, thumbDir, ffmpeg, relPath, existingPtr)
		if err != nil {
			slog.Warn("skip modified file", "path", relPath, "err", err)
			continue
		}
		meta.Files[relPath] = file
		docs = append(docs, store.BleveDoc{
			ID:     folderID + "/" + relPath,
			Fields: mediaBleveFields(folderID, relPath, file),
		})
	}
	return docs
}

// mediaBleveFields 构建媒体文件的 Bleve 索引字段
func mediaBleveFields(folderID, relPath string, file model.MediaFile) map[string]interface{} {
	return map[string]interface{}{
		"_type":      "media",
		"folder_id":  folderID,
		"media_type": file.MediaType,
		"filename":   path.Base(relPath),
		"tags":       file.Tags,
		"description": file.Description,
		"updated_at": file.UpdatedAt,
	}
}

// isSubPath 判断 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}
