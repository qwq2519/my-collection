package service

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"collections/internal/model"
)

// ────────────────────── ListFolders ──────────────────────

func TestMediaService_ListFoldersEmpty(t *testing.T) {
	svc := newMediaService(t)
	folders, err := svc.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 0 {
		t.Errorf("folders len = %d, want 0", len(folders))
	}
}

func TestMediaService_ListFoldersAfterAdd(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	if _, err := svc.AddFolder(model.AddFolderReq{Path: dir, Name: "test"}); err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	folders, err := svc.ListFolders()
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if len(folders) != 1 {
		t.Fatalf("folders len = %d, want 1", len(folders))
	}
	if folders[0].Name != "test" {
		t.Errorf("Name = %q, want %q", folders[0].Name, "test")
	}
}

// ────────────────────── AddFolder ──────────────────────

func TestMediaService_AddFolderValidation(t *testing.T) {
	svc := newMediaService(t)

	tests := []struct {
		name string
		req  model.AddFolderReq
	}{
		{"empty path", model.AddFolderReq{Path: ""}},
		{"whitespace path", model.AddFolderReq{Path: "   "}},
		{"nonexistent path", model.AddFolderReq{Path: "/nonexistent/path/abc123"}},
	}
	for _, tt := range tests {
		_, err := svc.AddFolder(tt.req)
		if err == nil {
			t.Errorf("AddFolder[%s] should return error", tt.name)
		}
	}
}

func TestMediaService_AddFolderNotDir(t *testing.T) {
	svc := newMediaService(t)

	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0644)

	_, err := svc.AddFolder(model.AddFolderReq{Path: tmpFile})
	if err == nil {
		t.Error("AddFolder should reject non-directory path")
	}
}

func TestMediaService_AddFolderAutoName(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}
	if folder.Name != filepath.Base(dir) {
		t.Errorf("auto name = %q, want %q", folder.Name, filepath.Base(dir))
	}
}

func TestMediaService_AddFolderCreatesDir(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir, Name: "photos"})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	thumbDir := filepath.Join(svc.Store.PersistDir(), "media-folders", folder.ID, "thumbnails")
	if _, err := os.Stat(thumbDir); os.IsNotExist(err) {
		t.Error("thumbnails dir should be created")
	}
}

func TestMediaService_AddFolderDuplicate(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	if _, err := svc.AddFolder(model.AddFolderReq{Path: dir}); err != nil {
		t.Fatalf("first AddFolder: %v", err)
	}

	_, err := svc.AddFolder(model.AddFolderReq{Path: dir})
	if err == nil {
		t.Error("AddFolder should reject duplicate path")
	}
}

func TestMediaService_AddFolderNestingChild(t *testing.T) {
	svc := newMediaService(t)

	parent := t.TempDir()
	child := filepath.Join(parent, "sub")
	os.Mkdir(child, 0755)

	if _, err := svc.AddFolder(model.AddFolderReq{Path: parent}); err != nil {
		t.Fatalf("add parent: %v", err)
	}

	_, err := svc.AddFolder(model.AddFolderReq{Path: child})
	if err == nil {
		t.Error("AddFolder should reject child of existing folder")
	}
}

func TestMediaService_AddFolderNestingParent(t *testing.T) {
	svc := newMediaService(t)

	parent := t.TempDir()
	child := filepath.Join(parent, "sub")
	os.Mkdir(child, 0755)

	if _, err := svc.AddFolder(model.AddFolderReq{Path: child}); err != nil {
		t.Fatalf("add child: %v", err)
	}

	_, err := svc.AddFolder(model.AddFolderReq{Path: parent})
	if err == nil {
		t.Error("AddFolder should reject parent of existing folder")
	}
}

func TestMediaService_AddFolderSiblingAllowed(t *testing.T) {
	svc := newMediaService(t)

	base := t.TempDir()
	sibA := filepath.Join(base, "a")
	sibB := filepath.Join(base, "b")
	os.Mkdir(sibA, 0755)
	os.Mkdir(sibB, 0755)

	if _, err := svc.AddFolder(model.AddFolderReq{Path: sibA}); err != nil {
		t.Fatalf("add sibling a: %v", err)
	}
	if _, err := svc.AddFolder(model.AddFolderReq{Path: sibB}); err != nil {
		t.Errorf("add sibling b should succeed: %v", err)
	}
}

// ────────────────────── RemoveFolder ──────────────────────

