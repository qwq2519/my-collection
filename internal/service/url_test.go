package service

import (
	"testing"

	"collections/internal/model"
)

func TestURLService_CreateSiteValidation(t *testing.T) {
	svc := newURLService(t)

	tests := []struct {
		name string
		req  model.CreateSiteReq
	}{
		{"empty title", model.CreateSiteReq{Title: "", URL: "https://example.com"}},
		{"whitespace title", model.CreateSiteReq{Title: "   ", URL: "https://example.com"}},
		{"empty URL", model.CreateSiteReq{Title: "Example", URL: ""}},
	}
	for _, tt := range tests {
		_, err := svc.CreateSite(tt.req)
		if err == nil {
			t.Errorf("CreateSite[%s] should return error", tt.name)
		}
	}
}

func TestURLService_CreateSiteAutoExtractDomain(t *testing.T) {
	svc := newURLService(t)

	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub",
		URL:   "https://github.com/explore",
	})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	if site.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", site.Domain, "github.com")
	}
}

func TestURLService_CreateSiteTagNormalization(t *testing.T) {
	svc := newURLService(t)

	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Test",
		URL:   "https://test.com",
		Tags:  []string{"  Go  ", "go", "React"},
	})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}
	if len(site.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 items (deduplicated, normalized)", site.Tags)
	}
	if site.Tags[0] != "go" {
		t.Errorf("Tags[0] = %q, want %q (lowercased, trimmed)", site.Tags[0], "go")
	}
}

func TestURLService_CreateSiteInvalidTag(t *testing.T) {
	svc := newURLService(t)
	_, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Test",
		URL:   "https://test.com",
		Tags:  []string{"tag/invalid"},
	})
	if err == nil {
		t.Error("CreateSite should reject invalid tag names")
	}
}

func TestURLService_UpdateSiteValidation(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{Title: "Test", URL: "https://test.com"}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	emptyID := ""
	_, err := svc.UpdateSite(model.UpdateSiteReq{ID: emptyID})
	if err == nil {
		t.Error("UpdateSite should reject empty ID")
	}

	emptyTitle := "   "
	_, err = svc.UpdateSite(model.UpdateSiteReq{ID: "some-id", Title: &emptyTitle})
	if err == nil {
		t.Error("UpdateSite should reject empty title")
	}
}

func TestURLService_UpdateSiteTagCountAdjustment(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com",
		Tags: []string{"old"},
	})
	if err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	newTags := []string{"new"}
	updated, err := svc.UpdateSite(model.UpdateSiteReq{ID: site.ID, Tags: &newTags})
	if err != nil {
		t.Fatalf("UpdateSite: %v", err)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "new" {
		t.Errorf("updated Tags = %v, want [new]", updated.Tags)
	}
}

func TestURLService_DeleteSiteTagCountDecrease(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com",
		Tags: []string{"web"},
	})
	if err != nil {
		t.Fatalf("CreateSite: %v", err)
	}

	tag, err := svc.Store.GetTag("url_tag", "web")
	if err != nil {
		t.Fatalf("GetTag after create: %v", err)
	}
	if tag == nil || tag.Count != 1 {
		t.Fatalf("tag count after create = %v, want 1", tag)
	}

	if err := svc.DeleteSite(site.ID); err != nil {
		t.Fatalf("DeleteSite: %v", err)
	}

	tag, err = svc.Store.GetTag("url_tag", "web")
	if err != nil {
		t.Fatalf("GetTag after delete: %v", err)
	}
	if tag != nil && tag.Count > 0 {
		t.Errorf("tag count after delete = %d, want 0", tag.Count)
	}
}

