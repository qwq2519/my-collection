package store

import (
	"testing"

	"collections/internal/model"
)

func TestNoteCreate(t *testing.T) {
	s := newTestStore(t)
	note, err := s.CreateNote(model.CreateNoteReq{
		Title: "First Note",
		Body:  "Hello **world**",
	})
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if note.ID == "" {
		t.Error("note ID should not be empty")
	}
	if note.Title != "First Note" {
		t.Errorf("Title = %q, want %q", note.Title, "First Note")
	}
	if note.Body != "Hello **world**" {
		t.Errorf("Body = %q, want %q", note.Body, "Hello **world**")
	}
}

func TestNoteGetAndUpdate(t *testing.T) {
	s := newTestStore(t)
	note, _ := s.CreateNote(model.CreateNoteReq{Title: "Original", Body: "body"})

	got, err := s.GetNote(note.ID)
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if got.Title != "Original" {
		t.Errorf("Title = %q, want %q", got.Title, "Original")
	}

	newTitle := "Updated"
	newBody := "new body"
	updated, err := s.UpdateNote(model.UpdateNoteReq{
		ID:    note.ID,
		Title: &newTitle,
		Body:  &newBody,
	})
	if err != nil {
		t.Fatalf("UpdateNote: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("updated Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Body != newBody {
		t.Errorf("updated Body = %q, want %q", updated.Body, newBody)
	}
	if !updated.UpdatedAt.After(note.UpdatedAt) {
		t.Error("UpdatedAt should advance")
	}
}

func TestNoteDelete(t *testing.T) {
	s := newTestStore(t)
	note, _ := s.CreateNote(model.CreateNoteReq{Title: "ToDelete", Body: "x"})

	if err := s.DeleteNote(note.ID); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	_, err := s.GetNote(note.ID)
	if err == nil {
		t.Error("GetNote should fail after delete")
	}
}

func TestNoteDeleteNotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.DeleteNote("nonexistent")
	if err == nil {
		t.Error("DeleteNote should fail for nonexistent ID")
	}
}

func TestNoteListPagination(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		s.CreateNote(model.CreateNoteReq{
			Title: "Note",
			Body:  "body",
		})
	}

	result, err := s.ListNotes(model.NoteListReq{Page: 1, PageSize: 3})
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

	page2, _ := s.ListNotes(model.NoteListReq{Page: 2, PageSize: 3})
	if len(page2.Items) != 2 {
		t.Errorf("page 2 Items len = %d, want 2", len(page2.Items))
	}
}

func TestNoteSearchBleve(t *testing.T) {
	s := newTestStore(t)
	s.CreateNote(model.CreateNoteReq{Title: "React Hooks Guide", Body: "useState useEffect"})
	s.CreateNote(model.CreateNoteReq{Title: "Go Concurrency", Body: "goroutines channels"})
	s.CreateNote(model.CreateNoteReq{Title: "CSS Grid Layout", Body: "grid template areas"})

	result, err := s.ListNotes(model.NoteListReq{
		Search:   "react",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListNotes search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("search Total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("search Items len = %d, want 1", len(result.Items))
	}
	if result.Items[0].Title != "React Hooks Guide" {
		t.Errorf("search result Title = %q, want %q", result.Items[0].Title, "React Hooks Guide")
	}
}

func TestNoteSearchByBody(t *testing.T) {
	s := newTestStore(t)
	s.CreateNote(model.CreateNoteReq{Title: "Note A", Body: "Kubernetes 容器编排"})
	s.CreateNote(model.CreateNoteReq{Title: "Note B", Body: "Docker 镜像构建"})

	result, err := s.ListNotes(model.NoteListReq{
		Search:   "容器",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListNotes body search: %v", err)
	}
	if result.Total < 1 {
		t.Errorf("body search Total = %d, want >= 1", result.Total)
	}
}