func TestMediaService_RemoveFolderValidation(t *testing.T) {
	svc := newMediaService(t)
	err := svc.RemoveFolder("")
	if err == nil {
		t.Error("RemoveFolder should reject empty ID")
	}
}

func TestMediaService_RemoveFolderNotFound(t *testing.T) {
	svc := newMediaService(t)
	err := svc.RemoveFolder("nonexistent-id")
	if err == nil {
		t.Error("RemoveFolder should fail for nonexistent folder")
	}
}

func TestMediaService_RemoveFolderCleansUp(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir, Name: "test"})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	persistDir := filepath.Join(svc.Store.PersistDir(), "media-folders", folder.ID)
	if _, err := os.Stat(persistDir); os.IsNotExist(err) {
		t.Fatal("persist dir should exist after add")
	}

	if err := svc.RemoveFolder(folder.ID); err != nil {
		t.Fatalf("RemoveFolder: %v", err)
	}

	if _, err := os.Stat(persistDir); !os.IsNotExist(err) {
		t.Error("persist dir should be removed after RemoveFolder")
	}

	folders, _ := svc.ListFolders()
	if len(folders) != 0 {
		t.Errorf("folders count = %d, want 0", len(folders))
	}
}

func TestMediaService_RemoveFolderTagCleanup(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir, Name: "test"})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	meta := &model.MediaMeta{
		SchemaVersion: 1,
		FolderID:      folder.ID,
		Files: map[string]model.MediaFile{
			"photo.jpg": {
				MediaType: "image",
				Tags:      []string{"landscape", "travel"},
			},
			"video.mp4": {
				MediaType: "video",
				Tags:      []string{"travel"},
			},
		},
	}
	if err := svc.Store.WriteMediaMeta(folder.ID, meta); err != nil {
		t.Fatalf("WriteMediaMeta: %v", err)
	}

	if err := svc.Store.AdjustTagCount("media_tag", "landscape", 1); err != nil {
		t.Fatalf("AdjustTagCount: %v", err)
	}
	if err := svc.Store.AdjustTagCount("media_tag", "travel", 2); err != nil {
		t.Fatalf("AdjustTagCount: %v", err)
	}

	if err := svc.RemoveFolder(folder.ID); err != nil {
		t.Fatalf("RemoveFolder: %v", err)
	}

	tag, _ := svc.Store.GetTag("media_tag", "landscape")
	if tag != nil && tag.Count > 0 {
		t.Errorf("landscape count = %d, want 0", tag.Count)
	}

	tag, _ = svc.Store.GetTag("media_tag", "travel")
	if tag != nil && tag.Count > 0 {
		t.Errorf("travel count = %d, want 0", tag.Count)
	}
}

// ────────────────────── UpdateFolderPath ──────────────────────

func TestMediaService_UpdateFolderPathValidation(t *testing.T) {
	svc := newMediaService(t)

	tests := []struct {
		name string
		req  model.UpdateFolderPathReq
	}{
		{"empty ID", model.UpdateFolderPathReq{ID: "", Path: "/tmp"}},
		{"empty path", model.UpdateFolderPathReq{ID: "x", Path: ""}},
		{"whitespace path", model.UpdateFolderPathReq{ID: "x", Path: "   "}},
		{"nonexistent path", model.UpdateFolderPathReq{ID: "x", Path: "/nonexistent/abc123"}},
	}
	for _, tt := range tests {
		_, err := svc.UpdateFolderPath(tt.req)
		if err == nil {
			t.Errorf("UpdateFolderPath[%s] should return error", tt.name)
		}
	}
}

func TestMediaService_UpdateFolderPathSuccess(t *testing.T) {
	svc := newMediaService(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: oldDir, Name: "test"})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	updated, err := svc.UpdateFolderPath(model.UpdateFolderPathReq{
		ID:   folder.ID,
		Path: newDir,
	})
	if err != nil {
		t.Fatalf("UpdateFolderPath: %v", err)
	}

	absNew, _ := filepath.Abs(newDir)
	if updated.Path != absNew {
		t.Errorf("Path = %q, want %q", updated.Path, absNew)
	}
}

