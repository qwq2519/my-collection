package service

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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

	meta, err := m.Store.ReadMediaMeta(folder.ID)
	if err != nil {
		slog.Warn("skip tag/index cleanup: cannot read media meta", "folder_id", folder.ID, "err", err)
	} else {
		m.cleanFolderTags(folder.ID, meta)
		m.deleteFolderIndex(folder.ID, meta)
	}

	if err := m.Store.DeleteFolder(id); err != nil {
		return fmt.Errorf("delete folder registry: %w", err)
	}

	if err := m.Store.RemoveMediaFolderDir(id); err != nil {
		slog.Warn("failed to remove media folder dir", "id", id, "err", err)
	}

	slog.Info("media folder removed", "id", id, "path", folder.Path)
	return nil
}

// cleanFolderTags 汇总文件夹下所有文件的标签，批量递减 media_tag count
func (m *MediaService) cleanFolderTags(folderID string, meta *model.MediaMeta) {
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
func (m *MediaService) deleteFolderIndex(folderID string, meta *model.MediaMeta) {
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
		newDocs = append(newDocs, m.processFiles(id, folder.Path, thumbDir, ffmpeg, diff.Added, meta)...)
	}

	if len(diff.Removed) > 0 {
		tagDeltas, deleteIDs = m.processRemoved(id, thumbDir, diff.Removed, meta)
	}

	if len(diff.Modified) > 0 {
		newDocs = append(newDocs, m.processFiles(id, folder.Path, thumbDir, ffmpeg, diff.Modified, meta)...)
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
	now := time.Now()
	folder.LastScanAt = &now
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

	default:
		file.Thumbnail = ""
	}

	return file, nil
}

