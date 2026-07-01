package service

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ────────────────────── thumbnailHash ──────────────────────

func TestThumbnailHash_Deterministic(t *testing.T) {
	h1 := thumbnailHash("folder-id", "sub/photo.jpg")
	h2 := thumbnailHash("folder-id", "sub/photo.jpg")
	if h1 != h2 {
		t.Errorf("same input produces different hashes: %q vs %q", h1, h2)
	}
}

func TestThumbnailHash_Length(t *testing.T) {
	h := thumbnailHash("abc", "photo.jpg")
	if len(h) != 16 {
		t.Errorf("hash length = %d, want 16", len(h))
	}
}

func TestThumbnailHash_DiffersOnInput(t *testing.T) {
	h1 := thumbnailHash("folder-a", "photo.jpg")
	h2 := thumbnailHash("folder-b", "photo.jpg")
	if h1 == h2 {
		t.Error("different folderIDs should produce different hashes")
	}

	h3 := thumbnailHash("folder-a", "a.jpg")
	h4 := thumbnailHash("folder-a", "b.jpg")
	if h3 == h4 {
		t.Error("different relPaths should produce different hashes")
	}
}

// ────────────────────── thumbNames ──────────────────────

func TestThumbNames(t *testing.T) {
	thumb, preview := thumbNames("folder-id", "photo.jpg")
	hash := thumbnailHash("folder-id", "photo.jpg")

	if thumb != hash+".jpg" {
		t.Errorf("thumb = %q, want %q", thumb, hash+".jpg")
	}
	if preview != hash+".preview.webp" {
		t.Errorf("preview = %q, want %q", preview, hash+".preview.webp")
	}
}

// ────────────────────── generateImageThumbnail ──────────────────────

func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test jpeg: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
}

func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
}

func TestGenerateImageThumbnail_JPEG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.jpg")
	thumb := filepath.Join(dir, "thumb.jpg")

	createTestJPEG(t, src, 800, 600)

	w, h, err := generateImageThumbnail(src, thumb)
	if err != nil {
		t.Fatalf("generateImageThumbnail: %v", err)
	}
	if w != 800 || h != 600 {
		t.Errorf("dimensions = %dx%d, want 800x600", w, h)
	}
	if _, err := os.Stat(thumb); os.IsNotExist(err) {
		t.Error("thumbnail file should exist")
	}
}

func TestGenerateImageThumbnail_PNG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "image.png")
	thumb := filepath.Join(dir, "thumb.jpg")

	createTestPNG(t, src, 400, 300)

	w, h, err := generateImageThumbnail(src, thumb)
	if err != nil {
		t.Fatalf("generateImageThumbnail: %v", err)
	}
	if w != 400 || h != 300 {
		t.Errorf("dimensions = %dx%d, want 400x300", w, h)
	}
}

func TestGenerateImageThumbnail_SmallImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tiny.jpg")
	thumb := filepath.Join(dir, "thumb.jpg")

	createTestJPEG(t, src, 50, 50)

	w, h, err := generateImageThumbnail(src, thumb)
	if err != nil {
		t.Fatalf("generateImageThumbnail: %v", err)
	}
	if w != 50 || h != 50 {
		t.Errorf("dimensions = %dx%d, want 50x50", w, h)
	}
}

func TestGenerateImageThumbnail_LargeImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "large.jpg")
	thumb := filepath.Join(dir, "thumb.jpg")

	createTestJPEG(t, src, 4000, 3000)

	_, _, err := generateImageThumbnail(src, thumb)
	if err != nil {
		t.Fatalf("generateImageThumbnail: %v", err)
	}

	f, err := os.Open(thumb)
	if err != nil {
		t.Fatalf("open thumb: %v", err)
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode thumb config: %v", err)
	}
	if cfg.Width > mediaThumbMaxDim || cfg.Height > mediaThumbMaxDim {
		t.Errorf("thumb size %dx%d exceeds max %d", cfg.Width, cfg.Height, mediaThumbMaxDim)
	}
}

