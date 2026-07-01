package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/png"

	"collections/internal/util"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const mediaThumbMaxDim = 300

// thumbnailHash 计算缩略图文件名：sha256(folderID/relPath)[:16]
func thumbnailHash(folderID, relPath string) string {
	h := sha256.Sum256([]byte(folderID + "/" + relPath))
	return hex.EncodeToString(h[:])[:16]
}

// thumbNames 返回静态缩略图和动画预览的文件名
func thumbNames(folderID, relPath string) (thumb, preview string) {
	hash := thumbnailHash(folderID, relPath)
	return hash + ".jpg", hash + ".preview.webp"
}

// ────────────────────── Image Thumbnail ──────────────────────

// generateImageThumbnail 生成静态 JPEG 缩略图（图片/GIF 首帧）。
// 返回原始图片的宽高。AVIF/SVG 等无纯 Go 解码器的格式会返回 error。
func generateImageThumbnail(srcPath, thumbPath string) (width, height int, err error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return 0, 0, fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image: %w", err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return 0, 0, fmt.Errorf("zero dimension image")
	}

	tw, th := fitDimensions(w, h, mediaThumbMaxDim)
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.BiLinear.Scale(dst, dst.Rect, src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return 0, 0, fmt.Errorf("encode jpeg: %w", err)
	}

	if err := util.AtomicWrite(thumbPath, buf.Bytes(), 0644); err != nil {
		return 0, 0, fmt.Errorf("write thumbnail: %w", err)
	}
	return w, h, nil
}

// ────────────────────── FFmpeg Operations ──────────────────────

// generateVideoThumbnail 使用 ffmpeg 提取视频第 1 秒帧作为 JPEG 缩略图
func generateVideoThumbnail(ffmpegPath, srcPath, thumbPath string) error {
	cmd := exec.Command(ffmpegPath,
		"-ss", "1",
		"-i", srcPath,
		"-frames:v", "1",
		"-q:v", "2",
		"-y",
		thumbPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w\n%s", err, limitOutput(out))
	}
	return nil
}

// generateAnimatedPreview 使用 ffmpeg 生成动画 WebP 预览（视频或 GIF）。
// 提取前 6 秒，每秒 2 帧，缩放到 320px 宽。
func generateAnimatedPreview(ffmpegPath, srcPath, previewPath string) error {
	cmd := exec.Command(ffmpegPath,
		"-ss", "0",
		"-t", "6",
		"-i", srcPath,
		"-vf", "fps=2,scale=320:-1",
		"-loop", "0",
		"-q:v", "75",
		"-y",
		previewPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg preview: %w\n%s", err, limitOutput(out))
	}
	return nil
}

// probeVideoMeta 使用 ffprobe 提取视频的宽高和时长
func probeVideoMeta(ffmpegPath, srcPath string) (width, height int, duration float64, err error) {
	probePath := deriveFFprobePath(ffmpegPath)

	cmd := exec.Command(probePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		srcPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe: %w", err)
	}

	var result struct {
		Streams []struct {
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			CodecType string `json:"codec_type"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return 0, 0, 0, fmt.Errorf("parse ffprobe output: %w", err)
	}

	for _, s := range result.Streams {
		if s.CodecType == "video" && s.Width > 0 {
			width = s.Width
			height = s.Height
			break
		}
	}

	if result.Format.Duration != "" {
		duration, _ = strconv.ParseFloat(result.Format.Duration, 64)
	}

	return width, height, duration, nil
}

// deriveFFprobePath 根据 ffmpeg 路径推导 ffprobe 路径
func deriveFFprobePath(ffmpegPath string) string {
	dir := filepath.Dir(ffmpegPath)
	name := "ffprobe"
	if runtime.GOOS == "windows" {
		name = "ffprobe.exe"
	}
	return filepath.Join(dir, name)
}

// removeThumbFiles 删除指定 hash 对应的缩略图和预览文件
func removeThumbFiles(thumbDir, thumbName, previewName string) {
	if thumbName != "" {
		os.Remove(filepath.Join(thumbDir, thumbName))
	}
	if previewName != "" {
		os.Remove(filepath.Join(thumbDir, previewName))
	}
}

func limitOutput(out []byte) string {
	s := strings.TrimSpace(string(out))
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
