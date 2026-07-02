package store

import (
	"testing"
	"time"

	"collections/internal/model"
)

func TestFolderCreate(t *testing.T) {
	s := newTestStore(t)
	folder, err := s.CreateFolder("/home/user/photos", "Photos")
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if folder.ID == "" {
		t.Error("folder ID should not be empty")
	}
	if folder.Path != "/home/user/photos" {
		t.Errorf("Path = %q, want %q", folder.Path, "/home/user/photos")
	}
	if folder.Name != "Photos" {
		t.Errorf("Name = %q, want %q", folder.Name, "Photos")
	}
	if folder.FileCount != 0 {
		t.Errorf("FileCount = %d, want 0", folder.FileCount)
	}
}

func TestFolderGetAndUpdate(t *testing.T) {
	s := newTestStore(t)
	folder, _ := s.CreateFolder("/home/user/photos", "Photos")

	got, err := s.GetFolder(folder.ID)
	if err != nil {
		t.Fatalf("GetFolder: %v", err)
	}
	if got.Name != "Photos" {
		t.Errorf("Name = %q, want %q", got.Name, "Photos")
	}

	got.Name = "My Photos"
	got.FileCount = 42
	now := time.Now()
	got.LastScanAt = &now
	if err := s.UpdateFolder(got); err != nil {
		t.Fatalf("UpdateFolder: %v", err)
	}

	updated, _ := s.GetFolder(folder.ID)
	if updated.Name != "My Photos" {
		t.Errorf("updated Name = %q, want %q", updated.Name, "My Photos")
	}
	if updated.FileCount != 42 {
		t.Errorf("updated FileCount = %d, want 42", updated.FileCount)
	}
}

func TestFolderDelete(t *testing.T) {
	s := newTestStore(t)
	folder, _ := s.CreateFolder("/home/user/photos", "Photos")

	if err := s.DeleteFolder(folder.ID); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	_, err := s.GetFolder(folder.ID)
	if err == nil {
		t.Error("GetFolder should fail after delete")
	}
}

func TestFolderList(t *testing.T) {
	s := newTestStore(t)
	s.CreateFolder("/a", "A")
	s.CreateFolder("/b", "B")
	s.CreateFolder("/c", "C")

	folders, err := s.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 3 {
		t.Errorf("folders len = %d, want 3", len(folders))
	}
}

func TestMediaMetaReadWrite(t *testing.T) {
	s := newTestStore(t)
	folder, _ := s.CreateFolder("/home/photos", "Photos")

	now := time.Now()
	meta := &model.MediaMeta{
		SchemaVersion: 1,
		FolderID:      folder.ID,
		Files: map[string]model.MediaFile{
			"vacation/beach.jpg": {
				MediaType: model.MediaTypeImage,
				Tags:      []string{"vacation", "beach"},
				FileSize:  1024000,
				ScannedAt: now,
				UpdatedAt: now,
			},
			"cats/meow.mp4": {
				MediaType: model.MediaTypeVideo,
				Tags:      []string{"cats"},
				FileSize:  5120000,
				ScannedAt: now,
				UpdatedAt: now,
			},
		},
	}

	if err := s.WriteMediaMeta(folder.ID, meta); err != nil {
		t.Fatalf("WriteMediaMeta: %v", err)
	}

	got, err := s.ReadMediaMeta(folder.ID)
	if err != nil {
		t.Fatalf("ReadMediaMeta: %v", err)
	}
	if len(got.Files) != 2 {
		t.Errorf("Files len = %d, want 2", len(got.Files))
	}

	beach, ok := got.Files["vacation/beach.jpg"]
	if !ok {
		t.Fatal("beach.jpg not found in meta")
	}
	if beach.MediaType != model.MediaTypeImage {
		t.Errorf("beach MediaType = %q, want %q", beach.MediaType, model.MediaTypeImage)
	}
	if beach.FileSize != 1024000 {
		t.Errorf("beach FileSize = %d, want 1024000", beach.FileSize)
	}
}

func TestTreeHashReadWrite(t *testing.T) {
	s := newTestStore(t)
	folder, _ := s.CreateFolder("/home/photos", "Photos")

	now := time.Now()
	size := int64(2048)
	th := &model.TreeHashFile{
		SchemaVersion: 1,
		FolderID:      folder.ID,
		Root: &model.TreeNode{
			Type: model.TreeNodeDir,
			Hash: "abc123",
			Children: map[string]*model.TreeNode{
				"photo.jpg": {
					Type:  model.TreeNodeFile,
					Hash:  "def456",
					Mtime: &now,
					Size:  &size,
				},
			},
		},
	}

	if err := s.WriteTreeHash(folder.ID, th); err != nil {
		t.Fatalf("WriteTreeHash: %v", err)
	}

	got, err := s.ReadTreeHash(folder.ID)
	if err != nil {
		t.Fatalf("ReadTreeHash: %v", err)
	}
	if got.Root == nil {
		t.Fatal("Root should not be nil")
	}
	if got.Root.Hash != "abc123" {
		t.Errorf("Root.Hash = %q, want %q", got.Root.Hash, "abc123")
	}
	child, ok := got.Root.Children["photo.jpg"]
	if !ok {
		t.Fatal("photo.jpg not found in tree")
	}
	if child.Hash != "def456" {
		t.Errorf("child Hash = %q, want %q", child.Hash, "def456")
	}
}

func TestMediaMetaReadNotFound(t *testing.T) {
	s := newTestStore(t)
	folder, _ := s.CreateFolder("/home/photos", "Photos")

	_, err := s.ReadMediaMeta(folder.ID)
	if err == nil {
		t.Error("ReadMediaMeta should fail when file does not exist")
	}
}
