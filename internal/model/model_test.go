package model

import "testing"

func TestBookmarkEnsureSlices(t *testing.T) {
	t.Run("nil slices become empty", func(t *testing.T) {
		bm := Bookmark{}
		bm.EnsureSlices()
		if bm.Tags == nil {
			t.Error("Tags should not be nil after EnsureSlices")
		}
		if len(bm.Tags) != 0 {
			t.Errorf("Tags should be empty, got %v", bm.Tags)
		}
		if bm.Attachments == nil {
			t.Error("Attachments should not be nil after EnsureSlices")
		}
		if len(bm.Attachments) != 0 {
			t.Errorf("Attachments should be empty, got %v", bm.Attachments)
		}
	})

	t.Run("existing values preserved", func(t *testing.T) {
		bm := Bookmark{
			Tags:        []string{"go", "rust"},
			Attachments: []Attachment{{Filename: "a.txt"}},
		}
		bm.EnsureSlices()
		if len(bm.Tags) != 2 {
			t.Errorf("Tags should have 2 items, got %d", len(bm.Tags))
		}
		if len(bm.Attachments) != 1 {
			t.Errorf("Attachments should have 1 item, got %d", len(bm.Attachments))
		}
	})
}

func TestSiteEnsureSlices(t *testing.T) {
	t.Run("nil slices become empty", func(t *testing.T) {
		s := Site{}
		s.EnsureSlices()
		if s.Tags == nil {
			t.Error("Tags should not be nil after EnsureSlices")
		}
		if len(s.Tags) != 0 {
			t.Errorf("Tags should be empty, got %v", s.Tags)
		}
		if s.Attachments == nil {
			t.Error("Attachments should not be nil after EnsureSlices")
		}
		if len(s.Attachments) != 0 {
			t.Errorf("Attachments should be empty, got %v", s.Attachments)
		}
	})

	t.Run("existing values preserved", func(t *testing.T) {
		s := Site{
			Tags:        []string{"前端", "react"},
			Attachments: []Attachment{{Filename: "b.pdf"}},
		}
		s.EnsureSlices()
		if len(s.Tags) != 2 {
			t.Errorf("Tags should have 2 items, got %d", len(s.Tags))
		}
		if len(s.Attachments) != 1 {
			t.Errorf("Attachments should have 1 item, got %d", len(s.Attachments))
		}
	})
}