func TestURLService_CreateBookmarkValidation(t *testing.T) {
	svc := newURLService(t)

	tests := []struct {
		name string
		req  model.CreateBookmarkReq
	}{
		{"invalid URL", model.CreateBookmarkReq{URL: "not-a-url", Title: "Test"}},
		{"empty title", model.CreateBookmarkReq{URL: "https://example.com/page", Title: ""}},
		{"whitespace title", model.CreateBookmarkReq{URL: "https://example.com/page", Title: "   "}},
	}
	for _, tt := range tests {
		_, err := svc.CreateBookmark(tt.req)
		if err == nil {
			t.Errorf("CreateBookmark[%s] should return error", tt.name)
		}
	}
}

func TestURLService_CreateBookmarkAutoMatchSite(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub", URL: "https://github.com",
	}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	bm, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL:   "https://github.com/golang/go",
		Title: "Go Repo",
	})
	if err != nil {
		t.Fatalf("CreateBookmark: %v", err)
	}
	if bm.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", bm.Domain, "github.com")
	}
	if bm.SiteID == "" {
		t.Error("SiteID should be auto-filled")
	}
}

func TestURLService_CreateBookmarkNoSite(t *testing.T) {
	svc := newURLService(t)

	_, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL:   "https://unknown-domain.com/page",
		Title: "Test",
	})
	if err == nil {
		t.Error("CreateBookmark should fail when no site exists for domain")
	}
}

func TestURLService_UpdateBookmarkValidation(t *testing.T) {
	svc := newURLService(t)

	_, err := svc.UpdateBookmark(model.UpdateBookmarkReq{ID: ""})
	if err == nil {
		t.Error("UpdateBookmark should reject empty ID")
	}

	emptyTitle := "   "
	_, err = svc.UpdateBookmark(model.UpdateBookmarkReq{ID: "x", Title: &emptyTitle})
	if err == nil {
		t.Error("UpdateBookmark should reject whitespace title")
	}
}

func TestURLService_DeleteBookmarkValidation(t *testing.T) {
	svc := newURLService(t)
	err := svc.DeleteBookmark("")
	if err == nil {
		t.Error("DeleteBookmark should reject empty ID")
	}
}

func TestURLService_LookupSiteByURL(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{Title: "GH", URL: "https://github.com"}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	result, err := svc.LookupSiteByURL(model.LookupSiteByURLReq{
		URL: "https://github.com/golang/go",
	})
	if err != nil {
		t.Fatalf("LookupSiteByURL: %v", err)
	}
	if !result.Found {
		t.Error("should find site for github.com")
	}
	if result.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", result.Domain, "github.com")
	}

	result2, _ := svc.LookupSiteByURL(model.LookupSiteByURLReq{
		URL: "https://unknown.org/page",
	})
	if result2.Found {
		t.Error("should not find site for unknown.org")
	}
}

func TestURLService_LookupSiteByURLValidation(t *testing.T) {
	svc := newURLService(t)

	_, err := svc.LookupSiteByURL(model.LookupSiteByURLReq{URL: ""})
	if err == nil {
		t.Error("LookupSiteByURL should reject empty URL")
	}

	_, err = svc.LookupSiteByURL(model.LookupSiteByURLReq{URL: "not-valid"})
	if err == nil {
		t.Error("LookupSiteByURL should reject invalid URL")
	}
}

func TestURLService_NormalizeURL(t *testing.T) {
	svc := newURLService(t)

	result, err := svc.NormalizeURL("https://www.Example.COM/page/")
	if err != nil {
		t.Fatalf("NormalizeURL: %v", err)
	}
	if result != "https://example.com/page" {
		t.Errorf("NormalizeURL = %q, want %q", result, "https://example.com/page")
	}

	empty, err := svc.NormalizeURL("")
	if err != nil {
		t.Fatalf("NormalizeURL empty: %v", err)
	}
	if empty != "" {
		t.Errorf("NormalizeURL('') = %q, want empty", empty)
	}
}

// ────────────────────── GetSite ──────────────────────