func TestMediaService_UpdateFolderPathNesting(t *testing.T) {
	svc := newMediaService(t)

	dirA := t.TempDir()
	dirB := t.TempDir()
	childB := filepath.Join(dirB, "sub")
	os.Mkdir(childB, 0755)

	if _, err := svc.AddFolder(model.AddFolderReq{Path: dirA}); err != nil {
		t.Fatalf("add folder A: %v", err)
	}
	folderB, err := svc.AddFolder(model.AddFolderReq{Path: dirB})
	if err != nil {
		t.Fatalf("add folder B: %v", err)
	}

	_, err = svc.UpdateFolderPath(model.UpdateFolderPathReq{
		ID:   folderB.ID,
		Path: filepath.Join(dirA, "sub"),
	})
	if err == nil {
		t.Error("UpdateFolderPath should reject path nesting with other folders")
	}
}

func TestMediaService_UpdateFolderPathSameLocation(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()

	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}

	_, err = svc.UpdateFolderPath(model.UpdateFolderPathReq{
		ID:   folder.ID,
		Path: dir,
	})
	if err != nil {
		t.Errorf("update to same path should succeed: %v", err)
	}
}

// ────────────────────── ScanFolder ──────────────────────

func setupFolderWithFiles(t *testing.T, svc *MediaService, files map[string][]byte) *model.MediaFolder {
	t.Helper()
	dir := t.TempDir()
	for relPath, content := range files {
		absPath := filepath.Join(dir, filepath.FromSlash(relPath))
		os.MkdirAll(filepath.Dir(absPath), 0755)
		if err := os.WriteFile(absPath, content, 0644); err != nil {
			t.Fatalf("write %s: %v", relPath, err)
		}
	}
	folder, err := svc.AddFolder(model.AddFolderReq{Path: dir, Name: "test"})
	if err != nil {
		t.Fatalf("AddFolder: %v", err)
	}
	return folder
}

func TestMediaService_ScanFolderValidation(t *testing.T) {
	svc := newMediaService(t)
	_, err := svc.ScanFolder("")
	if err == nil {
		t.Error("ScanFolder should reject empty ID")
	}
}

func TestMediaService_ScanFolderFirstScan(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 100, 80),
		"image.png": createPNGBytes(t, 50, 50),
	})

	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Added != 2 {
		t.Errorf("Added = %d, want 2", result.Added)
	}
	if result.Removed != 0 {
		t.Errorf("Removed = %d, want 0", result.Removed)
	}

	meta, err := svc.Store.ReadMediaMeta(folder.ID)
	if err != nil {
		t.Fatalf("ReadMediaMeta: %v", err)
	}
	if len(meta.Files) != 2 {
		t.Errorf("meta files = %d, want 2", len(meta.Files))
	}

	f := meta.Files["photo.jpg"]
	if f.MediaType != "image" {
		t.Errorf("media_type = %q, want image", f.MediaType)
	}
	if f.Width == nil || *f.Width != 100 {
		t.Errorf("width = %v, want 100", f.Width)
	}
	if f.Height == nil || *f.Height != 80 {
		t.Errorf("height = %v, want 80", f.Height)
	}

	updated, _ := svc.Store.GetFolder(folder.ID)
	if updated.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", updated.FileCount)
	}
	if updated.LastScanAt.IsZero() {
		t.Error("LastScanAt should be set")
	}
}

func TestMediaService_ScanFolderNoChange(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"a.jpg": createJPEGBytes(t, 10, 10),
	})

	svc.ScanFolder(folder.ID)

	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("second ScanFolder: %v", err)
	}
	if result.Added+result.Removed+result.Modified != 0 {
		t.Errorf("no-change scan: added=%d removed=%d modified=%d, want all 0",
			result.Added, result.Removed, result.Modified)
	}
}

func TestMediaService_ScanFolderFileAdded(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "old.jpg"), createJPEGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	svc.ScanFolder(folder.ID)

	os.WriteFile(filepath.Join(dir, "new.png"), createPNGBytes(t, 20, 20), 0644)
	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("Added = %d, want 1", result.Added)
	}

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	if len(meta.Files) != 2 {
		t.Errorf("meta files = %d, want 2", len(meta.Files))
	}
}

