package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/png"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const thumbMaxDim = 300

// UploadService 统一文件上传，按 scene 路由到 persist 子目录。
type UploadService struct {
	Store *store.Store
}

// UploadFile 上传文件。按 scene 路由到对应 persist 子目录，
// 图片附件自动生成缩略图（{filename}.thumb.jpg）。
func (u *UploadService) UploadFile(req model.UploadFileReq) (_ *model.UploadFileResult, err error) {
	defer logError(&err)
	if req.Scene == "" {
		return nil, fmt.Errorf("scene 不能为空")
	}
	if req.Filename == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if len(req.Data) == 0 {
		return nil, fmt.Errorf("文件内容为空")
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if !isAllowedUploadExt(req.Scene, ext) {
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	persistDir := u.Store.PersistDir()

	switch req.Scene {
	case "site-icon":
		dir := filepath.Join(persistDir, "url-assets", "icons")
		os.MkdirAll(dir, 0755)
		filename := req.EntityID + ext
		if err := util.AtomicWrite(filepath.Join(dir, filename), req.Data, 0644); err != nil {
			return nil, fmt.Errorf("保存文件失败: %w", err)
		}
		return &model.UploadFileResult{Path: filename}, nil

	case "site-attachment", "bm-attachment":
		dir := filepath.Join(persistDir, "url-assets", "attachments", req.EntityID)
		os.MkdirAll(dir, 0755)
		savePath := filepath.Join(dir, req.Filename)
		if err := util.AtomicWrite(savePath, req.Data, 0644); err != nil {
			return nil, fmt.Errorf("保存文件失败: %w", err)
		}
		if isImageExt(ext) {
			generateThumbnail(savePath)
		}
		// TODO: GIF 动画预览（当前仅取首帧生成静态缩略图，后续生成 .preview.webp 动画）
		// TODO: 视频缩略图（需 ffmpeg 抽帧，当前不可用时跳过）
		return &model.UploadFileResult{Path: req.Filename}, nil

	case "note-image":
		dir := filepath.Join(persistDir, "note-images", req.EntityID)
		os.MkdirAll(dir, 0755)
		hash := sha256.Sum256(req.Data)
		hashStr := hex.EncodeToString(hash[:])[:16]
		filename := hashStr + ext
		savePath := filepath.Join(dir, filename)
		if err := util.AtomicWrite(savePath, req.Data, 0644); err != nil {
			return nil, fmt.Errorf("保存文件失败: %w", err)
		}
		return &model.UploadFileResult{Path: filepath.ToSlash(filepath.Join("note-images", req.EntityID, filename))}, nil

	default:
		return nil, fmt.Errorf("未知的上传场景: %s", req.Scene)
	}
}

// DeleteAttachment 删除单个附件及其缩略图
func (u *UploadService) DeleteAttachment(entityID, filename string) (err error) {
	defer logError(&err)
	dir := filepath.Join(u.Store.PersistDir(), "url-assets", "attachments", entityID)
	src := filepath.Join(dir, filename)
	if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除附件失败: %w", err)
	}
	thumb := src + ".thumb.jpg"
	os.Remove(thumb)
	return nil
}

// --- 缩略图生成 ---

func generateThumbnail(srcPath string) {
	thumbPath := srcPath + ".thumb.jpg"

	f, err := os.Open(srcPath)
	if err != nil {
		slog.Warn("open for thumbnail failed", "path", srcPath, "err", err)
		return
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		// AVIF、SVG 等格式暂无纯 Go 解码器，跳过
		slog.Warn("decode image for thumbnail failed", "path", srcPath, "err", err)
		return
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return
	}

	tw, th := fitDimensions(w, h, thumbMaxDim)
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.BiLinear.Scale(dst, dst.Rect, src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		slog.Warn("encode thumbnail failed", "path", srcPath, "err", err)
		return
	}

	if err := util.AtomicWrite(thumbPath, buf.Bytes(), 0644); err != nil {
		slog.Warn("save thumbnail failed", "path", thumbPath, "err", err)
		return
	}

	slog.Info("thumbnail generated", "path", thumbPath, "size", fmt.Sprintf("%dx%d", tw, th))
}

func fitDimensions(w, h, maxDim int) (int, int) {
	if w <= maxDim && h <= maxDim {
		return w, h
	}
	if w > h {
		return maxDim, h * maxDim / w
	}
	return w * maxDim / h, maxDim
}

// --- 扩展名校验 ---

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".avif": true, ".svg": true,
}

var videoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
	".webm": true, ".wmv": true, ".flv": true,
}

func isImageExt(ext string) bool { return imageExts[ext] }

func isAllowedUploadExt(scene, ext string) bool {
	switch scene {
	case "site-icon":
		return isImageExt(ext)
	case "site-attachment", "bm-attachment":
		return isImageExt(ext) || videoExts[ext] || ext == ".txt"
	case "note-image":
		return isImageExt(ext)
	default:
		return false
	}
}
