package service

import (
	"os"
	"path/filepath"
	"testing"

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
