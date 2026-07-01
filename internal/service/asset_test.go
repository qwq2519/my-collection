package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func newTestAssetHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	persistDir := t.TempDir()
	embedded := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("embedded"))
	})
	return NewAssetHandler(embedded, persistDir), persistDir
}

func TestAssetHandler_ServesWhitelistedPath(t *testing.T) {
	handler, persistDir := newTestAssetHandler(t)

	iconDir := filepath.Join(persistDir, "url-assets", "icons")
	os.MkdirAll(iconDir, 0755)
	os.WriteFile(filepath.Join(iconDir, "github.com.png"), []byte("icon-data"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/persist/url-assets/icons/github.com.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "icon-data" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "icon-data")
	}
}

func TestAssetHandler_CacheControlOnSuccess(t *testing.T) {
	handler, persistDir := newTestAssetHandler(t)

	dir := filepath.Join(persistDir, "note-images", "abc")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "img.png"), []byte("png"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/persist/note-images/abc/img.png", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc == "" {
		t.Error("Cache-Control should be set on success")
	}
}

func TestAssetHandler_RejectsNonWhitelistedPath(t *testing.T) {
	handler, _ := newTestAssetHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/persist/main.db", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for non-whitelisted path", rec.Code)
	}
}

func TestAssetHandler_RejectsTraversal(t *testing.T) {
	handler, _ := newTestAssetHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/persist/url-assets/../main.db", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("should not serve file via path traversal")
	}
}

func TestAssetHandler_RejectsNonGET(t *testing.T) {
	handler, _ := newTestAssetHandler(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/persist/url-assets/icons/test.png", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s status = %d, want 405", method, rec.Code)
		}
	}
}

func TestAssetHandler_DelegatesToEmbedded(t *testing.T) {
	handler, _ := newTestAssetHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "embedded" {
		t.Errorf("body = %q, want %q (should delegate to embedded)", rec.Body.String(), "embedded")
	}
}

func TestAssetHandler_AllWhitelistPrefixes(t *testing.T) {
	handler, persistDir := newTestAssetHandler(t)

	prefixes := []string{"url-assets", "note-images", "media-folders"}
	for _, p := range prefixes {
		dir := filepath.Join(persistDir, p)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "test.txt"), []byte("ok"), 0644)

		req := httptest.NewRequest(http.MethodGet, "/persist/"+p+"/test.txt", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("prefix %q: status = %d, want 200", p, rec.Code)
		}
	}
}