func TestMediaService_ScanFolderFileRemoved(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.jpg"), createJPEGBytes(t, 10, 10), 0644)
	os.WriteFile(filepath.Join(dir, "delete.png"), createPNGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	svc.ScanFolder(folder.ID)

	os.Remove(filepath.Join(dir, "delete.png"))
	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Removed != 1 {
		t.Errorf("Removed = %d, want 1", result.Removed)
	}

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	if len(meta.Files) != 1 {
		t.Errorf("meta files = %d, want 1", len(meta.Files))
	}
	if _, exists := meta.Files["delete.png"]; exists {
		t.Error("deleted file should not be in meta")
	}

	updated, _ := svc.Store.GetFolder(folder.ID)
	if updated.FileCount != 1 {
		t.Errorf("FileCount = %d, want 1", updated.FileCount)
	}
}

func TestMediaService_ScanFolderTagCleanupOnRemove(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tagged.jpg"), createJPEGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	svc.ScanFolder(folder.ID)

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	f := meta.Files["tagged.jpg"]
	f.Tags = []string{"landscape", "nature"}
	meta.Files["tagged.jpg"] = f
	svc.Store.WriteMediaMeta(folder.ID, meta)
	svc.Store.AdjustTagCount("media_tag", "landscape", 1)
	svc.Store.AdjustTagCount("media_tag", "nature", 1)

	os.Remove(filepath.Join(dir, "tagged.jpg"))
	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Removed != 1 {
		t.Errorf("Removed = %d, want 1", result.Removed)
	}

	tag, _ := svc.Store.GetTag("media_tag", "landscape")
	if tag != nil && tag.Count > 0 {
		t.Errorf("landscape count = %d, want 0", tag.Count)
	}
	tag, _ = svc.Store.GetTag("media_tag", "nature")
	if tag != nil && tag.Count > 0 {
		t.Errorf("nature count = %d, want 0", tag.Count)
	}
}

func TestMediaService_ScanFolderPreservesUserData(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "photo.jpg")
	os.WriteFile(f, createJPEGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	svc.ScanFolder(folder.ID)

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	file := meta.Files["photo.jpg"]
	file.Tags = []string{"vacation"}
	file.Description = "Beach photo"
	meta.Files["photo.jpg"] = file
	svc.Store.WriteMediaMeta(folder.ID, meta)

	time.Sleep(10 * time.Millisecond)
	os.WriteFile(f, createJPEGBytes(t, 20, 20), 0644)

	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Modified != 1 {
		t.Errorf("Modified = %d, want 1", result.Modified)
	}

	meta, _ = svc.Store.ReadMediaMeta(folder.ID)
	updated := meta.Files["photo.jpg"]
	if len(updated.Tags) != 1 || updated.Tags[0] != "vacation" {
		t.Errorf("tags = %v, want [vacation]", updated.Tags)
	}
	if updated.Description != "Beach photo" {
		t.Errorf("description = %q, want %q", updated.Description, "Beach photo")
	}
}

func TestMediaService_ScanFolderSubDir(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "deep.jpg"), createJPEGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	result, err := svc.ScanFolder(folder.ID)
	if err != nil {
		t.Fatalf("ScanFolder: %v", err)
	}
	if result.Added != 1 {
		t.Errorf("Added = %d, want 1", result.Added)
	}

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	if _, ok := meta.Files["sub/deep.jpg"]; !ok {
		t.Error("sub/deep.jpg should be in meta")
	}
}

func TestMediaService_ScanFolderGeneratesThumbnail(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 200, 150),
	})

	svc.ScanFolder(folder.ID)

	meta, _ := svc.Store.ReadMediaMeta(folder.ID)
	f := meta.Files["photo.jpg"]
	if f.Thumbnail == "" {
		t.Error("thumbnail should be set")
	}

	thumbPath := filepath.Join(svc.Store.PersistDir(), "media-folders", folder.ID, "thumbnails", f.Thumbnail)
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Error("thumbnail file should exist on disk")
	}
}

func TestMediaService_ScanAllFolders(t *testing.T) {
	svc := newMediaService(t)
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "a.jpg"), createJPEGBytes(t, 10, 10), 0644)
	os.WriteFile(filepath.Join(dir2, "b.png"), createPNGBytes(t, 10, 10), 0644)

	svc.AddFolder(model.AddFolderReq{Path: dir1, Name: "one"})
	svc.AddFolder(model.AddFolderReq{Path: dir2, Name: "two"})

	results, err := svc.ScanAllFolders()
	if err != nil {
		t.Fatalf("ScanAllFolders: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results len = %d, want 2", len(results))
	}

	totalAdded := 0
	for _, r := range results {
		totalAdded += r.Added
	}
	if totalAdded != 2 {
		t.Errorf("total added = %d, want 2", totalAdded)
	}
}

