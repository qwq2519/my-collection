package service

import (
	"os"
	"path/filepath"
	"testing"

	"collections/internal/model"
)

func TestNoteService_CreateValidation(t *testing.T) {
	svc := newNoteService(t)

	_, err := svc.CreateNote(model.CreateNoteReq{Title: "", Body: "body"})
	if err == nil {
		t.Error("CreateNote should reject empty title")
	}

	_, err = svc.CreateNote(model.CreateNoteReq{Title: "   ", Body: "body"})
	if err == nil {
		t.Error("CreateNote should reject whitespace title")
	}
}

func TestNoteService_GetValidation(t *testing.T) {
	svc := newNoteService(t)
	_, err := svc.GetNote("")
	if err == nil {
		t.Error("GetNote should reject empty ID")
	}
}

func TestNoteService_UpdateValidation(t *testing.T) {
	svc := newNoteService(t)

	_, err := svc.UpdateNote(model.UpdateNoteReq{ID: ""})
	if err == nil {
		t.Error("UpdateNote should reject empty ID")
	}

	emptyTitle := "   "
	_, err = svc.UpdateNote(model.UpdateNoteReq{ID: "x", Title: &emptyTitle})
	if err == nil {
		t.Error("UpdateNote should reject whitespace title")
	}
}

func TestNoteService_DeleteValidation(t *testing.T) {
	svc := newNoteService(t)
	err := svc.DeleteNote("")
	if err == nil {
		t.Error("DeleteNote should reject empty ID")
	}
}

func TestNoteService_DeleteCleansImages(t *testing.T) {
	svc := newNoteService(t)

	note, err := svc.CreateNote(model.CreateNoteReq{Title: "Test", Body: "body"})
	if err != nil {
		t.Fatalf("setup CreateNote: %v", err)
	}

	imgDir := filepath.Join(svc.Store.PersistDir(), "note-images", note.ID)
	os.MkdirAll(imgDir, 0755)
	os.WriteFile(filepath.Join(imgDir, "img.png"), []byte("fake"), 0644)

	if err := svc.DeleteNote(note.ID); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	if _, err := os.Stat(imgDir); !os.IsNotExist(err) {
		t.Error("note images dir should be removed after delete")
	}
}

