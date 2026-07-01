package store

import (
	"testing"
	"time"

	"collections/internal/model"
)

func TestTagSetAndGet(t *testing.T) {
	s := newTestStore(t)
	tag := &model.Tag{Name: "go", Count: 5, CreatedAt: time.Now()}

	if err := s.SetTag("url_tag", tag); err != nil {
		t.Fatalf("SetTag: %v", err)
	}

	got, err := s.GetTag("url_tag", "go")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if got == nil {
		t.Fatal("tag should not be nil")
	}
	if got.Name != "go" {
		t.Errorf("Name = %q, want %q", got.Name, "go")
	}
	if got.Count != 5 {
		t.Errorf("Count = %d, want 5", got.Count)
	}
}

func TestTagGetNotFound(t *testing.T) {
	s := newTestStore(t)
	got, err := s.GetTag("url_tag", "nonexistent")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if got != nil {
		t.Error("should return nil for nonexistent tag")
	}
}

func TestTagList(t *testing.T) {
	s := newTestStore(t)
	s.SetTag("url_tag", &model.Tag{Name: "go", Count: 3, CreatedAt: time.Now()})
	s.SetTag("url_tag", &model.Tag{Name: "rust", Count: 1, CreatedAt: time.Now()})
	s.SetTag("media_tag", &model.Tag{Name: "photo", Count: 10, CreatedAt: time.Now()})

	result, err := s.ListTags("url_tag")
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("url_tag Total = %d, want 2", result.Total)
	}

	mediaResult, _ := s.ListTags("media_tag")
	if mediaResult.Total != 1 {
		t.Errorf("media_tag Total = %d, want 1", mediaResult.Total)
	}
}

func TestTagDeleteEntry(t *testing.T) {
	s := newTestStore(t)
	s.SetTag("url_tag", &model.Tag{Name: "old", Count: 1, CreatedAt: time.Now()})

	if err := s.DeleteTagEntry("url_tag", "old"); err != nil {
		t.Fatalf("DeleteTagEntry: %v", err)
	}

	got, _ := s.GetTag("url_tag", "old")
	if got != nil {
		t.Error("tag should be nil after delete")
	}
}

func TestTagAdjustCount(t *testing.T) {
	s := newTestStore(t)

	// auto-create on positive delta
	s.AdjustTagCount("url_tag", "new_tag", 3)
	got, _ := s.GetTag("url_tag", "new_tag")
	if got == nil || got.Count != 3 {
		t.Fatalf("auto-created tag Count = %v, want 3", got)
	}

	// increment
	s.AdjustTagCount("url_tag", "new_tag", 2)
	got, _ = s.GetTag("url_tag", "new_tag")
	if got.Count != 5 {
		t.Errorf("Count after +2 = %d, want 5", got.Count)
	}

	// decrement
	s.AdjustTagCount("url_tag", "new_tag", -3)
	got, _ = s.GetTag("url_tag", "new_tag")
	if got.Count != 2 {
		t.Errorf("Count after -3 = %d, want 2", got.Count)
	}

	// clamp to 0
	s.AdjustTagCount("url_tag", "new_tag", -10)
	got, _ = s.GetTag("url_tag", "new_tag")
	if got.Count != 0 {
		t.Errorf("Count after -10 = %d, want 0 (clamped)", got.Count)
	}

	// negative delta on nonexistent: no-op
	s.AdjustTagCount("url_tag", "ghost", -1)
	ghost, _ := s.GetTag("url_tag", "ghost")
	if ghost != nil {
		t.Error("negative delta on nonexistent should not create tag")
	}
}

func TestTagBatchAdjust(t *testing.T) {
	s := newTestStore(t)
	s.SetTag("url_tag", &model.Tag{Name: "a", Count: 10, CreatedAt: time.Now()})

	err := s.BatchAdjustTagCounts("url_tag", map[string]int{
		"a": -2,
		"b": 5,
	})
	if err != nil {
		t.Fatalf("BatchAdjustTagCounts: %v", err)
	}

	a, _ := s.GetTag("url_tag", "a")
	if a.Count != 8 {
		t.Errorf("a Count = %d, want 8", a.Count)
	}
	b, _ := s.GetTag("url_tag", "b")
	if b == nil || b.Count != 5 {
		t.Errorf("b Count = %v, want 5", b)
	}
}

