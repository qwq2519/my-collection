package service

import (
	"testing"

	"collections/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newURLService(t *testing.T) *URLService {
	t.Helper()
	return &URLService{Store: newTestStore(t)}
}

func newNoteService(t *testing.T) *NoteService {
	t.Helper()
	return &NoteService{Store: newTestStore(t)}
}

func newUploadService(t *testing.T) *UploadService {
	t.Helper()
	return &UploadService{Store: newTestStore(t)}
}
