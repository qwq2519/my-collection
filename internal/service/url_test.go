package service

import (
	"testing"

	"collections/internal/model"
)

func TestURLService_CreateSiteValidation(t *testing.T) {
	svc := &URLService{Store: newTestStore(t)}

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
	svc := &URLService{Store: newTestStore(t)}

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
	svc := &URLService{Store: newTestStore(t)}

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
	svc := &URLService{Store: newTestStore(t)}
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
	svc := &URLService{Store: newTestStore(t)}
	svc.CreateSite(model.CreateSiteReq{Title: "Test", URL: "https://test.com"})

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
	svc := &URLService{Store: newTestStore(t)}
	site, _ := svc.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com",
		Tags: []string{"old"},
	})

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
	svc := &URLService{Store: newTestStore(t)}
	site, _ := svc.CreateSite(model.CreateSiteReq{
		Title: "Test", URL: "https://test.com",
		Tags: []string{"web"},
	})

	svc.DeleteSite(site.ID)

	tag, _ := svc.Store.GetTag("url_tag", "web")
	if tag != nil && tag.Count > 0 {
		t.Errorf("tag count after delete = %d, want 0", tag.Count)
	}
}

func TestURLService_CreateBookmarkValidation(t *testing.T) {
	svc := &URLService{Store: newTestStore(t)}

	_, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "not-a-url", Title: "Test",
	})
	if err == nil {
		t.Error("CreateBookmark should reject invalid URL")
	}

	_, err = svc.CreateBookmark(model.CreateBookmarkReq{
		URL: "https://example.com/page", Title: "",
	})
	if err == nil {
		t.Error("CreateBookmark should reject empty title")
	}
}

func TestURLService_CreateBookmarkAutoMatchSite(t *testing.T) {
	svc := &URLService{Store: newTestStore(t)}
	svc.CreateSite(model.CreateSiteReq{
		Title: "GitHub", URL: "https://github.com",
	})

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
	svc := &URLService{Store: newTestStore(t)}

	_, err := svc.CreateBookmark(model.CreateBookmarkReq{
		URL:   "https://unknown-domain.com/page",
		Title: "Test",
	})
	if err == nil {
		t.Error("CreateBookmark should fail when no site exists for domain")
	}
}

func TestURLService_UpdateBookmarkValidation(t *testing.T) {
	svc := &URLService{Store: newTestStore(t)}

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
	svc := &URLService{Store: newTestStore(t)}
	err := svc.DeleteBookmark("")
	if err == nil {
		t.Error("DeleteBookmark should reject empty ID")
	}
}

func TestURLService_LookupSiteByURL(t *testing.T) {
	svc := &URLService{Store: newTestStore(t)}
	svc.CreateSite(model.CreateSiteReq{Title: "GH", URL: "https://github.com"})

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
	svc := &URLService{Store: newTestStore(t)}

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
	svc := &URLService{Store: newTestStore(t)}

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