// ────────────────────── mediaBleveFields ──────────────────────

func TestMediaBleveFields(t *testing.T) {
	file := model.MediaFile{
		MediaType:   "image",
		Tags:        []string{"landscape"},
		Description: "test",
	}
	fields := mediaBleveFields("folder-id", "sub/photo.jpg", file)
	if fields["_type"] != "media" {
		t.Errorf("_type = %v, want media", fields["_type"])
	}
	if fields["folder_id"] != "folder-id" {
		t.Errorf("folder_id = %v", fields["folder_id"])
	}
	if fields["filename"] != "photo.jpg" {
		t.Errorf("filename = %v, want photo.jpg", fields["filename"])
	}
	if fields["media_type"] != "image" {
		t.Errorf("media_type = %v", fields["media_type"])
	}
}

// ────────────────────── Test Image Helpers ──────────────────────

func createJPEGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func createPNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// ────────────────────── ListMediaFiles ──────────────────────

func TestMediaService_ListMediaFiles(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"a.jpg": createJPEGBytes(t, 10, 10),
		"b.png": createPNGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	result, err := svc.ListMediaFiles(model.MediaListReq{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListMediaFiles: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	for _, item := range result.Items {
		if item.ThumbnailURL == "" && item.Thumbnail != "" {
			t.Errorf("ThumbnailURL should be filled for %s", item.RelPath)
		}
	}
}

func TestMediaService_ListMediaFilesFilterType(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "img.jpg"), createJPEGBytes(t, 10, 10), 0644)
	os.WriteFile(filepath.Join(dir, "sound.mp3"), []byte("fake mp3"), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})
	svc.ScanFolder(folder.ID)

	result, err := svc.ListMediaFiles(model.MediaListReq{
		Page: 1, PageSize: 10, MediaType: "image",
	})
	if err != nil {
		t.Fatalf("ListMediaFiles: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1 (only images)", result.Total)
	}
}

// ────────────────────── GetMediaFile ──────────────────────

func TestMediaService_GetMediaFile(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 100, 80),
	})
	svc.ScanFolder(folder.ID)

	item, err := svc.GetMediaFile(folder.ID, "photo.jpg")
	if err != nil {
		t.Fatalf("GetMediaFile: %v", err)
	}
	if item.MediaType != "image" {
		t.Errorf("media_type = %q, want image", item.MediaType)
	}
	if item.ThumbnailURL == "" {
		t.Error("ThumbnailURL should be set")
	}
	if item.FolderID != folder.ID {
		t.Errorf("FolderID = %q, want %q", item.FolderID, folder.ID)
	}
}

func TestMediaService_GetMediaFileValidation(t *testing.T) {
	svc := newMediaService(t)

	if _, err := svc.GetMediaFile("", "photo.jpg"); err == nil {
		t.Error("should reject empty folder ID")
	}
	if _, err := svc.GetMediaFile("x", ""); err == nil {
		t.Error("should reject empty relPath")
	}
}

func TestMediaService_GetMediaFileNotFound(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"a.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	_, err := svc.GetMediaFile(folder.ID, "nonexistent.jpg")
	if err == nil {
		t.Error("should error for nonexistent file")
	}
}

// ────────────────────── UpdateMediaFile ──────────────────────

func TestMediaService_UpdateMediaFileTags(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	tags := []string{"landscape", "Nature"}
	item, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID: folder.ID,
		RelPath:  "photo.jpg",
		Tags:     &tags,
	})
	if err != nil {
		t.Fatalf("UpdateMediaFile: %v", err)
	}
	if len(item.Tags) != 2 {
		t.Errorf("tags = %v, want 2 items", item.Tags)
	}
	if item.Tags[0] != "landscape" || item.Tags[1] != "nature" {
		t.Errorf("tags = %v, want [landscape nature]", item.Tags)
	}

	tag, _ := svc.Store.GetTag("media_tag", "landscape")
	if tag == nil || tag.Count != 1 {
		t.Errorf("landscape count = %v, want 1", tag)
	}
}

