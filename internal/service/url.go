package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"
)

// URLService 站点 + 书签的业务逻辑层。
// 公开方法即前端可调用接口（通过 Wails 绑定）。
type URLService struct {
	Store *store.Store
}

// ────────────────────── Site ──────────────────────

// CreateSite 创建站点。校验 title 必填、URL 合法性，
// 从 URL 自动提取 Domain，归一化标签并维护 url_tag 注册表 count。
func (u *URLService) CreateSite(req model.CreateSiteReq) (_ *model.Site, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("site title required")
	}
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("site URL required")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	req.Domain = domain

	tags, err := util.NormalizeTags(req.Tags)
	if err != nil {
		return nil, err
	}
	req.Tags = tags

	site, err := u.Store.CreateSite(req)
	if err != nil {
		return nil, err
	}

	u.adjustURLTagCounts(site.Tags, nil)
	return site, nil
}

// GetSite 按 ID 查询站点详情
func (u *URLService) GetSite(id string) (_ *model.Site, err error) {
	defer logError(&err)
	if id == "" {
		return nil, fmt.Errorf("site ID required")
	}
	return u.Store.GetSite(id)
}

// UpdateSite 更新站点（URL/Domain 不可修改）。
// 指针字段为 nil 表示不更新；Tags 传空切片表示清空所有标签。
// 当 Tags 字段变化时维护 url_tag 注册表 count。
func (u *URLService) UpdateSite(req model.UpdateSiteReq) (_ *model.Site, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("site ID required")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, fmt.Errorf("site title required")
	}

	if req.Tags != nil {
		tags, err := util.NormalizeTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		req.Tags = &tags
	}

	site, oldTags, err := u.Store.UpdateSite(req)
	if err != nil {
		return nil, err
	}

	if oldTags != nil {
		u.adjustURLTagCounts(site.Tags, oldTags)
	}
	return site, nil
}

// DeleteSite 删除站点。同步清理 icon、封面、附件目录，
// 并递减 url_tag 注册表 count。
func (u *URLService) DeleteSite(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("site ID required")
	}

	site, err := u.Store.DeleteSite(id)
	if err != nil {
		return err
	}

	u.adjustURLTagCounts(nil, site.Tags)
	u.cleanSiteAssets(site)
	return nil
}

// ListSites 分页查询站点列表（纯列表，不含搜索）
func (u *URLService) ListSites(req model.SiteListReq) (_ *model.SiteListResult, err error) {
	defer logError(&err)
	return u.Store.ListSites(req)
}

// SearchURL 统一搜索站点和书签，按站点分组返回。
// 同时匹配站点和书签的 title/description/domain，命中的书签归入所属站点。
// 至少需要 Search 或 Tags 之一非空。
func (u *URLService) SearchURL(req model.SearchURLReq) (_ *model.SearchURLResult, err error) {
	defer logError(&err)
	if req.Search == "" && len(req.Tags) == 0 {
		return nil, fmt.Errorf("search keyword or tags required")
	}
	return u.Store.SearchURL(req)
}

// ────────────────────── Bookmark ──────────────────────

// CreateBookmark 创建书签。校验 URL 合法性，自动提取域名匹配站点，
// 归一化标签并维护 url_tag 注册表 count。
func (u *URLService) CreateBookmark(req model.CreateBookmarkReq) (_ *model.Bookmark, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("bookmark title required")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, err
	}
	site, err := u.Store.GetSiteByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("lookup site failed: %w", err)
	}
	if site == nil {
		return nil, fmt.Errorf("no site for domain %q, create one first", domain)
	}
	req.SiteID = site.ID
	req.Domain = domain

	tags, err := util.NormalizeTags(req.Tags)
	if err != nil {
		return nil, err
	}
	req.Tags = tags

	bm, err := u.Store.CreateBookmark(req)
	if err != nil {
		return nil, err
	}

	u.adjustURLTagCounts(bm.Tags, nil)
	return bm, nil
}

// GetBookmark 按 ID 查询书签详情
func (u *URLService) GetBookmark(id string) (_ *model.Bookmark, err error) {
	defer logError(&err)
	if id == "" {
		return nil, fmt.Errorf("bookmark ID required")
	}
	return u.Store.GetBookmark(id)
}

// UpdateBookmark 更新书签（URL/Domain/SiteID 不可修改）。
// 指针字段为 nil 表示不更新；Tags 传空切片表示清空所有标签。
// 当 Tags 字段变化时维护 url_tag 注册表 count。
func (u *URLService) UpdateBookmark(req model.UpdateBookmarkReq) (_ *model.Bookmark, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("bookmark ID required")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, fmt.Errorf("bookmark title required")
	}

	if req.Tags != nil {
		tags, err := util.NormalizeTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		req.Tags = &tags
	}

	bm, oldTags, err := u.Store.UpdateBookmark(req)
	if err != nil {
		return nil, err
	}

	if oldTags != nil {
		u.adjustURLTagCounts(bm.Tags, oldTags)
	}
	return bm, nil
}

