package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"
)

const maxUploadSize = 10 << 20 // 10 MB

// UploadService 统一文件上传，按 scene 路由到 persist 子目录。
type UploadService struct {
	Store *store.Store
}

// UploadFile 上传文件。按 scene 路由到对应 persist 子目录，
// 图片附件自动生成缩略图（{filename}.thumb.jpg）。
// scene、filename、data 必填，EntityID 用于构建存储路径。
// 文件大小上限 10MB。note-image 场景使用内容 hash 命名实现去重。
func (u *UploadService) UploadFile(req model.UploadFileReq) (_ *model.UploadFileResult, err error) {
	defer logError(&err)
	if req.Scene == "" {
		return nil, fmt.Errorf("scene required")
	}
	if req.Filename == "" {
		return nil, fmt.Errorf("filename required")
	}
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("file data empty")
	}
	if len(req.Data) > maxUploadSize {
		return nil, fmt.Errorf("file exceeds %dMB limit", maxUploadSize>>20)
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if !isAllowedUploadExt(req.Scene, ext) {
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	persistDir := u.Store.PersistDir()

	switch req.Scene {
	case "site-icon":
		dir := filepath.Join(persistDir, "url-assets", "icons")
		filename := req.EntityID + ext
		savePath, err := util.SafePath(dir, filename)
		if err != nil {
			return nil, fmt.Errorf("invalid filename: %w", err)
		}
		if err := util.AtomicWrite(savePath, req.Data, 0644); err != nil {
			return nil, fmt.Errorf("save file failed: %w", err)
		}
		return &model.UploadFileResult{Path: filename}, nil

	case "site-attachment", "bm-attachment":
		baseDir := filepath.Join(persistDir, "url-assets", "attachments")
		savePath, err := util.SafePath(baseDir, filepath.Join(req.EntityID, req.Filename))
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
		if err := util.AtomicWrite(savePath, req.Data, 0644); err != nil {
			return nil, fmt.Errorf("save file failed: %w", err)
		}
		if model.IsImageExt(ext) {
			generateThumbnail(savePath)
		}
		// TODO: GIF 动画预览（当前仅取首帧生成静态缩略图，后续生成 .preview.webp 动画）
		// TODO: 视频缩略图（需 ffmpeg 抽帧，当前不可用时跳过）
		return &model.UploadFileResult{Path: req.Filename}, nil

	case "note-image":
		baseDir := filepath.Join(persistDir, "note-images")
		hash := sha256.Sum256(req.Data)
		hashStr := hex.EncodeToString(hash[:])[:32]
		filename := hashStr + ext
		savePath, err := util.SafePath(baseDir, filepath.Join(req.EntityID, filename))
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}
		if err := util.AtomicWrite(savePath, req.Data, 0644); err != nil {
			return nil, fmt.Errorf("save file failed: %w", err)
		}
		return &model.UploadFileResult{Path: filepath.ToSlash(filepath.Join("note-images", req.EntityID, filename))}, nil

	default:
		return nil, fmt.Errorf("unknown upload scene: %s", req.Scene)
	}
}

// DeleteAttachment 删除单个附件及其缩略图（仅 url-assets/attachments/{entityID}/ 范围）。
// entityID 和 filename 必填。缩略图删除失败时静默忽略。
func (u *UploadService) DeleteAttachment(entityID, filename string) (err error) {
	defer logError(&err)
	if entityID == "" {
		return fmt.Errorf("entity ID required")
	}
	if filename == "" {
		return fmt.Errorf("filename required")
	}
	baseDir := filepath.Join(u.Store.PersistDir(), "url-assets", "attachments")
	src, err := util.SafePath(baseDir, filepath.Join(entityID, filename))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete attachment failed: %w", err)
	}
	thumb := src + ".thumb.jpg"
	os.Remove(thumb)
	return nil
}

// --- 缩略图生成 ---

// generateThumbnail 为图片附件生成 JPEG 缩略图（{srcPath}.thumb.jpg），
// 复用 generateImageThumbnail 的实现。
func generateThumbnail(srcPath string) {
	thumbPath := srcPath + ".thumb.jpg"
	if _, _, err := generateImageThumbnail(srcPath, thumbPath); err != nil {
		slog.Warn("thumbnail generation failed", "path", srcPath, "err", err)
	}
}

// fitDimensions 按最大边等比缩放，保证宽高不超过 maxDim
func fitDimensions(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w > h {
		return maxDim, max(h*maxDim/w, 1)
	}
	return max(w*maxDim/h, 1), maxDim
}

// --- 扩展名校验 ---

// isAllowedUploadExt 按 scene 校验文件扩展名是否在允许范围内
func isAllowedUploadExt(scene, ext string) bool {
	switch scene {
	case "site-icon":
		return model.IsImageExt(ext)
	case "site-attachment", "bm-attachment":
		return model.IsImageExt(ext) || model.IsVideoExt(ext) || ext == ".txt"
	case "note-image":
		return model.IsImageExt(ext)
	default:
		return false
	}
}