func TestNoteService_DetectOrphanImages(t *testing.T) {
	svc := newNoteService(t)

	noteID := "test-note-id"
	imgDir := filepath.Join(svc.Store.PersistDir(), "note-images", noteID)
	os.MkdirAll(imgDir, 0755)
	os.WriteFile(filepath.Join(imgDir, "used.png"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(imgDir, "orphan.png"), []byte("x"), 0644)

	body := "some text ![img](note-images/test-note-id/used.png) more text"

	result, err := svc.DetectOrphanImages(model.DetectOrphanImagesReq{
		NoteID: noteID,
		Body:   body,
	})
	if err != nil {
		t.Fatalf("DetectOrphanImages: %v", err)
	}
	if len(result.OrphanFiles) != 1 {
		t.Fatalf("orphans len = %d, want 1", len(result.OrphanFiles))
	}
	if result.OrphanFiles[0] != "orphan.png" {
		t.Errorf("orphan = %q, want %q", result.OrphanFiles[0], "orphan.png")
	}
}

func TestNoteService_DetectOrphanImagesNoDir(t *testing.T) {
	svc := newNoteService(t)
	result, err := svc.DetectOrphanImages(model.DetectOrphanImagesReq{
		NoteID: "nonexistent",
		Body:   "some text",
	})
	if err != nil {
		t.Fatalf("DetectOrphanImages: %v", err)
	}
	if len(result.OrphanFiles) != 0 {
		t.Errorf("orphans = %v, want empty", result.OrphanFiles)
	}
}

func TestNoteService_DeleteOrphanImages(t *testing.T) {
	svc := newNoteService(t)

	noteID := "test-note-id"
	imgDir := filepath.Join(svc.Store.PersistDir(), "note-images", noteID)
	os.MkdirAll(imgDir, 0755)
	os.WriteFile(filepath.Join(imgDir, "orphan.png"), []byte("x"), 0644)

	err := svc.DeleteOrphanImages(model.DeleteOrphanImagesReq{
		NoteID: noteID,
		Files:  []string{"orphan.png"},
	})
	if err != nil {
		t.Fatalf("DeleteOrphanImages: %v", err)
	}

	if _, err := os.Stat(filepath.Join(imgDir, "orphan.png")); !os.IsNotExist(err) {
		t.Error("orphan.png should be deleted")
	}
}

func TestNoteService_DeleteOrphanImagesTraversalRejected(t *testing.T) {
	svc := newNoteService(t)

	noteID := "test-note-id"
	imgDir := filepath.Join(svc.Store.PersistDir(), "note-images", noteID)
	os.MkdirAll(imgDir, 0755)

	err := svc.DeleteOrphanImages(model.DeleteOrphanImagesReq{
		NoteID: noteID,
		Files:  []string{"../../main.db"},
	})
	if err != nil {
		t.Fatalf("DeleteOrphanImages should not error (skips unsafe): %v", err)
	}
}

// ────────────────────── ListNotes ──────────────────────

func TestNoteService_ListNotes(t *testing.T) {
	svc := newNoteService(t)
	for i := 0; i < 5; i++ {
		if _, err := svc.CreateNote(model.CreateNoteReq{
			Title: "Note", Body: "body",
		}); err != nil {
			t.Fatalf("setup CreateNote: %v", err)
		}
	}

	result, err := svc.ListNotes(model.NoteListReq{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("Total = %d, want 5", result.Total)
	}
	if len(result.Items) != 3 {
		t.Errorf("Items len = %d, want 3", len(result.Items))
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}

	page2, err := svc.ListNotes(model.NoteListReq{Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("ListNotes page 2: %v", err)
	}
	if len(page2.Items) != 2 {
		t.Errorf("page 2 Items len = %d, want 2", len(page2.Items))
	}
	if page2.HasMore {
		t.Error("page 2 HasMore should be false")
	}
}

func TestNoteService_ListNotesSearch(t *testing.T) {
	svc := newNoteService(t)
	if _, err := svc.CreateNote(model.CreateNoteReq{
		Title: "React Hooks Guide", Body: "useState useEffect",
	}); err != nil {
		t.Fatalf("setup CreateNote: %v", err)
	}
	if _, err := svc.CreateNote(model.CreateNoteReq{
		Title: "Go Concurrency", Body: "goroutines channels",
	}); err != nil {
		t.Fatalf("setup CreateNote: %v", err)
	}

	result, err := svc.ListNotes(model.NoteListReq{
		Search: "react", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListNotes search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("search Total = %d, want 1", result.Total)
	}
	if len(result.Items) == 1 && result.Items[0].Title != "React Hooks Guide" {
		t.Errorf("result Title = %q, want %q", result.Items[0].Title, "React Hooks Guide")
	}
}

func TestNoteService_ListNotesSearchChinese(t *testing.T) {
	svc := newNoteService(t)
	if _, err := svc.CreateNote(model.CreateNoteReq{
		Title: "K8s 笔记", Body: "Kubernetes 容器编排",
	}); err != nil {
		t.Fatalf("setup CreateNote: %v", err)
	}
	if _, err := svc.CreateNote(model.CreateNoteReq{
		Title: "Docker 笔记", Body: "Docker 镜像构建",
	}); err != nil {
		t.Fatalf("setup CreateNote: %v", err)
	}

	result, err := svc.ListNotes(model.NoteListReq{
		Search: "容器", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListNotes body search: %v", err)
	}
	if result.Total < 1 {
		t.Errorf("body search Total = %d, want >= 1", result.Total)
	}
}
