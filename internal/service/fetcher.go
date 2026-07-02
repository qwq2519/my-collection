package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"golang.org/x/net/html"
)

const (
	fetchTimeout      = 5 * time.Second
	totalFetchTimeout = 15 * time.Second
	maxBodySize       = 512 << 10
	maxIconSize       = 1 << 20
	minIconBytes      = 100
	userAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
)

var fetchClient = &http.Client{Timeout: fetchTimeout}

// FetchMetadata 抓取 URL 页面元数据（title、description、icon、og:image）。
// icon 当前仅使用 Google S2 获取。
// 单次 HTTP 请求超时 5 秒，整体超时上限 15 秒，不缓存抓取结果。
func (u *URLService) FetchMetadata(req model.FetchMetaReq) (_ *model.FetchMetaResult, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("URL required")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), totalFetchTimeout)
	defer cancel()

	result := &model.FetchMetaResult{}

	meta, iconHrefs := fetchPageMeta(ctx, req.URL)
	result.Title = meta.title
	result.Description = meta.description
	result.OGImage = meta.ogImage

	result.Icon = u.fetchAndSaveIcon(ctx, domain, req.URL, iconHrefs)

	return result, nil
}

// --- HTML 解析 ---

type pageMeta struct {
	title       string
	description string
	ogImage     string
}

// fetchPageMeta 抓取页面 HTML 并解析 title/description/og:image 和 icon href 列表
func fetchPageMeta(ctx context.Context, rawURL string) (pageMeta, []string) {
	pm := pageMeta{}

	resp, err := doGet(ctx, rawURL)
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
	var hasOGTitle, hasOGDesc bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if !hasOGTitle && pm.title == "" {
					var sb strings.Builder
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							sb.WriteString(c.Data)
						}
					}
					if t := strings.TrimSpace(sb.String()); t != "" {
						pm.title = t
					}
				}
			case "meta":
				name, content := metaAttrs(n)
				switch strings.ToLower(name) {
				case "description":
					if !hasOGDesc && pm.description == "" {
						pm.description = content
					}
				case "og:title":
					if content != "" {
						pm.title = content
						hasOGTitle = true
					}
				case "og:description":
					if content != "" {
						pm.description = content
						hasOGDesc = true
					}
				case "og:image":
					if pm.ogImage == "" {
						pm.ogImage = content
					}
				}
			case "link":
				rel, href := linkAttrs(n)
				if strings.Contains(strings.ToLower(rel), "icon") && href != "" && !strings.HasPrefix(href, "data:") {
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

// metaAttrs 从 <meta> 节点提取 name/property 和 content 属性值
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

// linkAttrs 从 <link> 节点提取 rel 和 href 属性值
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

// --- Icon ---

// fetchAndSaveIcon 尝试从 Google S2 获取站点图标并保存到 persist/url-assets/icons/
func (u *URLService) fetchAndSaveIcon(ctx context.Context, domain, pageURL string, iconHrefs []string) string {
	iconsDir := filepath.Join(u.Store.PersistDir(), "url-assets", "icons")

	// Google S2
	if data := fetchIconData(ctx, fmt.Sprintf(
		"https://www.google.com/s2/favicons?domain=%s&sz=64", domain,
	)); len(data) > minIconBytes {
		filename := domain + ".png"
		if saveIcon(iconsDir, filename, data) == nil {
			slog.Info("icon from google s2", "domain", domain)
			return filename
		}
	}

	// // DuckDuckGo（暂时禁用）
	// if data := fetchIconData(ctx, fmt.Sprintf(
	// 	"https://icons.duckduckgo.com/ip3/%s.ico", domain,
	// )); len(data) > minIconBytes {
	// 	filename := domain + ".ico"
	// 	if saveIcon(iconsDir, filename, data) == nil {
	// 		slog.Info("icon from duckduckgo", "domain", domain)
	// 		return filename
	// 	}
	// }

	// // HTML <link rel="icon">（暂时禁用）
	// for _, href := range iconHrefs {
	// 	iconURL := resolveHref(pageURL, href)
	// 	if iconURL == "" {
	// 		continue
	// 	}
	// 	data := fetchIconData(ctx, iconURL)
	// 	if len(data) <= minIconBytes {
	// 		continue
	// 	}
	// 	ext := guessExt(iconURL, ".png")
	// 	filename := domain + ext
	// 	if saveIcon(iconsDir, filename, data) == nil {
	// 		slog.Info("icon from html link", "domain", domain, "src", iconURL)
	// 		return filename
	// 	}
	// }

	slog.Info("no icon found", "domain", domain)
	return ""
}

// fetchIconData 下载图标数据，失败或非 200 时返回 nil
func fetchIconData(ctx context.Context, iconURL string) []byte {
	resp, err := doGet(ctx, iconURL)
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

// saveIcon 将图标数据原子写入到指定目录
func saveIcon(dir, filename string, data []byte) error {
	return util.AtomicWrite(filepath.Join(dir, filename), data, 0644)
}

// --- 工具函数 ---

// doGet 发起带 context 和 User-Agent 的 HTTP GET 请求
func doGet(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return fetchClient.Do(req)
}

// resolveHref 将相对 href 解析为绝对 URL，解析失败返回空串
func resolveHref(base, href string) string {
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

// guessExt 从 URL 路径猜测图片扩展名，无法识别时返回 fallback
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