func TestMediaService_UpdateMediaFileTagsReplacesOld(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	oldTags := []string{"old-tag"}
	svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID: folder.ID, RelPath: "photo.jpg",
		Tags: &oldTags,
	})

	newTags := []string{"new-tag"}
	svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID: folder.ID, RelPath: "photo.jpg",
		Tags: &newTags,
	})

	item, _ := svc.GetMediaFile(folder.ID, "photo.jpg")
	if len(item.Tags) != 1 || item.Tags[0] != "new-tag" {
		t.Errorf("tags = %v, want [new-tag]", item.Tags)
	}

	tag, _ := svc.Store.GetTag("media_tag", "old-tag")
	if tag != nil && tag.Count > 0 {
		t.Errorf("old-tag count = %d, want 0", tag.Count)
	}
	tag, _ = svc.Store.GetTag("media_tag", "new-tag")
	if tag == nil || tag.Count != 1 {
		t.Errorf("new-tag count = %v, want 1", tag)
	}
}

func TestMediaService_UpdateMediaFileDescription(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	desc := "Beautiful sunset"
	item, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID:    folder.ID,
		RelPath:     "photo.jpg",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("UpdateMediaFile: %v", err)
	}
	if item.Description != "Beautiful sunset" {
		t.Errorf("description = %q, want %q", item.Description, "Beautiful sunset")
	}

	got, _ := svc.GetMediaFile(folder.ID, "photo.jpg")
	if got.Description != "Beautiful sunset" {
		t.Errorf("persisted description = %q", got.Description)
	}
}

func TestMediaService_UpdateMediaFileTagsAndDesc(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	tags := []string{"travel"}
	desc := "Tokyo tower"
	item, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID:    folder.ID,
		RelPath:     "photo.jpg",
		Tags:        &tags,
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("UpdateMediaFile: %v", err)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "travel" {
		t.Errorf("tags = %v, want [travel]", item.Tags)
	}
	if item.Description != "Tokyo tower" {
		t.Errorf("description = %q", item.Description)
	}
}

func TestMediaService_UpdateMediaFileNoop(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"photo.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	item, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID: folder.ID, RelPath: "photo.jpg",
	})
	if err != nil {
		t.Fatalf("noop UpdateMediaFile: %v", err)
	}
	if item == nil {
		t.Error("should return current item on noop")
	}
}

func TestMediaService_UpdateMediaFileValidation(t *testing.T) {
	svc := newMediaService(t)

	if _, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{FolderID: ""}); err == nil {
		t.Error("should reject empty folder ID")
	}
	if _, err := svc.UpdateMediaFile(model.UpdateMediaFileReq{FolderID: "x", RelPath: ""}); err == nil {
		t.Error("should reject empty relPath")
	}
}

// ────────────────────── BatchUpdateMediaTags ──────────────────────

func TestMediaService_BatchUpdateMediaTags(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"a.jpg": createJPEGBytes(t, 10, 10),
		"b.png": createPNGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	err := svc.BatchUpdateMediaTags(model.BatchUpdateMediaTagsReq{
		FolderID: folder.ID,
		RelPaths: []string{"a.jpg", "b.png"},
		Tags:     []string{"batch-tag"},
	})
	if err != nil {
		t.Fatalf("BatchUpdateMediaTags: %v", err)
	}

	a, _ := svc.GetMediaFile(folder.ID, "a.jpg")
	if len(a.Tags) != 1 || a.Tags[0] != "batch-tag" {
		t.Errorf("a.jpg tags = %v, want [batch-tag]", a.Tags)
	}
	b, _ := svc.GetMediaFile(folder.ID, "b.png")
	if len(b.Tags) != 1 || b.Tags[0] != "batch-tag" {
		t.Errorf("b.png tags = %v, want [batch-tag]", b.Tags)
	}

	tag, _ := svc.Store.GetTag("media_tag", "batch-tag")
	if tag == nil || tag.Count != 2 {
		t.Errorf("batch-tag count = %v, want 2", tag)
	}
}

func TestMediaService_BatchUpdateMediaTagsDedup(t *testing.T) {
	svc := newMediaService(t)
	folder := setupFolderWithFiles(t, svc, map[string][]byte{
		"a.jpg": createJPEGBytes(t, 10, 10),
	})
	svc.ScanFolder(folder.ID)

	existingTags := []string{"existing"}
	svc.UpdateMediaFile(model.UpdateMediaFileReq{
		FolderID: folder.ID, RelPath: "a.jpg",
		Tags: &existingTags,
	})

	err := svc.BatchUpdateMediaTags(model.BatchUpdateMediaTagsReq{
		FolderID: folder.ID,
		RelPaths: []string{"a.jpg"},
		Tags:     []string{"existing"},
	})
	if err != nil {
		t.Fatalf("BatchUpdateMediaTags: %v", err)
	}

	a, _ := svc.GetMediaFile(folder.ID, "a.jpg")
	if len(a.Tags) != 1 {
		t.Errorf("tags = %v, want [existing] (no dup)", a.Tags)
	}
}

