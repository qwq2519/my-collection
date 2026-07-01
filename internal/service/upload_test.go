package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"collections/internal/model"
)

func TestUploadService_Validation(t *testing.T) {
	svc := &UploadService{Store: newTestStore(t)}

	tests := []struct {
		name string
		req  model.UploadFileReq
	}{
		{"empty scene", model.UploadFileReq{Scene: "", Filename: "a.png", Data: []byte("x")}},
		{"empty filename", model.UploadFileReq{Scene: "site-icon", Filename: "", Data: []byte("x")}},
		{"empty data", model.UploadFileReq{Scene: "site-icon", Filename: "a.png", Data: nil}},
	}
	for _, tt := range tests {
		_, err := svc.UploadFile(tt.req)
		if err == nil {
			t.Errorf("UploadFile[%s] should return error", tt.name)
		}
	}
}

func TestUploadService_FileSizeLimit(t *testing.T) {
	svc := &UploadService{Store: newTestStore(t)}
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
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

	_, err := svc.UploadFile(model.UploadFileReq{
		Scene: "unknown-scene", Filename: "a.png",
		EntityID: "test-id", Data: []byte("x"),
	})
	if err == nil {
		t.Error("UploadFile should reject unknown scene")
	}
}

func TestUploadService_SiteIcon(t *testing.T) {
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

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
	svc := &UploadService{Store: newTestStore(t)}

	if err := svc.DeleteAttachment("", "file.txt"); err == nil {
		t.Error("should reject empty entityID")
	}
	if err := svc.DeleteAttachment("id", ""); err == nil {
		t.Error("should reject empty filename")
	}
}
