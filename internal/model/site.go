package model

import "time"

// Site 站点实体
type Site struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	URL           string       `json:"url"`
	Domain        string       `json:"domain"`
	Icon          string       `json:"icon,omitempty"`
	Description   string       `json:"description,omitempty"`
	Tags          []string     `json:"tags"`
	Attachments   []Attachment `json:"attachments"`
	BookmarkCount int          `json:"bookmark_count"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// CreateSiteReq 创建站点请求。
// URL 必填，Domain 由后端从 URL 自动提取，前端无需传递。
type CreateSiteReq struct {
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	Domain      string       `json:"domain,omitempty"`
	Description string       `json:"description,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	Icon        string       `json:"icon,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// UpdateSiteReq 更新站点请求。
// 传入 URL 时后端自动重新提取 Domain，前端无需单独传 Domain。
type UpdateSiteReq struct {
	ID          string        `json:"id"`
	Title       *string       `json:"title,omitempty"`
	URL         *string       `json:"url,omitempty"`
	Domain      *string       `json:"domain,omitempty"`
	Description *string       `json:"description,omitempty"`
	Tags        *[]string     `json:"tags,omitempty"`
	Icon        *string       `json:"icon,omitempty"`
	Attachments *[]Attachment `json:"attachments,omitempty"`
}

// SiteListReq 站点列表请求（纯列表，不含搜索；搜索走 SearchURL）
type SiteListReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// SiteListResult 站点列表响应
type SiteListResult struct {
	Items   []Site `json:"items"`
	Total   int    `json:"total"`
	HasMore bool   `json:"has_more"`
}

// SearchURLReq 统一搜索请求（同时搜索站点和书签，按站点分组返回）
type SearchURLReq struct {
	Search   string   `json:"search,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// SiteWithBookmarks 搜索结果中的站点项（含命中的书签）
type SiteWithBookmarks struct {
	Site      Site       `json:"site"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

// SearchURLResult 统一搜索结果
type SearchURLResult struct {
	Items   []SiteWithBookmarks `json:"items"`
	Total   int                 `json:"total"`
	HasMore bool                `json:"has_more"`
}

// EnsureSlices 确保切片字段非 nil，避免 JSON 序列化为 null。
// Store 层写入前调用。
func (s *Site) EnsureSlices() {
	if s.Tags == nil {
		s.Tags = []string{}
	}
	if s.Attachments == nil {
		s.Attachments = []Attachment{}
	}
}

// LookupSiteByURLReq 根据 URL 查询对应站点请求
type LookupSiteByURLReq struct {
	URL string `json:"url"`
}

// LookupSiteResult 站点查询结果
type LookupSiteResult struct {
	Found  bool   `json:"found"`
	ID     string `json:"site_id,omitempty"`
	URL    string `json:"url,omitempty"`
	Domain string `json:"domain"`
}
