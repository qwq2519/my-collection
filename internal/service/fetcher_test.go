package service

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"collections/internal/model"
	"collections/internal/store"
)

var (
	flagFetchURL = flag.String("fetch-url", "", "URL for TestFetch_Metadata / TestFetch_PageMeta")
	flagIconURL  = flag.String("icon-url", "", "icon URL for TestFetch_Icon")
	flagSaveDir  = flag.String("save-dir", "", "directory to save fetched files (default: t.TempDir())")
)

// TestFetch_Metadata 抓取指定 URL 的页面元数据（title、description、icon、og:image）。
// icon 会保存到临时 persist 目录（或 -save-dir 指定的目录）。
//
//	go test -run TestFetch_Metadata -v ./internal/service/ -fetch-url=https://github.com
//	go test -run TestFetch_Metadata -v ./internal/service/ -fetch-url=https://react.dev -save-dir=./tmp
func TestFetch_Metadata(t *testing.T) {
	rawURL := *flagFetchURL
	if rawURL == "" {
		t.Skip("use -fetch-url=https://example.com to fetch metadata")
	}

	svc := newURLServiceWithSaveDir(t)

	result, err := svc.FetchMetadata(model.FetchMetaReq{URL: rawURL})
	if err != nil {
		t.Fatalf("FetchMetadata: %v", err)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	t.Log(string(data))

	if result.Icon != "" {
		t.Logf("icon saved: %s", filepath.Join(svc.Store.PersistDir(), "url-assets", "icons", result.Icon))
	}
}

// TestFetch_PageMeta 仅解析 HTML 页面的 meta 信息（不下载 icon、不需要 Store）。
//
//	go test -run TestFetch_PageMeta -v ./internal/service/ -fetch-url=https://github.com
//	go test -run TestFetch_PageMeta -v ./internal/service/ -fetch-url=https://react.dev
func TestFetch_PageMeta(t *testing.T) {
	rawURL := *flagFetchURL
	if rawURL == "" {
		t.Skip("use -fetch-url=https://example.com to parse page meta")
	}

	meta, iconHrefs := fetchPageMeta(t.Context(), rawURL)

	t.Logf("title:       %s", meta.title)
	t.Logf("description: %s", meta.description)
	t.Logf("og:image:    %s", meta.ogImage)
	if len(iconHrefs) > 0 {
		t.Log("icon hrefs:")
		for _, href := range iconHrefs {
			t.Logf("  - %s", href)
		}
	}
}

// TestFetch_Icon 下载指定 URL 的图片/icon 并保存到本地。
//
//	go test -run TestFetch_Icon -v ./internal/service/ -icon-url="https://www.google.com/s2/favicons?domain=github.com&sz=64"
//	go test -run TestFetch_Icon -v ./internal/service/ -icon-url=https://react.dev/favicon.ico -save-dir=./tmp
func TestFetch_Icon(t *testing.T) {
	iconURL := *flagIconURL
	if iconURL == "" {
		t.Skip("use -icon-url=https://example.com/favicon.ico to download icon")
	}

	saveDir := *flagSaveDir
	if saveDir == "" {
		saveDir = t.TempDir()
	}
	os.MkdirAll(saveDir, 0755)

	resp, err := doGet(t.Context(), iconURL)
	if err != nil {
		t.Fatalf("GET %s: %v", iconURL, err)
	}
	defer resp.Body.Close()

	t.Logf("status: %d, content-type: %s, content-length: %s",
		resp.StatusCode, resp.Header.Get("Content-Type"), resp.Header.Get("Content-Length"))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxIconSize))
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	ext := guessExt(iconURL, ".png")
	filename := "downloaded-icon" + ext
	savePath := filepath.Join(saveDir, filename)
	if err := os.WriteFile(savePath, body, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	t.Logf("saved %d bytes to %s", len(body), savePath)
}

// newURLServiceWithSaveDir 创建 URLService，若指定 -save-dir 则用该目录作为 persist
// （main.db、search.bleve、url-assets 等都会写入该目录）。
func newURLServiceWithSaveDir(t *testing.T) *URLService {
	t.Helper()
	dir := *flagSaveDir
	if dir == "" {
		dir = t.TempDir()
	}
	os.MkdirAll(dir, 0755)

	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return &URLService{Store: s}
}

