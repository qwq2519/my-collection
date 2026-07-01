package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"collections/internal/model"
)

// ────────────────────── buildTree ──────────────────────

func TestBuildTree_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	tree, err := buildTree(dir, nil)
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	if tree.Type != model.TreeNodeDir {
		t.Errorf("root type = %q, want %q", tree.Type, model.TreeNodeDir)
	}
	if len(tree.Children) != 0 {
		t.Errorf("children len = %d, want 0", len(tree.Children))
	}
	if tree.Hash != "0" {
		t.Errorf("empty dir hash = %q, want %q", tree.Hash, "0")
	}
}

func TestBuildTree_SkipsNonMedia(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b"), 0644)
	os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("jpeg"), 0644)

	tree, err := buildTree(dir, nil)
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("children len = %d, want 1 (only photo.jpg)", len(tree.Children))
	}
	if _, ok := tree.Children["photo.jpg"]; !ok {
		t.Error("should include photo.jpg")
	}
}

func TestBuildTree_FileNode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "img.png"), []byte("png data"), 0644)

	tree, err := buildTree(dir, nil)
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}

	child := tree.Children["img.png"]
	if child == nil {
		t.Fatal("img.png not found in tree")
	}
	if child.Type != model.TreeNodeFile {
		t.Errorf("type = %q, want %q", child.Type, model.TreeNodeFile)
	}
	if child.Hash == "" {
		t.Error("file hash should not be empty")
	}
	if child.Mtime == nil {
		t.Error("file mtime should not be nil")
	}
	if child.Size == nil || *child.Size != 8 {
		t.Errorf("file size = %v, want 8", child.Size)
	}
}

func TestBuildTree_SubDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "vacation")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "beach.jpg"), []byte("jpg"), 0644)

	tree, err := buildTree(dir, nil)
	if err != nil {
		t.Fatalf("buildTree: %v", err)
	}

	subNode := tree.Children["vacation"]
	if subNode == nil {
		t.Fatal("vacation dir not in tree")
	}
	if subNode.Type != model.TreeNodeDir {
		t.Errorf("vacation type = %q, want dir", subNode.Type)
	}
	if subNode.Children["beach.jpg"] == nil {
		t.Error("beach.jpg not found in vacation")
	}
}

func TestBuildTree_DeterministicHash(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "b.png"), []byte("bbb"), 0644)

	tree1, _ := buildTree(dir, nil)
	tree2, _ := buildTree(dir, nil)

	if tree1.Hash != tree2.Hash {
		t.Errorf("hashes differ: %q vs %q", tree1.Hash, tree2.Hash)
	}
}

func TestBuildTree_HashChangesOnModify(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "photo.jpg")
	os.WriteFile(file, []byte("v1"), 0644)

	tree1, _ := buildTree(dir, nil)

	time.Sleep(10 * time.Millisecond)
	os.WriteFile(file, []byte("v2 longer"), 0644)

	tree2, _ := buildTree(dir, nil)

	if tree1.Hash == tree2.Hash {
		t.Error("root hash should change when file is modified")
	}
	if tree1.Children["photo.jpg"].Hash == tree2.Children["photo.jpg"].Hash {
		t.Error("file hash should change when modified")
	}
}

func TestBuildTree_CachePruning(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "stable.jpg"), []byte("unchanged"), 0644)

	tree1, _ := buildTree(dir, nil)
	originalHash := tree1.Children["stable.jpg"].Hash

	tree2, _ := buildTree(dir, tree1)

	if tree2.Children["stable.jpg"].Hash != originalHash {
		t.Error("cached file hash should be reused when mtime+size unchanged")
	}
}

func TestBuildTree_SupportedFormats(t *testing.T) {
	dir := t.TempDir()
	exts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".avif", ".svg",
		".mp4", ".mkv", ".avi", ".mov", ".webm", ".wmv", ".flv",
		".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a"}

	for _, ext := range exts {
		os.WriteFile(filepath.Join(dir, "file"+ext), []byte("data"), 0644)
	}

	tree, _ := buildTree(dir, nil)
	if len(tree.Children) != len(exts) {
		t.Errorf("children = %d, want %d (all supported formats)", len(tree.Children), len(exts))
	}
}

func TestBuildTree_NestedSubDirs(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "deep.jpg"), []byte("deep"), 0644)

	tree, _ := buildTree(dir, nil)

	node := tree.Children["a"]
	if node == nil {
		t.Fatal("missing dir a")
	}
	node = node.Children["b"]
	if node == nil {
		t.Fatal("missing dir b")
	}
	node = node.Children["c"]
	if node == nil {
		t.Fatal("missing dir c")
	}
	if node.Children["deep.jpg"] == nil {
		t.Error("missing deep.jpg in a/b/c")
	}
}

// ────────────────────── diffTrees ──────────────────────

func TestDiffTrees_BothNil(t *testing.T) {
	diff := diffTrees(nil, nil)
	if len(diff.Added)+len(diff.Removed)+len(diff.Modified) != 0 {
		t.Error("diff of two nil trees should be empty")
	}
}

func TestDiffTrees_FirstScan(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.png"), []byte("b"), 0644)

	tree, _ := buildTree(dir, nil)
	diff := diffTrees(nil, tree)

	if len(diff.Added) != 2 {
		t.Errorf("Added = %d, want 2", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("Removed = %d, want 0", len(diff.Removed))
	}
}

func TestDiffTrees_NoChange(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "img.jpg"), []byte("data"), 0644)

	tree, _ := buildTree(dir, nil)
	diff := diffTrees(tree, tree)

	if len(diff.Added)+len(diff.Removed)+len(diff.Modified) != 0 {
		t.Error("diff of identical trees should be empty")
	}
}