// DeleteBookmark 删除单条书签。同步清理封面和附件文件，
// 并递减 url_tag 注册表 count。
func (u *URLService) DeleteBookmark(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("bookmark ID required")
	}

	bm, err := u.Store.DeleteBookmark(id)
	if err != nil {
		return err
	}

	u.adjustURLTagCounts(nil, bm.Tags)
	u.cleanBookmarkAssets(bm)
	return nil
}

// BatchDeleteBookmarks 批量删除同一站点下的书签。
// ids 为空时静默返回 nil。汇总所有被删除书签的标签后一次性更新
// url_tag 注册表 count，并逐个清理文件资源。不存在的 ID 跳过并记录日志。
func (u *URLService) BatchDeleteBookmarks(siteID string, ids []string) (err error) {
	defer logError(&err)
	if len(ids) == 0 {
		return nil
	}

	deleted, err := u.Store.BatchDeleteBookmarks(siteID, ids)
	if err != nil {
		return err
	}

	var allOldTags []string
	for _, bm := range deleted {
		allOldTags = append(allOldTags, bm.Tags...)
		u.cleanBookmarkAssets(bm)
	}
	u.adjustURLTagCounts(nil, allOldTags)
	return nil
}

// BatchTagBookmarks 批量为书签追加标签（不覆盖已有标签）。
// ids 或 tagsToAdd 为空时静默返回 nil。同步维护 url_tag 注册表 count。
// 不存在的 ID 跳过并记录日志。
func (u *URLService) BatchTagBookmarks(ids []string, tagsToAdd []string) (err error) {
	defer logError(&err)
	if len(ids) == 0 || len(tagsToAdd) == 0 {
		return nil
	}

	tags, err := util.NormalizeTags(tagsToAdd)
	if err != nil {
		return err
	}

	deltas, err := u.Store.BatchAppendBookmarkTags(ids, tags)
	if err != nil {
		return err
	}

	if len(deltas) > 0 {
		if err := u.Store.BatchAdjustTagCounts("url_tag", deltas); err != nil {
			slog.Warn("failed to adjust url tag counts after batch tag", "err", err)
		}
	}
	return nil
}

// ListBookmarks 分页查询书签列表
func (u *URLService) ListBookmarks(req model.BookmarkListReq) (_ *model.BookmarkListResult, err error) {
	defer logError(&err)
	return u.Store.ListBookmarks(req)
}

// cleanBookmarkAssets 删除书签关联的文件资源（附件目录）
func (u *URLService) cleanBookmarkAssets(bm *model.Bookmark) {
	assetsDir := filepath.Join(u.Store.PersistDir(), "url-assets")
	u.cleanEntityAttachments(assetsDir, bm.ID)
}

// ────────────────────── Normalize ──────────────────────

// NormalizeURL 归一化 URL（去协议、去 www、去尾部斜杠等），
// 供前端输入时实时预览标准化后的地址。空串输入返回 ("", nil)。
func (u *URLService) NormalizeURL(rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", nil
	}
	return util.NormalizeURL(rawURL)
}

// ────────────────────── Lookup ──────────────────────

// LookupSiteByURL 根据 URL 提取域名并查找对应站点，供前端在新增书签前判断是否需要先创建站点。
// URL 必填且需通过 ValidateURL 校验。未找到站点时返回 Found=false（不报错）。
func (u *URLService) LookupSiteByURL(req model.LookupSiteByURLReq) (_ *model.LookupSiteResult, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("URL required")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, err
	}

	site, err := u.Store.GetSiteByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("lookup site failed: %w", err)
	}

	result := &model.LookupSiteResult{Domain: domain}
	if site != nil {
		result.Found = true
		result.ID = site.ID
		result.URL = site.URL
	}
	return result, nil
}

// ────────────────────── helpers ──────────────────────

// adjustURLTagCounts 计算新旧标签的差值，批量更新 url_tag 注册表 count。
// newTags 为新标签列表（创建/更新后），oldTags 为旧标签列表（更新/删除前）。
func (u *URLService) adjustURLTagCounts(newTags, oldTags []string) {
	deltas := util.ComputeTagDeltas(newTags, oldTags)
	if len(deltas) == 0 {
		return
	}
	if err := u.Store.BatchAdjustTagCounts("url_tag", deltas); err != nil {
		slog.Warn("failed to adjust url tag counts", "err", err)
	}
}

// cleanSiteAssets 删除站点关联的文件资源：icon、附件目录
func (u *URLService) cleanSiteAssets(site *model.Site) {
	persistDir := u.Store.PersistDir()
	assetsDir := filepath.Join(persistDir, "url-assets")

	if site.Icon != "" {
		iconPath := filepath.Join(assetsDir, "icons", site.Icon)
		if err := os.Remove(iconPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove site icon", "path", iconPath, "err", err)
		}
	}

	u.cleanEntityAttachments(assetsDir, site.ID)
}

// cleanEntityAttachments 删除 attachments/{entity_id}/ 整个目录
func (u *URLService) cleanEntityAttachments(assetsDir, entityID string) {
	dir := filepath.Join(assetsDir, "attachments", entityID)
	if err := os.RemoveAll(dir); err != nil {
		slog.Warn("failed to remove attachments dir", "path", dir, "err", err)
	}
}
