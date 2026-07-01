package service

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
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
	fmt.Println(string(data))

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

	fmt.Printf("title:       %s\n", meta.title)
	fmt.Printf("description: %s\n", meta.description)
	fmt.Printf("og:image:    %s\n", meta.ogImage)
	if len(iconHrefs) > 0 {
		fmt.Printf("icon hrefs:\n")
		for _, href := range iconHrefs {
			fmt.Printf("  - %s\n", href)
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
