package service

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"golang.org/x/net/html"
)

const (
	fetchTimeout = 10 * time.Second
	maxBodySize  = 512 << 10
	maxIconSize  = 1 << 20
	minIconBytes = 100
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

var fetchClient = &http.Client{Timeout: fetchTimeout}

// FetchMetadata 抓取 URL 页面元数据（title、description、icon、og:image）。
// icon 使用多级降级：Google S2 → DuckDuckGo → 解析 HTML <link rel="icon"> → 为空。
// 超时上限 10 秒，不缓存抓取结果。
func (u *URLService) FetchMetadata(req model.FetchMetaReq) (_ *model.FetchMetaResult, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}
	if err := util.ValidateURL(req.URL); err != nil {
		return nil, err
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, err
	}

	result := &model.FetchMetaResult{}

	meta, iconHrefs := fetchPageMeta(req.URL)
	result.Title = meta.title
	result.Description = meta.description
	result.OGImage = meta.ogImage

	result.Icon = u.fetchAndSaveIcon(domain, req.URL, iconHrefs)

	return result, nil
}

// --- HTML 解析 ---

type pageMeta struct {
	title       string
	description string
	ogImage     string
}

func fetchPageMeta(rawURL string) (pageMeta, []string) {
	pm := pageMeta{}

	resp, err := doGet(rawURL)
	if err != nil {
		slog.Warn("fetch page failed", "url", rawURL, "err", err)
		return pm, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("fetch page non-200", "url", rawURL, "status", resp.StatusCode)
		return pm, nil
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		slog.Warn("parse html failed", "url", rawURL, "err", err)
		return pm, nil
	}

	var iconHrefs []string
	var hasOGTitle bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil && !hasOGTitle && pm.title == "" {
					pm.title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				name, content := metaAttrs(n)
				switch strings.ToLower(name) {
				case "description":
					if pm.description == "" {
						pm.description = content
					}
				case "og:title":
					if content != "" {
						pm.title = content
						hasOGTitle = true
					}
				case "og:description":
					if pm.description == "" {
						pm.description = content
					}
				case "og:image":
					if pm.ogImage == "" {
						pm.ogImage = content
					}
				}
			case "link":
				rel, href := linkAttrs(n)
				if strings.Contains(strings.ToLower(rel), "icon") && href != "" {
					iconHrefs = append(iconHrefs, href)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return pm, iconHrefs
}

func metaAttrs(n *html.Node) (name, content string) {
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "name", "property":
			name = a.Val
		case "content":
			content = a.Val
		}
	}
	return
}

func linkAttrs(n *html.Node) (rel, href string) {
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "rel":
			rel = a.Val
		case "href":
			href = a.Val
		}
	}
	return
}

// --- Icon 多级降级 ---

func (u *URLService) fetchAndSaveIcon(domain, pageURL string, iconHrefs []string) string {
	iconsDir := filepath.Join(u.Store.PersistDir(), "url-assets", "icons")
	os.MkdirAll(iconsDir, 0755)

	// Level 1: Google S2
	if data := fetchIconData(fmt.Sprintf(
		"https://www.google.com/s2/favicons?domain=%s&sz=64", domain,
	)); len(data) > minIconBytes {
		filename := domain + ".png"
		if saveIcon(iconsDir, filename, data) == nil {
			slog.Info("icon from google s2", "domain", domain)
			return filename
		}
	}

	// Level 2: DuckDuckGo
	if data := fetchIconData(fmt.Sprintf(
		"https://icons.duckduckgo.com/ip3/%s.ico", domain,
	)); len(data) > minIconBytes {
		filename := domain + ".ico"
		if saveIcon(iconsDir, filename, data) == nil {
			slog.Info("icon from duckduckgo", "domain", domain)
			return filename
		}
	}

	// Level 3: HTML <link rel="icon">
	for _, href := range iconHrefs {
		iconURL := resolveHref(pageURL, href)
		if iconURL == "" {
			continue
		}
		data := fetchIconData(iconURL)
		if len(data) <= minIconBytes {
			continue
		}
		ext := guessExt(iconURL, ".png")
		filename := domain + ext
		if saveIcon(iconsDir, filename, data) == nil {
			slog.Info("icon from html link", "domain", domain, "src", iconURL)
			return filename
		}
	}

	// Level 4: 留空，前端展示占位 icon
	slog.Info("no icon found", "domain", domain)
	return ""
}

func fetchIconData(iconURL string) []byte {
	resp, err := doGet(iconURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIconSize))
	if err != nil {
		return nil
	}
	return data
}

func saveIcon(dir, filename string, data []byte) error {
	return util.AtomicWrite(filepath.Join(dir, filename), data, 0644)
}

// --- 工具函数 ---

func doGet(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return fetchClient.Do(req)
}

func resolveHref(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(ref).String()
}

func guessExt(rawURL, fallback string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fallback
	}
	ext := strings.ToLower(filepath.Ext(u.Path))
	switch ext {
	case ".png", ".ico", ".svg", ".jpg", ".jpeg", ".gif", ".webp":
		return ext
	default:
		return fallback
	}
}