// processFiles 处理新增或修改的文件：生成缩略图/预览、提取元数据、更新 meta 和 Bleve 索引。
// 新增文件在 meta 中无记录，自动以 nil existing 处理；修改文件则保留已有用户数据。
func (m *MediaService) processFiles(folderID, folderPath, thumbDir, ffmpeg string, relPaths []string, meta *model.MediaMeta) []store.BleveDoc {
	var docs []store.BleveDoc
	for _, relPath := range relPaths {
		existing, ok := meta.Files[relPath]
		var existingPtr *model.MediaFile
		if ok {
			existingPtr = &existing
		}
		file, err := m.processFile(folderID, folderPath, thumbDir, ffmpeg, relPath, existingPtr)
		if err != nil {
			slog.Warn("skip file", "path", relPath, "err", err)
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

// ────────────────────── Query ──────────────────────

// ListMediaFiles 分页查询媒体文件，支持搜索、标签筛选、文件夹筛选、类型筛选
func (m *MediaService) ListMediaFiles(req model.MediaListReq) (_ *model.MediaListResult, err error) {
	defer logError(&err)
	result, err := m.Store.ListMediaFiles(req)
	if err != nil {
		return nil, err
	}
	for i := range result.Items {
		m.fillItemURLs(&result.Items[i])
	}
	return result, nil
}

// GetMediaFile 获取单个媒体文件详情
func (m *MediaService) GetMediaFile(folderID, relPath string) (_ *model.MediaFileItem, err error) {
	defer logError(&err)
	if folderID == "" {
		return nil, fmt.Errorf("folder ID required")
	}
	if relPath == "" {
		return nil, fmt.Errorf("file path required")
	}

	meta, err := m.Store.ReadMediaMeta(folderID)
	if err != nil {
		return nil, fmt.Errorf("read media meta: %w", err)
	}

	file, exists := meta.Files[relPath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", relPath)
	}

	if file.Tags == nil {
		file.Tags = []string{}
	}
	item := &model.MediaFileItem{
		MediaFile: file,
		FolderID:  folderID,
		RelPath:   relPath,
	}
	m.fillItemURLs(item)
	return item, nil
}

// fillItemURLs 为 MediaFileItem 填充完整的前端可访问路径
func (m *MediaService) fillItemURLs(item *model.MediaFileItem) {
	base := "/persist/media-folders/" + item.FolderID + "/thumbnails/"
	if item.Thumbnail != "" {
		item.ThumbnailURL = base + item.Thumbnail
	}
	if item.Preview != "" {
		item.PreviewURL = base + item.Preview
	}
}

// ────────────────────── Tags & Description ──────────────────────

// UpdateMediaFile 更新单个媒体文件的标签和/或描述。
// Tags 非 nil 时替换标签并同步维护 media_tag 注册表 count；Description 非 nil 时更新描述。
// 任一字段有更新则更新 updated_at。
func (m *MediaService) UpdateMediaFile(req model.UpdateMediaFileReq) (_ *model.MediaFileItem, err error) {
	defer logError(&err)
	if req.FolderID == "" {
		return nil, fmt.Errorf("folder ID required")
	}
	if req.RelPath == "" {
		return nil, fmt.Errorf("file path required")
	}
	if req.Tags == nil && req.Description == nil {
		return m.GetMediaFile(req.FolderID, req.RelPath)
	}

	if req.Tags != nil {
		tags, err := normalizeTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		req.Tags = &tags
	}

	meta, err := m.Store.ReadMediaMeta(req.FolderID)
	if err != nil {
		return nil, fmt.Errorf("read media meta: %w", err)
	}

	file, exists := meta.Files[req.RelPath]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", req.RelPath)
	}

	var oldTags []string
	if req.Tags != nil {
		oldTags = file.Tags
		file.Tags = *req.Tags
	}
	if req.Description != nil {
		file.Description = *req.Description
	}
	file.UpdatedAt = time.Now()
	meta.Files[req.RelPath] = file

	if err := m.Store.WriteMediaMeta(req.FolderID, meta); err != nil {
		return nil, fmt.Errorf("write media meta: %w", err)
	}

	if req.Tags != nil {
		m.adjustMediaTagCounts(file.Tags, oldTags)
	}

	docID := req.FolderID + "/" + req.RelPath
	m.Store.IndexDoc(docID, mediaBleveFields(req.FolderID, req.RelPath, file))

	item := &model.MediaFileItem{
		MediaFile: file,
		FolderID:  req.FolderID,
		RelPath:   req.RelPath,
	}
	m.fillItemURLs(item)
	return item, nil
}

// BatchUpdateMediaTags 批量为媒体文件追加标签（不覆盖已有标签）
func (m *MediaService) BatchUpdateMediaTags(req model.BatchUpdateMediaTagsReq) (err error) {
	defer logError(&err)
	if req.FolderID == "" {
		return fmt.Errorf("folder ID required")
	}
	if len(req.RelPaths) == 0 || len(req.Tags) == 0 {
		return nil
	}

	tags, err := normalizeTags(req.Tags)
	if err != nil {
		return err
	}

	meta, err := m.Store.ReadMediaMeta(req.FolderID)
	if err != nil {
		return fmt.Errorf("read media meta: %w", err)
	}

	deltas := make(map[string]int)
	var docs []store.BleveDoc

	for _, relPath := range req.RelPaths {
		file, exists := meta.Files[relPath]
		if !exists {
			slog.Warn("skip missing file in batch tag", "path", relPath)
			continue
		}

		merged, added := mergeTags(file.Tags, tags)
		if len(added) == 0 {
			continue
		}

		for _, t := range added {
			deltas[t]++
		}

		file.Tags = merged
		file.UpdatedAt = time.Now()
		meta.Files[relPath] = file

		docs = append(docs, store.BleveDoc{
			ID:     req.FolderID + "/" + relPath,
			Fields: mediaBleveFields(req.FolderID, relPath, file),
		})
	}

	if err := m.Store.WriteMediaMeta(req.FolderID, meta); err != nil {
		return fmt.Errorf("write media meta: %w", err)
	}

	if len(deltas) > 0 {
		if err := m.Store.BatchAdjustTagCounts("media_tag", deltas); err != nil {
			slog.Warn("failed to adjust media tag counts after batch tag", "err", err)
		}
	}

	if len(docs) > 0 {
		if err := m.Store.RebuildDocs(nil, docs); err != nil {
			slog.Warn("bleve batch update failed after batch tag", "err", err)
		}
	}

	return nil
}

// adjustMediaTagCounts 计算新旧标签的差值，批量更新 media_tag 注册表 count
func (m *MediaService) adjustMediaTagCounts(newTags, oldTags []string) {
	deltas := make(map[string]int)
	for _, t := range newTags {
		deltas[t]++
	}
	for _, t := range oldTags {
		deltas[t]--
	}

	nonZero := make(map[string]int)
	for k, v := range deltas {
		if v != 0 {
			nonZero[k] = v
		}
	}
	if len(nonZero) == 0 {
		return
	}

	if err := m.Store.BatchAdjustTagCounts("media_tag", nonZero); err != nil {
		slog.Warn("failed to adjust media tag counts", "err", err)
	}
}

// ────────────────────── System Integration ──────────────────────

// OpenInExplorer 在系统文件管理器中打开媒体文件所在目录并选中该文件
func (m *MediaService) OpenInExplorer(folderID, relPath string) (err error) {
	defer logError(&err)
	if folderID == "" {
		return fmt.Errorf("folder ID required")
	}
	if relPath == "" {
		return fmt.Errorf("file path required")
	}

	folder, err := m.Store.GetFolder(folderID)
	if err != nil {
		return fmt.Errorf("folder not found: %w", err)
	}

	absPath := filepath.Join(folder.Path, filepath.FromSlash(relPath))
	return openFileInExplorer(absPath)
}

// openFileInExplorer 调用系统文件管理器定位文件（跨平台）
func openFileInExplorer(absPath string) error {
	var cmd string
	var args []string

	switch goos := goOS(); goos {
	case "windows":
		cmd = "explorer.exe"
		args = []string{"/select,", absPath}
	case "darwin":
		cmd = "open"
		args = []string{"-R", absPath}
	default:
		cmd = "xdg-open"
		args = []string{filepath.Dir(absPath)}
	}

	return execCommand(cmd, args...)
}

// 以下两个函数方便测试时 mock

var goOS = func() string {
	return goOSReal()
}

func goOSReal() string {
	return runtime.GOOS
}

var execCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}

// isSubPath 判断 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	return strings.HasPrefix(child, parent+string(filepath.Separator))
}