// ────────────────────── 纯函数单测（自动运行） ──────────────────────

func TestGuessExt(t *testing.T) {
	tests := []struct {
		url      string
		fallback string
		want     string
	}{
		{"https://example.com/icon.png", ".ico", ".png"},
		{"https://example.com/icon.ico", ".png", ".ico"},
		{"https://example.com/icon.svg", ".png", ".svg"},
		{"https://example.com/icon.jpg", ".png", ".jpg"},
		{"https://example.com/icon.jpeg", ".png", ".jpeg"},
		{"https://example.com/icon.gif", ".png", ".gif"},
		{"https://example.com/icon.webp", ".png", ".webp"},
		{"https://example.com/favicon", ".png", ".png"},
		{"https://example.com/path?q=1", ".ico", ".ico"},
		{"https://example.com/icon.bmp", ".png", ".png"},
		{"://invalid", ".png", ".png"},
	}
	for _, tt := range tests {
		got := guessExt(tt.url, tt.fallback)
		if got != tt.want {
			t.Errorf("guessExt(%q, %q) = %q, want %q", tt.url, tt.fallback, got, tt.want)
		}
	}
}

func TestResolveHref(t *testing.T) {
	tests := []struct {
		base string
		href string
		want string
	}{
		{"https://example.com/page", "/favicon.ico", "https://example.com/favicon.ico"},
		{"https://example.com/a/b", "../icon.png", "https://example.com/icon.png"},
		{"https://example.com", "https://cdn.example.com/icon.png", "https://cdn.example.com/icon.png"},
		{"https://example.com/page", "icon.png", "https://example.com/icon.png"},
	}
	for _, tt := range tests {
		got := resolveHref(tt.base, tt.href)
		if got != tt.want {
			t.Errorf("resolveHref(%q, %q) = %q, want %q", tt.base, tt.href, got, tt.want)
		}
	}
}

func TestResolveHrefInvalid(t *testing.T) {
	if got := resolveHref("://bad", "/icon.png"); got != "" {
		t.Errorf("resolveHref with bad base = %q, want empty", got)
	}
	if got := resolveHref("https://example.com", "://bad"); got != "" {
		t.Errorf("resolveHref with bad href = %q, want empty", got)
	}
}

func TestFetchPageMeta_WithHTTPTest(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>Test Page</title>
  <meta name="description" content="A test description">
  <meta property="og:image" content="https://example.com/og.jpg">
  <link rel="icon" href="/favicon.ico">
  <link rel="apple-touch-icon" href="/apple-icon.png">
</head>
<body>Hello</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer srv.Close()

	meta, iconHrefs := fetchPageMeta(t.Context(), srv.URL)

	if meta.title != "Test Page" {
		t.Errorf("title = %q, want %q", meta.title, "Test Page")
	}
	if meta.description != "A test description" {
		t.Errorf("description = %q, want %q", meta.description, "A test description")
	}
	if meta.ogImage != "https://example.com/og.jpg" {
		t.Errorf("ogImage = %q, want %q", meta.ogImage, "https://example.com/og.jpg")
	}
	if len(iconHrefs) != 2 {
		t.Fatalf("iconHrefs len = %d, want 2", len(iconHrefs))
	}
	if iconHrefs[0] != "/favicon.ico" {
		t.Errorf("iconHrefs[0] = %q, want %q", iconHrefs[0], "/favicon.ico")
	}
}

func TestFetchPageMeta_OGTitleOverrides(t *testing.T) {
	html := `<html><head>
  <title>Fallback Title</title>
  <meta property="og:title" content="OG Title">
  <meta property="og:description" content="OG Desc">
</head></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer srv.Close()

	meta, _ := fetchPageMeta(t.Context(), srv.URL)

	if meta.title != "OG Title" {
		t.Errorf("title = %q, want OG title to override <title>", meta.title)
	}
	if meta.description != "OG Desc" {
		t.Errorf("description = %q, want OG description", meta.description)
	}
}

func TestFetchPageMeta_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	meta, iconHrefs := fetchPageMeta(t.Context(), srv.URL)

	if meta.title != "" {
		t.Errorf("title = %q, want empty on 404", meta.title)
	}
	if len(iconHrefs) != 0 {
		t.Errorf("iconHrefs = %v, want empty on 404", iconHrefs)
	}
}
