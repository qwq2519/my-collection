package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"collections/internal/model"
)

func TestUploadService_Validation(t *testing.T) {
	svc := newUploadService(t)

	tests := []struct {
		name string
		req  model.UploadFileReq
	}{
		{"empty scene", model.UploadFileReq{Scene: "", Filename: "a.png", Data: []byte("x")}},
		{"empty filename", model.UploadFileReq{Scene: "site-icon", Filename: "", Data: []byte("x")}},
		{"empty data", model.UploadFileReq{Scene: "site-icon", Filename: "a.png", Data: nil}},
		{"empty site icon entity ID", model.UploadFileReq{Scene: "site-icon", Filename: "a.png", Data: []byte("x")}},
		{"empty attachment entity ID", model.UploadFileReq{Scene: "site-attachment", Filename: "a.txt", Data: []byte("x")}},
		{"empty note image entity ID", model.UploadFileReq{Scene: "note-image", Filename: "a.png", Data: []byte("x")}},
	}
	for _, tt := range tests {
		_, err := svc.UploadFile(tt.req)
		if err == nil {
			t.Errorf("UploadFile[%s] should return error", tt.name)
		}
	}
}

func TestUploadService_FileSizeLimit(t *testing.T) {
	svc := newUploadService(t)
	bigData := make([]byte, 11<<20) // 11 MB

	_, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-icon", Filename: "big.png",
		EntityID: "test-id", Data: bigData,
	})
	if err == nil {
		t.Error("UploadFile should reject files exceeding 10MB")
	}
}

func TestUploadService_UnsupportedExtension(t *testing.T) {
	svc := newUploadService(t)

	_, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-icon", Filename: "icon.exe",
		EntityID: "test-id", Data: []byte("x"),
	})
	if err == nil {
		t.Error("UploadFile should reject unsupported extension for site-icon")
	}

	_, err = svc.UploadFile(model.UploadFileReq{
		Scene: "note-image", Filename: "doc.pdf",
		EntityID: "test-id", Data: []byte("x"),
	})
	if err == nil {
		t.Error("UploadFile should reject .pdf for note-image")
	}
}

func TestUploadService_UnknownScene(t *testing.T) {
	svc := newUploadService(t)

	_, err := svc.UploadFile(model.UploadFileReq{
		Scene: "unknown-scene", Filename: "a.png",
		EntityID: "test-id", Data: []byte("x"),
	})
	if err == nil {
		t.Error("UploadFile should reject unknown scene")
	}
}

func TestUploadService_SiteIcon(t *testing.T) {
	svc := newUploadService(t)

	result, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-icon", Filename: "icon.png",
		EntityID: "github.com", Data: []byte("fake-png"),
	})
	if err != nil {
		t.Fatalf("UploadFile site-icon: %v", err)
	}
	if result.Path != "github.com.png" {
		t.Errorf("Path = %q, want %q", result.Path, "github.com.png")
	}

	saved := filepath.Join(svc.Store.PersistDir(), "url-assets", "icons", "github.com.png")
	data, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "fake-png" {
		t.Errorf("content = %q, want %q", data, "fake-png")
	}
}

func TestUploadService_NoteImage(t *testing.T) {
	svc := newUploadService(t)

	result, err := svc.UploadFile(model.UploadFileReq{
		Scene: "note-image", Filename: "photo.jpg",
		EntityID: "note-123", Data: []byte("fake-jpg"),
	})
	if err != nil {
		t.Fatalf("UploadFile note-image: %v", err)
	}
	if !strings.HasPrefix(result.Path, "note-images/note-123/") {
		t.Errorf("Path = %q, want prefix 'note-images/note-123/'", result.Path)
	}
	if !strings.HasSuffix(result.Path, ".jpg") {
		t.Errorf("Path = %q, should end with .jpg", result.Path)
	}
}

func TestUploadService_Attachment(t *testing.T) {
	svc := newUploadService(t)

	result, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-attachment", Filename: "readme.txt",
		EntityID: "site-abc", Data: []byte("content"),
	})
	if err != nil {
		t.Fatalf("UploadFile attachment: %v", err)
	}
	if result.Path != "readme.txt" {
		t.Errorf("Path = %q, want %q", result.Path, "readme.txt")
	}
}

