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

// URLService 站点 + 书签 + 临时队列的业务逻辑层。
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
		return nil, fmt.Errorf("站点标题不能为空")
	}
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("请提供完整的站点 URL（如 https://example.com）")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, fmt.Errorf("URL 格式不正确，请提供完整的 URL（如 https://example.com）")
	}
	req.Domain = domain

	tags, err := normalizeTags(req.Tags)
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
		return nil, fmt.Errorf("站点 ID 不能为空")
	}
	return u.Store.GetSite(id)
}

// UpdateSite 更新站点（URL/Domain 不可修改）。
// 当 Tags 字段变化时维护 url_tag 注册表 count。
func (u *URLService) UpdateSite(req model.UpdateSiteReq) (_ *model.Site, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("站点 ID 不能为空")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, fmt.Errorf("站点标题不能为空")
	}

	var oldTags []string
	if req.Tags != nil {
		tags, err := normalizeTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		req.Tags = &tags

		old, err := u.Store.GetSite(req.ID)
		if err != nil {
			return nil, err
		}
		oldTags = old.Tags
	}

	site, err := u.Store.UpdateSite(req)
	if err != nil {
		return nil, err
	}

	if req.Tags != nil {
		u.adjustURLTagCounts(site.Tags, oldTags)
	}
	return site, nil
}

// DeleteSite 删除站点。同步清理 icon、封面、附件目录，
// 并递减 url_tag 注册表 count。
func (u *URLService) DeleteSite(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("站点 ID 不能为空")
	}

	site, err := u.Store.GetSite(id)
	if err != nil {
		return err
	}

	if err := u.Store.DeleteSite(id); err != nil {
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
func (u *URLService) SearchURL(req model.SearchURLReq) (_ *model.SearchURLResult, err error) {
	defer logError(&err)
	return u.Store.SearchURL(req)
}

// ────────────────────── Bookmark ──────────────────────

// CreateBookmark 创建书签。校验 URL 合法性，自动提取域名匹配站点，
// 归一化标签并维护 url_tag 注册表 count。
// 按域名自动查找站点，未找到站点则存入临时队列而非创建书签。
func (u *URLService) CreateBookmark(req model.CreateBookmarkReq) (_ *model.Bookmark, err error) {
	defer logError(&err)
	if err := util.ValidateURL(req.URL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("书签标题不能为空")
	}

	domain, err := util.ExtractDomain(req.URL)
	if err != nil {
		return nil, err
	}
	site, err := u.Store.GetSiteByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("查询站点失败: %w", err)
	}
	if site == nil {
		return nil, fmt.Errorf("域名 %q 无对应站点，请先创建站点", domain)
	}
	req.SiteID = site.ID
	req.Domain = domain

	tags, err := normalizeTags(req.Tags)
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
		return nil, fmt.Errorf("书签 ID 不能为空")
	}
	return u.Store.GetBookmark(id)
}

// UpdateBookmark 更新书签。当 Tags 字段变化时维护 url_tag 注册表 count。
func (u *URLService) UpdateBookmark(req model.UpdateBookmarkReq) (_ *model.Bookmark, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("书签 ID 不能为空")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, fmt.Errorf("书签标题不能为空")
	}

	var oldTags []string
	if req.Tags != nil {
		tags, err := normalizeTags(*req.Tags)
		if err != nil {
			return nil, err
		}
		req.Tags = &tags

		old, err := u.Store.GetBookmark(req.ID)
		if err != nil {
			return nil, err
		}
		oldTags = old.Tags
	}

	bm, err := u.Store.UpdateBookmark(req)
	if err != nil {
		return nil, err
	}

	if req.Tags != nil {
		u.adjustURLTagCounts(bm.Tags, oldTags)
	}
	return bm, nil
}

// DeleteBookmark 删除单条书签。同步清理封面和附件文件，
// 并递减 url_tag 注册表 count。
func (u *URLService) DeleteBookmark(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("书签 ID 不能为空")
	}

	bm, err := u.Store.GetBookmark(id)
	if err != nil {
		return err
	}

	if err := u.Store.DeleteBookmark(id); err != nil {
		return err
	}

	u.adjustURLTagCounts(nil, bm.Tags)
	u.cleanBookmarkAssets(bm)
	return nil
}

// BatchDeleteBookmarks 批量删除同一站点下的书签。汇总所有被删除书签的标签后
// 一次性更新 url_tag 注册表 count，并逐个清理文件资源。
func (u *URLService) BatchDeleteBookmarks(siteID string, ids []string) (err error) {
	defer logError(&err)
	if len(ids) == 0 {
		return nil
	}

	bookmarks := make([]*model.Bookmark, 0, len(ids))
	for _, id := range ids {
		bm, err := u.Store.GetBookmark(id)
		if err != nil {
			slog.Warn("skip missing bookmark in batch delete", "id", id)
			continue
		}
		bookmarks = append(bookmarks, bm)
	}

	if err := u.Store.BatchDeleteBookmarks(siteID, ids); err != nil {
		return err
	}

	var allOldTags []string
	for _, bm := range bookmarks {
		allOldTags = append(allOldTags, bm.Tags...)
		u.cleanBookmarkAssets(bm)
	}
	u.adjustURLTagCounts(nil, allOldTags)
	return nil
}