func TestMediaService_BatchUpdateMediaTagsEmpty(t *testing.T) {
	svc := newMediaService(t)
	err := svc.BatchUpdateMediaTags(model.BatchUpdateMediaTagsReq{
		FolderID: "x", RelPaths: nil, Tags: nil,
	})
	if err != nil {
		t.Errorf("empty batch should return nil: %v", err)
	}
}

// ────────────────────── OpenInExplorer ──────────────────────

func TestMediaService_OpenInExplorerValidation(t *testing.T) {
	svc := newMediaService(t)
	if err := svc.OpenInExplorer("", "photo.jpg"); err == nil {
		t.Error("should reject empty folder ID")
	}
	if err := svc.OpenInExplorer("x", ""); err == nil {
		t.Error("should reject empty relPath")
	}
}

func TestMediaService_OpenInExplorerBuildPath(t *testing.T) {
	svc := newMediaService(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "photo.jpg")
	os.WriteFile(f, createJPEGBytes(t, 10, 10), 0644)

	folder, _ := svc.AddFolder(model.AddFolderReq{Path: dir})

	var capturedCmd string
	var capturedArgs []string
	origExec := execCommand
	execCommand = func(name string, args ...string) error {
		capturedCmd = name
		capturedArgs = args
		return nil
	}
	defer func() { execCommand = origExec }()

	if err := svc.OpenInExplorer(folder.ID, "photo.jpg"); err != nil {
		t.Fatalf("OpenInExplorer: %v", err)
	}

	if capturedCmd == "" {
		t.Fatal("exec should have been called")
	}
	absExpected := filepath.Join(dir, "photo.jpg")
	found := false
	for _, a := range capturedArgs {
		if a == absExpected || a == "-R" {
			found = true
		}
	}
	if !found && len(capturedArgs) > 0 {
		lastArg := capturedArgs[len(capturedArgs)-1]
		if lastArg != absExpected && lastArg != filepath.Dir(absExpected) {
			t.Errorf("args = %v, should contain %q", capturedArgs, absExpected)
		}
	}
}

// ────────────────────── fillItemURLs ──────────────────────

func TestFillItemURLs(t *testing.T) {
	svc := newMediaService(t)
	item := &model.MediaFileItem{
		MediaFile: model.MediaFile{
			Thumbnail: "abc123.jpg",
			Preview:   "abc123.preview.webp",
		},
		FolderID: "folder-id",
	}
	svc.fillItemURLs(item)

	wantThumb := "/persist/media-folders/folder-id/thumbnails/abc123.jpg"
	if item.ThumbnailURL != wantThumb {
		t.Errorf("ThumbnailURL = %q, want %q", item.ThumbnailURL, wantThumb)
	}
	wantPreview := "/persist/media-folders/folder-id/thumbnails/abc123.preview.webp"
	if item.PreviewURL != wantPreview {
		t.Errorf("PreviewURL = %q, want %q", item.PreviewURL, wantPreview)
	}
}

func TestFillItemURLs_Empty(t *testing.T) {
	svc := newMediaService(t)
	item := &model.MediaFileItem{
		MediaFile: model.MediaFile{},
		FolderID:  "folder-id",
	}
	svc.fillItemURLs(item)

	if item.ThumbnailURL != "" {
		t.Errorf("ThumbnailURL = %q, want empty", item.ThumbnailURL)
	}
	if item.PreviewURL != "" {
		t.Errorf("PreviewURL = %q, want empty", item.PreviewURL)
	}
}

// ────────────────────── isSubPath ──────────────────────

func TestIsSubPath(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		child, parent string
		want          bool
	}{
		{"/a/b/c", "/a/b", true},
		{"/a/b", "/a/b/c", false},
		{"/a/b", "/a/b", false},
		{"/a/bc", "/a/b", false},
		{"/a/b" + sep + "c", "/a/b", true},
	}
	for _, tt := range tests {
		got := isSubPath(tt.child, tt.parent)
		if got != tt.want {
			t.Errorf("isSubPath(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
		}
	}
}
