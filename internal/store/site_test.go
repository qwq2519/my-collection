package store

import (
	"testing"

	"collections/internal/model"
)

func createTestSite(t *testing.T, s *Store, domain string) *model.Site {
	t.Helper()
	site, err := s.CreateSite(model.CreateSiteReq{
		Title:  domain + " site",
		URL:    "https://" + domain,
		Domain: domain,
	})
	if err != nil {
		t.Fatalf("create site %s: %v", domain, err)
	}
	return site
}

func TestSiteCreate(t *testing.T) {
	s := newTestStore(t)
	site, err := s.CreateSite(model.CreateSiteReq{
		Title:       "GitHub",
		URL:         "https://github.com",
		Domain:      "github.com",
		Description: "Where the world builds software",
		Tags:        []string{"dev", "git"},
	})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}

	if site.ID == "" {
		t.Error("site ID should not be empty")
	}
	if site.Title != "GitHub" {
		t.Errorf("Title = %q, want %q", site.Title, "GitHub")
	}
	if site.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", site.Domain, "github.com")
	}
	if site.BookmarkCount != 0 {
		t.Errorf("BookmarkCount = %d, want 0", site.BookmarkCount)
	}
	if len(site.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(site.Tags))
	}
	if site.Attachments == nil {
		t.Error("Attachments should not be nil (EnsureSlices)")
	}
}

func TestSiteDuplicateDomain(t *testing.T) {
	s := newTestStore(t)
	createTestSite(t, s, "github.com")

	_, err := s.CreateSite(model.CreateSiteReq{
		Title:  "GitHub 2",
		URL:    "https://github.com",
		Domain: "github.com",
	})
	if err == nil {
		t.Error("expected error for duplicate domain, got nil")
	}
}

func TestSiteGetAndUpdate(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")

	got, err := s.GetSite(site.ID)
	if err != nil {
		t.Fatalf("GetSite: %v", err)
	}
	if got.Title != site.Title {
		t.Errorf("Title = %q, want %q", got.Title, site.Title)
	}

	newTitle := "Updated Title"
	newDesc := "new description"
	updated, err := s.UpdateSite(model.UpdateSiteReq{
		ID:          site.ID,
		Title:       &newTitle,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateSite: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("updated Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Description != newDesc {
		t.Errorf("updated Description = %q, want %q", updated.Description, newDesc)
	}
	if !updated.UpdatedAt.After(site.UpdatedAt) {
		t.Error("UpdatedAt should advance after update")
	}
}

func TestSiteDelete(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "example.com")

	if _, err := s.DeleteSite(site.ID); err != nil {
		t.Fatalf("DeleteSite: %v", err)
	}

	_, err := s.GetSite(site.ID)
	if err == nil {
		t.Error("GetSite should fail after delete")
	}
}

func TestSiteDeleteWithBookmarks(t *testing.T) {
	s := newTestStore(t)
	site := createTestSite(t, s, "github.com")

	_, err := s.CreateBookmark(model.CreateBookmarkReq{
		URL:    "https://github.com/golang/go",
		Domain: "github.com",
		SiteID: site.ID,
		Title:  "Go repo",
	})
	if err != nil {
		t.Fatalf("CreateBookmark: %v", err)
	}

	_, err = s.DeleteSite(site.ID)
	if err == nil {
		t.Error("DeleteSite should fail when site has bookmarks")
	}
}

func TestSiteList(t *testing.T) {
	s := newTestStore(t)
	createTestSite(t, s, "a.com")
	createTestSite(t, s, "b.com")
	createTestSite(t, s, "c.com")

	result, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("ListSites: %v", err)
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

	page2, err := s.ListSites(model.SiteListReq{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListSites page 2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("page 2 Items len = %d, want 1", len(page2.Items))
	}
	if page2.HasMore {
		t.Error("page 2 HasMore should be false")
	}
}

func TestSiteGetByDomain(t *testing.T) {
	s := newTestStore(t)
	createTestSite(t, s, "github.com")

	site, err := s.GetSiteByDomain("github.com")
	if err != nil {
		t.Fatalf("GetSiteByDomain: %v", err)
	}
	if site == nil {
		t.Fatal("site should not be nil")
	}
	if site.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", site.Domain, "github.com")
	}

	notFound, err := s.GetSiteByDomain("nonexistent.com")
	if err != nil {
		t.Fatalf("GetSiteByDomain nonexistent: %v", err)
	}
	if notFound != nil {
		t.Error("should return nil for nonexistent domain")
	}
}