func TestURLService_GetSite(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub", URL: "https://github.com",
	})
	if err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	got, err := svc.GetSite(site.ID)
	if err != nil {
		t.Fatalf("GetSite: %v", err)
	}
	if got.Title != "GitHub" {
		t.Errorf("Title = %q, want %q", got.Title, "GitHub")
	}
	if got.Domain != "github.com" {
		t.Errorf("Domain = %q, want %q", got.Domain, "github.com")
	}
}

func TestURLService_GetSiteNotFound(t *testing.T) {
	svc := newURLService(t)
	_, err := svc.GetSite("nonexistent-id")
	if err == nil {
		t.Error("GetSite should fail for nonexistent ID")
	}
}

// ────────────────────── ListSites ──────────────────────

func TestURLService_ListSites(t *testing.T) {
	svc := newURLService(t)
	for _, u := range []string{"https://a.com", "https://b.com", "https://c.com"} {
		if _, err := svc.CreateSite(model.CreateSiteReq{Title: u, URL: u}); err != nil {
			t.Fatalf("setup CreateSite(%s): %v", u, err)
		}
	}

	result, err := svc.ListSites(model.SiteListReq{Page: 1, PageSize: 2})
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

	page2, err := svc.ListSites(model.SiteListReq{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("ListSites page 2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Errorf("page 2 Items len = %d, want 1", len(page2.Items))
	}
}

// ────────────────────── GetBookmark ──────────────────────

func TestURLService_GetBookmark(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub", URL: "https://github.com",
	}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	bm, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://github.com/golang/go", Title: "Go Repo",
	})
	if err != nil {
		t.Fatalf("setup CreateBookmark: %v", err)
	}

	got, err := svc.GetBookmark(bm.ID)
	if err != nil {
		t.Fatalf("GetBookmark: %v", err)
	}
	if got.Title != "Go Repo" {
		t.Errorf("Title = %q, want %q", got.Title, "Go Repo")
	}
	if got.SiteID == "" {
		t.Error("SiteID should not be empty")
	}
}

func TestURLService_GetBookmarkNotFound(t *testing.T) {
	svc := newURLService(t)
	_, err := svc.GetBookmark("nonexistent-id")
	if err == nil {
		t.Error("GetBookmark should fail for nonexistent ID")
	}
}

// ────────────────────── ListBookmarks ──────────────────────

func TestURLService_ListBookmarks(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Example", URL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	for i, path := range []string{"/a", "/b", "/c"} {
		if _, err := svc.CreateBookmark(model.CreateBookmarkReq{
			URL: "https://example.com" + path, Title: string(rune('A' + i)),
		}); err != nil {
			t.Fatalf("setup CreateBookmark(%s): %v", path, err)
		}
	}

	result, err := svc.ListBookmarks(model.BookmarkListReq{
		SiteID: site.ID, Page: 1, PageSize: 2,
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

// ────────────────────── BatchDeleteBookmarks ──────────────────────

func TestURLService_BatchDeleteBookmarks(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Example", URL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	var ids []string
	for _, path := range []string{"/a", "/b", "/c"} {
		bm, err := svc.CreateBookmark(model.CreateBookmarkReq{
			URL: "https://example.com" + path, Title: "BM",
			Tags: []string{"batch"},
		})
		if err != nil {
			t.Fatalf("setup CreateBookmark: %v", err)
		}
		ids = append(ids, bm.ID)
	}

	if err := svc.BatchDeleteBookmarks(site.ID, ids[:2]); err != nil {
		t.Fatalf("BatchDeleteBookmarks: %v", err)
	}

	remaining, err := svc.ListBookmarks(model.BookmarkListReq{
		SiteID: site.ID, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListBookmarks: %v", err)
	}
	if remaining.Total != 1 {
		t.Errorf("remaining Total = %d, want 1", remaining.Total)
	}

	got, _ := svc.Store.GetSite(site.ID)
	if got.BookmarkCount != 1 {
		t.Errorf("BookmarkCount = %d, want 1", got.BookmarkCount)
	}
}

// ────────────────────── BatchTagBookmarks ──────────────────────

func TestURLService_BatchTagBookmarks(t *testing.T) {
	svc := newURLService(t)
	site, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Example", URL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	bm1, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://example.com/a", Title: "A", Tags: []string{"existing"},
	})
	if err != nil {
		t.Fatalf("setup CreateBookmark: %v", err)
	}
	bm2, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://example.com/b", Title: "B",
	})
	if err != nil {
		t.Fatalf("setup CreateBookmark: %v", err)
	}
	_ = site

	if err := svc.BatchTagBookmarks([]string{bm1.ID, bm2.ID}, []string{"newtag"}); err != nil {
		t.Fatalf("BatchTagBookmarks: %v", err)
	}

	got1, _ := svc.GetBookmark(bm1.ID)
	if len(got1.Tags) != 2 {
		t.Errorf("bm1 Tags = %v, want [existing newtag]", got1.Tags)
	}

	got2, _ := svc.GetBookmark(bm2.ID)
	if len(got2.Tags) != 1 || got2.Tags[0] != "newtag" {
		t.Errorf("bm2 Tags = %v, want [newtag]", got2.Tags)
	}
}