func TestUploadService_AttachmentAllowedFormats(t *testing.T) {
	svc := newUploadService(t)

	allowed := []string{"test.png", "test.mp4", "test.txt", "test.webm"}
	for _, f := range allowed {
		_, err := svc.UploadFile(model.UploadFileReq{
			Scene: "bm-attachment", Filename: f,
			EntityID: "bm-1", Data: []byte("x"),
		})
		if err != nil {
			t.Errorf("attachment %q should be allowed, got: %v", f, err)
		}
	}
}

func TestUploadService_DeleteAttachment(t *testing.T) {
	svc := newUploadService(t)

	svc.UploadFile(model.UploadFileReq{
		Scene: "site-attachment", Filename: "doc.txt",
		EntityID: "ent-1", Data: []byte("hello"),
	})

	err := svc.DeleteAttachment("ent-1", "doc.txt")
	if err != nil {
		t.Fatalf("DeleteAttachment: %v", err)
	}

	path := filepath.Join(svc.Store.PersistDir(), "url-assets", "attachments", "ent-1", "doc.txt")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestUploadService_DeleteAttachmentValidation(t *testing.T) {
	svc := newUploadService(t)

	if err := svc.DeleteAttachment("", "file.txt"); err == nil {
		t.Error("should reject empty entityID")
	}
	if err := svc.DeleteAttachment("id", ""); err == nil {
		t.Error("should reject empty filename")
	}
}

// ────────────────────── fitDimensions ──────────────────────

func TestFitDimensions(t *testing.T) {
	tests := []struct {
		name         string
		w, h, maxDim int
		wantW, wantH int
	}{
		{"smaller than max", 100, 80, 300, 100, 80},
		{"equal to max", 300, 300, 300, 300, 300},
		{"landscape", 600, 300, 300, 300, 150},
		{"portrait", 300, 600, 300, 150, 300},
		{"square larger", 500, 500, 300, 300, 300},
		{"wide aspect", 1200, 100, 300, 300, 25},
		{"tall aspect", 100, 1200, 300, 25, 300},
		{"very small dimension clamped", 1000, 1, 300, 300, 1},
	}
	for _, tt := range tests {
		gotW, gotH := fitDimensions(tt.w, tt.h, tt.maxDim)
		if gotW != tt.wantW || gotH != tt.wantH {
			t.Errorf("fitDimensions[%s](%d,%d,%d) = (%d,%d), want (%d,%d)",
				tt.name, tt.w, tt.h, tt.maxDim, gotW, gotH, tt.wantW, tt.wantH)
		}
	}
}

// ────────────────────── generateThumbnail ──────────────────────

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestUploadService_AttachmentImageThumbnail(t *testing.T) {
	svc := newUploadService(t)

	pngData := makePNG(t, 600, 400)

	result, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-attachment", Filename: "photo.png",
		EntityID: "ent-1", Data: pngData,
	})
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if result.Path != "photo.png" {
		t.Errorf("Path = %q, want %q", result.Path, "photo.png")
	}

	thumbPath := filepath.Join(svc.Store.PersistDir(), "url-assets", "attachments", "ent-1", "photo.png.thumb.jpg")
	info, err := os.Stat(thumbPath)
	if err != nil {
		t.Fatalf("thumbnail should exist at %s: %v", thumbPath, err)
	}
	if info.Size() == 0 {
		t.Error("thumbnail should not be empty")
	}
}

func TestUploadService_NonImageAttachmentNoThumbnail(t *testing.T) {
	svc := newUploadService(t)

	if _, err := svc.UploadFile(model.UploadFileReq{
		Scene: "site-attachment", Filename: "readme.txt",
		EntityID: "ent-2", Data: []byte("hello"),
	}); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	thumbPath := filepath.Join(svc.Store.PersistDir(), "url-assets", "attachments", "ent-2", "readme.txt.thumb.jpg")
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Error("text file should not have a thumbnail")
	}
}
