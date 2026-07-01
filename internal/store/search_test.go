package store

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
)

func TestIndexDocAndSearch(t *testing.T) {
	s := newTestStore(t)

	s.IndexDoc("site:abc", map[string]interface{}{
		"_type":      "site",
		"title":      "Go Official",
		"domain":     "golang.org",
		"updated_at": time.Now(),
	})

	q := bleve.NewTermQuery("site")
	q.SetField("_type")
	req := bleve.NewSearchRequest(q)
	req.Fields = []string{"title"}

	result, err := s.Search(req)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

func TestDeleteDocFromIndex(t *testing.T) {
	s := newTestStore(t)

	s.IndexDoc("note:xyz", map[string]interface{}{
		"_type":      "note",
		"title":      "Test Note",
		"updated_at": time.Now(),
	})

	s.DeleteDoc("note:xyz", "note")

	q := bleve.NewTermQuery("note")
	q.SetField("_type")
	req := bleve.NewSearchRequest(q)
	result, err := s.Search(req)
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total after delete = %d, want 0", result.Total)
	}
}

func TestSearchByText(t *testing.T) {
	s := newTestStore(t)

	s.IndexDoc("site:1", map[string]interface{}{
		"_type":      "site",
		"title":      "React Documentation",
		"domain":     "react.dev",
		"updated_at": time.Now(),
	})
	s.IndexDoc("site:2", map[string]interface{}{
		"_type":      "site",
		"title":      "Vue.js Guide",
		"domain":     "vuejs.org",
		"updated_at": time.Now(),
	})

	titleQ := bleve.NewMatchQuery("react")
	titleQ.SetField("title")
	req := bleve.NewSearchRequest(titleQ)

	result, err := s.Search(req)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
	if result.Total > 0 && result.Hits[0].ID != "site:1" {
		t.Errorf("hit ID = %q, want %q", result.Hits[0].ID, "site:1")
	}
}

func TestSearchByTag(t *testing.T) {
	s := newTestStore(t)

	s.IndexDoc("bm:1", map[string]interface{}{
		"_type":      "bookmark",
		"title":      "Article A",
		"tags":       []string{"frontend", "react"},
		"updated_at": time.Now(),
	})
	s.IndexDoc("bm:2", map[string]interface{}{
		"_type":      "bookmark",
		"title":      "Article B",
		"tags":       []string{"backend", "go"},
		"updated_at": time.Now(),
	})

	tagQ := bleve.NewTermQuery("react")
	tagQ.SetField("tags")
	req := bleve.NewSearchRequest(tagQ)

	result, err := s.Search(req)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

func TestDirtyItemsLifecycle(t *testing.T) {
	s := newTestStore(t)

	if s.HasDirtyItems() {
		t.Error("fresh store should have no dirty items")
	}

	s.addDirtyItem("site:broken", "site")
	s.addDirtyItem("note:bad", "note")

	if !s.HasDirtyItems() {
		t.Error("should have dirty items after adding")
	}

	items := s.GetDirtyItems()
	if len(items) != 2 {
		t.Errorf("dirty items len = %d, want 2", len(items))
	}

	status := s.GetDirtyIndexStatus()
	if !status.HasDirty {
		t.Error("HasDirty should be true")
	}
	if status.URLCount != 1 {
		t.Errorf("URLCount = %d, want 1", status.URLCount)
	}
	if status.NoteCount != 1 {
		t.Errorf("NoteCount = %d, want 1", status.NoteCount)
	}

	s.removeDirtyItem("site:broken")
	items = s.GetDirtyItems()
	if len(items) != 1 {
		t.Errorf("after remove, dirty items len = %d, want 1", len(items))
	}

	s.ClearAllDirtyItems()
	if s.HasDirtyItems() {
		t.Error("should have no dirty items after clear")
	}
}

func TestClearDirtyByType(t *testing.T) {
	s := newTestStore(t)

	s.addDirtyItem("site:a", "site")
	s.addDirtyItem("bm:b", "bookmark")
	s.addDirtyItem("note:c", "note")

	s.ClearDirtyByType("site")

	status := s.GetDirtyIndexStatus()
	if status.URLCount != 1 {
		t.Errorf("URLCount after clear site = %d, want 1 (bookmark remains)", status.URLCount)
	}
	if status.NoteCount != 1 {
		t.Errorf("NoteCount = %d, want 1", status.NoteCount)
	}
}

func TestRebuildIndexByType(t *testing.T) {
	s := newTestStore(t)

	s.IndexDoc("site:old1", map[string]interface{}{
		"_type": "site", "title": "Old Site",
		"updated_at": time.Now(),
	})
	s.IndexDoc("note:keep", map[string]interface{}{
		"_type": "note", "title": "Keep Me",
		"updated_at": time.Now(),
	})

	newDocs := []BleveDoc{
		{ID: "site:new1", Fields: map[string]interface{}{
			"_type": "site", "title": "New Site A",
			"updated_at": time.Now(),
		}},
		{ID: "site:new2", Fields: map[string]interface{}{
			"_type": "site", "title": "New Site B",
			"updated_at": time.Now(),
		}},
	}

	if err := s.RebuildIndexByType("site", newDocs); err != nil {
		t.Fatalf("RebuildIndexByType: %v", err)
	}

	siteQ := bleve.NewTermQuery("site")
	siteQ.SetField("_type")
	req := bleve.NewSearchRequest(siteQ)
	result, _ := s.Search(req)
	if result.Total != 2 {
		t.Errorf("site docs after rebuild = %d, want 2", result.Total)
	}

	noteQ := bleve.NewTermQuery("note")
	noteQ.SetField("_type")
	req2 := bleve.NewSearchRequest(noteQ)
	noteResult, _ := s.Search(req2)
	if noteResult.Total != 1 {
		t.Errorf("note docs should be untouched = %d, want 1", noteResult.Total)
	}
}