func TestURLService_BatchTagBookmarksDedup(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Example", URL: "https://example.com",
	}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	bm, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://example.com/a", Title: "A", Tags: []string{"go"},
	})
	if err != nil {
		t.Fatalf("setup CreateBookmark: %v", err)
	}

	if err := svc.BatchTagBookmarks([]string{bm.ID}, []string{"go"}); err != nil {
		t.Fatalf("BatchTagBookmarks: %v", err)
	}

	got, _ := svc.GetBookmark(bm.ID)
	if len(got.Tags) != 1 {
		t.Errorf("Tags = %v, want [go] (no duplicate)", got.Tags)
	}
}

func TestURLService_BatchTagBookmarksEmpty(t *testing.T) {
	svc := newURLService(t)
	if err := svc.BatchTagBookmarks(nil, nil); err != nil {
		t.Errorf("empty batch should return nil, got: %v", err)
	}
}

// ────────────────────── SearchURL ──────────────────────

func TestURLService_SearchURL(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "React Docs", URL: "https://react.dev",
		Tags: []string{"frontend"},
	}); err != nil {
		t.Fatalf("setup CreateSite react: %v", err)
	}
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "Go Official", URL: "https://go.dev",
		Tags: []string{"backend"},
	}); err != nil {
		t.Fatalf("setup CreateSite go: %v", err)
	}

	result, err := svc.SearchURL(model.SearchURLReq{
		Search: "react", Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("SearchURL: %v", err)
	}
	if result.Total < 1 {
		t.Errorf("Total = %d, want >= 1", result.Total)
	}

	found := false
	for _, item := range result.Items {
		if item.Site.Domain == "react.dev" {
			found = true
		}
	}
	if !found {
		t.Error("search for 'react' should find react.dev site")
	}
}

func TestURLService_SearchURLByTag(t *testing.T) {
	svc := newURLService(t)
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub", URL: "https://github.com",
		Tags: []string{"dev"},
	}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}
	if _, err := svc.CreateSite(model.CreateSiteReq{
		Title: "MDN", URL: "https://developer.mozilla.org",
		Tags: []string{"docs"},
	}); err != nil {
		t.Fatalf("setup CreateSite: %v", err)
	}

	result, err := svc.SearchURL(model.SearchURLReq{
		Tags: []string{"dev"}, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("SearchURL by tag: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}
}

func TestURLService_SearchURLValidation(t *testing.T) {
	svc := newURLService(t)
	_, err := svc.SearchURL(model.SearchURLReq{Page: 1, PageSize: 10})
	if err == nil {
		t.Error("SearchURL should reject empty search and empty tags")
	}
}
