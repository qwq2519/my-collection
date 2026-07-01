package service

import (
	"os"
	"path/filepath"
	"testing"

	"collections/internal/model"
)

func TestNoteService_CreateValidation(t *testing.T) {
	svc := &NoteService{Store: newTestStore(t)}

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
	svc := &NoteService{Store: newTestStore(t)}
	_, err := svc.GetNote("")
	if err == nil {
		t.Error("GetNote should reject empty ID")
	}
}

func TestNoteService_UpdateValidation(t *testing.T) {
	svc := &NoteService{Store: newTestStore(t)}

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
	svc := &NoteService{Store: newTestStore(t)}
	err := svc.DeleteNote("")
	if err == nil {
		t.Error("DeleteNote should reject empty ID")
	}
}

func TestNoteService_DeleteCleansImages(t *testing.T) {
	s := newTestStore(t)
	svc := &NoteService{Store: s}

	note, _ := svc.CreateNote(model.CreateNoteReq{Title: "Test", Body: "body"})

	imgDir := filepath.Join(s.PersistDir(), "note-images", note.ID)
	os.MkdirAll(imgDir, 0755)
	os.WriteFile(filepath.Join(imgDir, "img.png"), []byte("fake"), 0644)

	svc.DeleteNote(note.ID)

	if _, err := os.Stat(imgDir); !os.IsNotExist(err) {
		t.Error("note images dir should be removed after delete")
	}
}

func TestNoteService_DetectOrphanImages(t *testing.T) {
	s := newTestStore(t)
	svc := &NoteService{Store: s}

	noteID := "test-note-id"
	imgDir := filepath.Join(s.PersistDir(), "note-images", noteID)
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
	svc := &NoteService{Store: newTestStore(t)}
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
	s := newTestStore(t)
	svc := &NoteService{Store: s}

	noteID := "test-note-id"
	imgDir := filepath.Join(s.PersistDir(), "note-images", noteID)
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
	s := newTestStore(t)
	svc := &NoteService{Store: s}

	noteID := "test-note-id"
	imgDir := filepath.Join(s.PersistDir(), "note-images", noteID)
	os.MkdirAll(imgDir, 0755)

	err := svc.DeleteOrphanImages(model.DeleteOrphanImagesReq{
		NoteID: noteID,
		Files:  []string{"../../main.db"},
	})
	if err != nil {
		t.Fatalf("DeleteOrphanImages should not error (skips unsafe): %v", err)
	}
}