func TestDiffTrees_FileAdded(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "old.jpg"), []byte("old"), 0644)
	tree1, _ := buildTree(dir, nil)

	os.WriteFile(filepath.Join(dir, "new.png"), []byte("new"), 0644)
	tree2, _ := buildTree(dir, nil)

	diff := diffTrees(tree1, tree2)
	if len(diff.Added) != 1 || diff.Added[0] != "new.png" {
		t.Errorf("Added = %v, want [new.png]", diff.Added)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("Removed = %v, want []", diff.Removed)
	}
}

func TestDiffTrees_FileRemoved(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "gone.jpg")
	os.WriteFile(f, []byte("bye"), 0644)
	tree1, _ := buildTree(dir, nil)

	os.Remove(f)
	tree2, _ := buildTree(dir, nil)

	diff := diffTrees(tree1, tree2)
	if len(diff.Removed) != 1 || diff.Removed[0] != "gone.jpg" {
		t.Errorf("Removed = %v, want [gone.jpg]", diff.Removed)
	}
	if len(diff.Added) != 0 {
		t.Errorf("Added = %v, want []", diff.Added)
	}
}

func TestDiffTrees_FileModified(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "photo.jpg")
	os.WriteFile(f, []byte("v1"), 0644)
	tree1, _ := buildTree(dir, nil)

	time.Sleep(10 * time.Millisecond)
	os.WriteFile(f, []byte("v2 longer content"), 0644)
	tree2, _ := buildTree(dir, nil)

	diff := diffTrees(tree1, tree2)
	if len(diff.Modified) != 1 || diff.Modified[0] != "photo.jpg" {
		t.Errorf("Modified = %v, want [photo.jpg]", diff.Modified)
	}
}

func TestDiffTrees_SubDirChanges(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "a.jpg"), []byte("a"), 0644)

	tree1, _ := buildTree(dir, nil)

	os.WriteFile(filepath.Join(sub, "b.png"), []byte("b"), 0644)
	tree2, _ := buildTree(dir, nil)

	diff := diffTrees(tree1, tree2)
	if len(diff.Added) != 1 || diff.Added[0] != "sub/b.png" {
		t.Errorf("Added = %v, want [sub/b.png]", diff.Added)
	}
}

func TestDiffTrees_AllDeleted(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.png"), []byte("b"), 0644)

	tree, _ := buildTree(dir, nil)
	diff := diffTrees(tree, nil)

	if len(diff.Removed) != 2 {
		t.Errorf("Removed = %d, want 2", len(diff.Removed))
	}
	if len(diff.Added) != 0 {
		t.Errorf("Added = %d, want 0", len(diff.Added))
	}
}

func TestDiffTrees_MixedChanges(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "keep.jpg"), []byte("keep"), 0644)
	os.WriteFile(filepath.Join(dir, "modify.png"), []byte("v1"), 0644)
	os.WriteFile(filepath.Join(dir, "remove.gif"), []byte("bye"), 0644)

	tree1, _ := buildTree(dir, nil)

	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "modify.png"), []byte("v2 changed"), 0644)
	os.Remove(filepath.Join(dir, "remove.gif"))
	os.WriteFile(filepath.Join(dir, "add.mp4"), []byte("new"), 0644)

	tree2, _ := buildTree(dir, nil)
	diff := diffTrees(tree1, tree2)

	if len(diff.Added) != 1 || diff.Added[0] != "add.mp4" {
		t.Errorf("Added = %v, want [add.mp4]", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "remove.gif" {
		t.Errorf("Removed = %v, want [remove.gif]", diff.Removed)
	}
	if len(diff.Modified) != 1 || diff.Modified[0] != "modify.png" {
		t.Errorf("Modified = %v, want [modify.png]", diff.Modified)
	}
}

// ────────────────────── computeFileHash ──────────────────────

func TestComputeFileHash_Deterministic(t *testing.T) {
	mtime := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	h1 := computeFileHash("photo.jpg", mtime, 1024)
	h2 := computeFileHash("photo.jpg", mtime, 1024)
	if h1 != h2 {
		t.Errorf("same input produces different hashes: %q vs %q", h1, h2)
	}
}

func TestComputeFileHash_DiffersOnPath(t *testing.T) {
	mtime := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	h1 := computeFileHash("a.jpg", mtime, 100)
	h2 := computeFileHash("b.jpg", mtime, 100)
	if h1 == h2 {
		t.Error("different paths should produce different hashes")
	}
}

func TestComputeFileHash_DiffersOnMtime(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	h1 := computeFileHash("a.jpg", t1, 100)
	h2 := computeFileHash("a.jpg", t2, 100)
	if h1 == h2 {
		t.Error("different mtimes should produce different hashes")
	}
}

func TestComputeFileHash_DiffersOnSize(t *testing.T) {
	mtime := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	h1 := computeFileHash("a.jpg", mtime, 100)
	h2 := computeFileHash("a.jpg", mtime, 200)
	if h1 == h2 {
		t.Error("different sizes should produce different hashes")
	}
}

// ────────────────────── computeDirHash ──────────────────────

func TestComputeDirHash_Empty(t *testing.T) {
	h := computeDirHash(map[string]*model.TreeNode{})
	if h != "0" {
		t.Errorf("empty dir hash = %q, want %q", h, "0")
	}
}

func TestComputeDirHash_OrderIndependent(t *testing.T) {
	children := map[string]*model.TreeNode{
		"a": {Hash: "abc"},
		"b": {Hash: "def"},
	}
	h1 := computeDirHash(children)

	children2 := map[string]*model.TreeNode{
		"b": {Hash: "def"},
		"a": {Hash: "abc"},
	}
	h2 := computeDirHash(children2)

	if h1 != h2 {
		t.Error("dir hash should not depend on iteration order")
	}
}
