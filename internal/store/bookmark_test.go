package store

import (
	"testing"

	"collections/internal/model"
)

func createTestBookmark(t *testing.T, s *Store, siteID, url, title string) *model.Bookmark {
	t.Helper()
	bm, err := s.CreateBookmark(model.CreateBookmarkReq{
		URL:    url,
		Domain: "example.com",
		SiteID: siteID,
		Title:  title,
	})
	if err != nil {
		t.Fatalf("create bookmark %q: %v", url, err)
	}
	return bm
}

func TestBookmarkCreate(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "github.com")

	bm, err := s.CreateBookmark(model.CreateBookmarkReq{
		URL:         "https://github.com/golang/go",
		Domain:      "github.com",
		SiteID:      site.ID,
		Title:       "Go repo",
		Description: "The Go programming language",
		Tags:        []string{"go", "lang"},
	})
	if err != nil {
		t.Fatalf("CreateBookmark: %v", err)
	}

	if bm.ID == "" {
		t.Error("bookmark ID should not be empty")
	}
	if bm.SiteID != site.ID {
		t.Errorf("SiteID = %q, want %q", bm.SiteID, site.ID)
	}
	if bm.Status != model.BookmarkStatusAlive {
		t.Errorf("Status = %q, want %q", bm.Status, model.BookmarkStatusAlive)
	}
	if len(bm.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(bm.Tags))
	}

	updated, _ := s.GetSite(site.ID)
	if updated.BookmarkCount != 1 {
		t.Errorf("site BookmarkCount = %d, want 1", updated.BookmarkCount)
	}
}

func TestBookmarkDuplicateURL(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "github.com")
	createTestBookmark(t, s, site.ID, "https://github.com/golang/go", "Go")

	_, err := s.CreateBookmark(model.CreateBookmarkReq{
		URL:    "https://github.com/golang/go",
		Domain: "github.com",
		SiteID: site.ID,
		Title:  "Go duplicate",
	})
	if err == nil {
		t.Error("expected error for duplicate URL, got nil")
	}
}

func TestBookmarkGetAndUpdate(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")
	bm := createTestBookmark(t, s, site.ID, "https://example.com/page", "Page")

	got, err := s.GetBookmark(bm.ID)
	if err != nil {
		t.Fatalf("GetBookmark: %v", err)
	}
	if got.Title != "Page" {
		t.Errorf("Title = %q, want %q", got.Title, "Page")
	}

	newTitle := "Updated Page"
	newTags := []string{"web"}
	updated, _, err := s.UpdateBookmark(model.UpdateBookmarkReq{
		ID:    bm.ID,
		Title: &newTitle,
		Tags:  &newTags,
	})
	if err != nil {
		t.Fatalf("UpdateBookmark: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("updated Title = %q, want %q", updated.Title, newTitle)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "web" {
		t.Errorf("updated Tags = %v, want [web]", updated.Tags)
	}
}

func TestBookmarkDelete(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")
	bm := createTestBookmark(t, s, site.ID, "https://example.com/a", "A")

	siteAfterCreate, _ := s.GetSite(site.ID)
	if siteAfterCreate.BookmarkCount != 1 {
		t.Fatalf("BookmarkCount after create = %d, want 1", siteAfterCreate.BookmarkCount)
	}

	if _, err := s.DeleteBookmark(bm.ID); err != nil {
		t.Fatalf("DeleteBookmark: %v", err)
	}

	_, err := s.GetBookmark(bm.ID)
	if err == nil {
		t.Error("GetBookmark should fail after delete")
	}

	siteAfterDelete, _ := s.GetSite(site.ID)
	if siteAfterDelete.BookmarkCount != 0 {
		t.Errorf("BookmarkCount after delete = %d, want 0", siteAfterDelete.BookmarkCount)
	}
}

func TestBookmarkBatchDelete(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")
	bm1 := createTestBookmark(t, s, site.ID, "https://example.com/1", "One")
	bm2 := createTestBookmark(t, s, site.ID, "https://example.com/2", "Two")
	createTestBookmark(t, s, site.ID, "https://example.com/3", "Three")

	if _, err := s.BatchDeleteBookmarks(site.ID, []string{bm1.ID, bm2.ID}); err != nil {
		t.Fatalf("BatchDeleteBookmarks: %v", err)
	}

	siteAfter, _ := s.GetSite(site.ID)
	if siteAfter.BookmarkCount != 1 {
		t.Errorf("BookmarkCount after batch delete = %d, want 1", siteAfter.BookmarkCount)
	}
}

func TestBookmarkBatchDeleteRejectsCrossSiteIDs(t *testing.T) {
	s := newTestStore(t)
	siteA := createTestSite(t, s, "example.com")
	siteB := createTestSite(t, s, "another.com")
	bmA := createTestBookmark(t, s, siteA.ID, "https://example.com/1", "One")
	bmB := createTestBookmark(t, s, siteB.ID, "https://another.com/1", "Two")

	if _, err := s.BatchDeleteBookmarks(siteA.ID, []string{bmA.ID, bmB.ID}); err == nil {
		t.Fatal("BatchDeleteBookmarks should reject bookmarks from another site")
	}

	if _, err := s.GetBookmark(bmA.ID); err != nil {
		t.Fatalf("bookmark A should remain after rollback: %v", err)
	}
	if _, err := s.GetBookmark(bmB.ID); err != nil {
		t.Fatalf("bookmark B should remain after rollback: %v", err)
	}

	siteAAfter, _ := s.GetSite(siteA.ID)
	if siteAAfter.BookmarkCount != 1 {
		t.Errorf("site A BookmarkCount = %d, want 1", siteAAfter.BookmarkCount)
	}
	siteBAfter, _ := s.GetSite(siteB.ID)
	if siteBAfter.BookmarkCount != 1 {
		t.Errorf("site B BookmarkCount = %d, want 1", siteBAfter.BookmarkCount)
	}
}

func TestBookmarkList(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")
	createTestBookmark(t, s, site.ID, "https://example.com/a", "A")
	createTestBookmark(t, s, site.ID, "https://example.com/b", "B")
	createTestBookmark(t, s, site.ID, "https://example.com/c", "C")

	result, err := s.ListBookmarks(model.BookmarkListReq{
		SiteID:   site.ID,
		Page:     1,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("ListBookmarks: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("Items len = %d, want 2", len(result.Items))
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestBookmarkGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetBookmark("nonexistent-id")
	if err == nil {
		t.Error("GetBookmark should fail for nonexistent ID")
	}
}