// BatchTagBookmarks 批量为书签追加标签（不覆盖已有标签），
// 同步维护 url_tag 注册表 count。
func (u *URLService) BatchTagBookmarks(ids []string, tagsToAdd []string) (err error) {
	defer logError(&err)
	if len(ids) == 0 || len(tagsToAdd) == 0 {
		return nil
	}

	tags, err := normalizeTags(tagsToAdd)
	if err != nil {
		return err
	}

	deltas := make(map[string]int)
	for _, id := range ids {
		bm, err := u.Store.GetBookmark(id)
		if err != nil {
			slog.Warn("skip missing bookmark in batch tag", "id", id)
			continue
		}

		merged, added := mergeTags(bm.Tags, tags)
		if added == 0 {
			continue
		}

		for _, t := range tags {
			if util.StringIndex(bm.Tags, t) < 0 {
				deltas[t]++
			}
		}

		newTags := merged
		if _, err := u.Store.UpdateBookmark(model.UpdateBookmarkReq{
			ID:   id,
			Tags: &newTags,
		}); err != nil {
			slog.Warn("failed to update bookmark tags", "id", id, "err", err)
		}
	}

	if len(deltas) > 0 {
		if err := u.Store.BatchAdjustTagCounts("url_tag:", deltas); err != nil {
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

// mergeTags 将 toAdd 追加到 existing 中（去重），返回合并后的切片和实际追加数量
func mergeTags(existing, toAdd []string) ([]string, int) {
	set := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		set[t] = struct{}{}
	}

	merged := make([]string, len(existing))
	copy(merged, existing)
	added := 0
	for _, t := range toAdd {
		if _, ok := set[t]; !ok {
			merged = append(merged, t)
			set[t] = struct{}{}
			added++
		}
	}
	return merged, added
}

// ────────────────────── Queue ──────────────────────

// AddToQueue 将 URL 加入临时队列。
// TODO: 入队去重（归一化 URL 后校验队列 + 已有书签，重复则拒绝）
func (u *URLService) AddToQueue(rawURL string) (_ *model.QueueItem, err error) {
	defer logError(&err)
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}
	if err := util.ValidateURL(rawURL); err != nil {
		return nil, err
	}
	return u.Store.AddToQueue(rawURL)
}

// ListQueue 获取临时队列中所有条目（按 added_at 降序）
func (u *URLService) ListQueue() (_ []model.QueueItem, err error) {
	defer logError(&err)
	return u.Store.ListQueue()
}

// DeleteQueueItem 删除临时队列中的指定条目
func (u *URLService) DeleteQueueItem(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("队列条目 ID 不能为空")
	}
	return u.Store.DeleteQueueItem(id)
}

// ClearQueue 清空临时队列
func (u *URLService) ClearQueue() (err error) {
	defer logError(&err)
	return u.Store.ClearQueue()
}

// ────────────────────── Normalize ──────────────────────

// NormalizeURL 归一化 URL（去协议、去 www、去尾部斜杠等），
// 供前端输入时实时预览标准化后的地址。
func (u *URLService) NormalizeURL(rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", nil
	}
	return util.NormalizeURL(rawURL)
}

// ────────────────────── Lookup ──────────────────────

// LookupSiteByURL 根据 URL 查询对应站点。
// 提取 URL 中的域名，查找是否有对应站点。
// 在新增url前判断用户是否需要先创建site
func (u *URLService) LookupSiteByURL(req model.LookupSiteByURLReq) (_ *model.LookupSiteResult, err error) {
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

	site, err := u.Store.GetSiteByDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("查询站点失败: %w", err)
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

// normalizeTags 校验并归一化标签列表，去重后返回
func normalizeTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return tags, nil
	}
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		if err := util.ValidateTagName(t); err != nil {
			return nil, fmt.Errorf("标签 %q: %w", t, err)
		}
		normalized := util.NormalizeTagName(t)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result, nil
}

// adjustURLTagCounts 计算新旧标签的差值，批量更新 url_tag 注册表 count。
// newTags 为新标签列表（创建/更新后），oldTags 为旧标签列表（更新/删除前）。
func (u *URLService) adjustURLTagCounts(newTags, oldTags []string) {
	deltas := make(map[string]int)
	for _, t := range newTags {
		deltas[t]++
	}
	for _, t := range oldTags {
		deltas[t]--
	}

	nonZero := make(map[string]int)
	for k, v := range deltas {
		if v != 0 {
			nonZero[k] = v
		}
	}
	if len(nonZero) == 0 {
		return
	}

	if err := u.Store.BatchAdjustTagCounts("url_tag:", nonZero); err != nil {
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