func TestRenameURLTag(t *testing.T) {
	s := newTestStore(t)
	site, _ := s.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com", Domain: "test.com",
		Tags: []string{"old_name"},
	})
	s.SetTag("url_tag", &model.Tag{Name: "old_name", Count: 1, CreatedAt: time.Now()})

	affected, err := s.RenameURLTag(model.RenameTagReq{OldName: "old_name", NewName: "new_name"})
	if err != nil {
		t.Fatalf("RenameURLTag: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	// tag registry updated
	old, _ := s.GetTag("url_tag", "old_name")
	if old != nil {
		t.Error("old tag should be deleted")
	}
	newTag, _ := s.GetTag("url_tag", "new_name")
	if newTag == nil {
		t.Fatal("new tag should exist")
	}

	// entity tags updated
	updatedSite, _ := s.GetSite(site.ID)
	if len(updatedSite.Tags) != 1 || updatedSite.Tags[0] != "new_name" {
		t.Errorf("site Tags = %v, want [new_name]", updatedSite.Tags)
	}
}

func TestRenameURLTagConflict(t *testing.T) {
	s := newTestStore(t)
	s.SetTag("url_tag", &model.Tag{Name: "a", Count: 1, CreatedAt: time.Now()})
	s.SetTag("url_tag", &model.Tag{Name: "b", Count: 1, CreatedAt: time.Now()})

	_, err := s.RenameURLTag(model.RenameTagReq{OldName: "a", NewName: "b"})
	if err == nil {
		t.Error("rename to existing tag should fail (use merge instead)")
	}
}

func TestMergeURLTag(t *testing.T) {
	s := newTestStore(t)
	s.CreateSite(model.CreateSiteReq{
		Title: "S1", URL: "https://s1.com", Domain: "s1.com",
		Tags: []string{"source", "target"},
	})
	s.CreateSite(model.CreateSiteReq{
		Title: "S2", URL: "https://s2.com", Domain: "s2.com",
		Tags: []string{"source"},
	})
	s.SetTag("url_tag", &model.Tag{Name: "source", Count: 2, CreatedAt: time.Now()})
	s.SetTag("url_tag", &model.Tag{Name: "target", Count: 1, CreatedAt: time.Now()})

	affected, err := s.MergeURLTag(model.MergeTagReq{Source: "source", Target: "target"})
	if err != nil {
		t.Fatalf("MergeURLTag: %v", err)
	}
	if affected != 2 {
		t.Errorf("affected = %d, want 2", affected)
	}

	src, _ := s.GetTag("url_tag", "source")
	if src != nil {
		t.Error("source tag should be deleted after merge")
	}
	tgt, _ := s.GetTag("url_tag", "target")
	if tgt == nil {
		t.Fatal("target tag should exist")
	}
	if tgt.Count != 2 {
		t.Errorf("target Count = %d, want 2", tgt.Count)
	}
}

func TestDeleteURLTagFromEntities(t *testing.T) {
	s := newTestStore(t)
	site, _ := s.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com", Domain: "test.com",
		Tags: []string{"keep", "remove"},
	})
	s.SetTag("url_tag", &model.Tag{Name: "remove", Count: 1, CreatedAt: time.Now()})

	affected, err := s.DeleteURLTagFromEntities("remove")
	if err != nil {
		t.Fatalf("DeleteURLTagFromEntities: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	updatedSite, _ := s.GetSite(site.ID)
	if len(updatedSite.Tags) != 1 || updatedSite.Tags[0] != "keep" {
		t.Errorf("site Tags = %v, want [keep]", updatedSite.Tags)
	}

	tag, _ := s.GetTag("url_tag", "remove")
	if tag != nil {
		t.Error("tag registry entry should be deleted")
	}
}

func TestRecountURLTags(t *testing.T) {
	s := newTestStore(t)
	s.CreateSite(model.CreateSiteReq{
		Title: "S1", URL: "https://s1.com", Domain: "s1.com",
		Tags: []string{"go", "web"},
	})
	site2, _ := s.CreateSite(model.CreateSiteReq{
		Title: "S2", URL: "https://s2.com", Domain: "s2.com",
		Tags: []string{"go"},
	})
	s.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://s2.com/page", Domain: "s2.com", SiteID: site2.ID,
		Title: "Page", Tags: []string{"go", "tutorial"},
	})

	if err := s.RecountURLTags(); err != nil {
		t.Fatalf("RecountURLTags: %v", err)
	}

	goTag, _ := s.GetTag("url_tag", "go")
	if goTag == nil || goTag.Count != 3 {
		t.Errorf("go Count = %v, want 3", goTag)
	}
	webTag, _ := s.GetTag("url_tag", "web")
	if webTag == nil || webTag.Count != 1 {
		t.Errorf("web Count = %v, want 1", webTag)
	}
	tutTag, _ := s.GetTag("url_tag", "tutorial")
	if tutTag == nil || tutTag.Count != 1 {
		t.Errorf("tutorial Count = %v, want 1 (auto-created by recount)", tutTag)
	}
}