func TestGenerateImageThumbnail_NotFound(t *testing.T) {
	_, _, err := generateImageThumbnail("/nonexistent/path.jpg", "/tmp/out.jpg")
	if err == nil {
		t.Error("should error on nonexistent file")
	}
}

func TestGenerateImageThumbnail_InvalidImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bad.jpg")
	os.WriteFile(src, []byte("not an image"), 0644)

	_, _, err := generateImageThumbnail(src, filepath.Join(dir, "thumb.jpg"))
	if err == nil {
		t.Error("should error on invalid image data")
	}
}

// ────────────────────── removeThumbFiles ──────────────────────

func TestRemoveThumbFiles(t *testing.T) {
	dir := t.TempDir()
	thumb := filepath.Join(dir, "abc.jpg")
	preview := filepath.Join(dir, "abc.preview.webp")
	os.WriteFile(thumb, []byte("t"), 0644)
	os.WriteFile(preview, []byte("p"), 0644)

	removeThumbFiles(dir, "abc.jpg", "abc.preview.webp")

	if _, err := os.Stat(thumb); !os.IsNotExist(err) {
		t.Error("thumb should be removed")
	}
	if _, err := os.Stat(preview); !os.IsNotExist(err) {
		t.Error("preview should be removed")
	}
}

func TestRemoveThumbFiles_EmptyNames(t *testing.T) {
	dir := t.TempDir()
	removeThumbFiles(dir, "", "")
}

// ────────────────────── deriveFFprobePath ──────────────────────

func TestDeriveFFprobePath(t *testing.T) {
	p := deriveFFprobePath("/usr/local/bin/ffmpeg")
	expected := "/usr/local/bin/ffprobe"
	if p != expected {
		t.Errorf("probe path = %q, want %q", p, expected)
	}
}

// ────────────────────── ffmpeg-dependent tests ──────────────────────

func findFFmpeg(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available, skipping")
	}
	return path
}

func TestGenerateVideoThumbnail(t *testing.T) {
	ffmpeg := findFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "test.mp4")
	thumb := filepath.Join(dir, "thumb.jpg")

	createTestVideo(t, ffmpeg, src)

	if err := generateVideoThumbnail(ffmpeg, src, thumb); err != nil {
		t.Fatalf("generateVideoThumbnail: %v", err)
	}
	if _, err := os.Stat(thumb); os.IsNotExist(err) {
		t.Error("video thumbnail should exist")
	}
}

func TestGenerateAnimatedPreview(t *testing.T) {
	ffmpeg := findFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "test.mp4")
	preview := filepath.Join(dir, "preview.webp")

	createTestVideo(t, ffmpeg, src)

	if err := generateAnimatedPreview(ffmpeg, src, preview); err != nil {
		t.Fatalf("generateAnimatedPreview: %v", err)
	}
	if _, err := os.Stat(preview); os.IsNotExist(err) {
		t.Error("animated preview should exist")
	}
}

func TestProbeVideoMeta(t *testing.T) {
	ffmpeg := findFFmpeg(t)

	dir := t.TempDir()
	src := filepath.Join(dir, "test.mp4")
	createTestVideo(t, ffmpeg, src)

	w, h, dur, err := probeVideoMeta(ffmpeg, src)
	if err != nil {
		t.Fatalf("probeVideoMeta: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("dimensions = %dx%d, want positive", w, h)
	}
	if dur <= 0 {
		t.Errorf("duration = %f, want positive", dur)
	}
}

// createTestVideo generates a minimal 2-second test video via ffmpeg
func createTestVideo(t *testing.T, ffmpegPath, outPath string) {
	t.Helper()
	cmd := exec.Command(ffmpegPath,
		"-f", "lavfi",
		"-i", "color=c=blue:s=320x240:d=2",
		"-c:v", "libx264",
		"-t", "2",
		"-pix_fmt", "yuv420p",
		"-y",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create test video: %v\n%s", err, string(out))
	}
}
