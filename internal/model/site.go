package model

import "time"

// Site 站点实体
type Site struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Domain        string       `json:"domain"`
	Icon          string       `json:"icon,omitempty"`
	Cover         string       `json:"cover,omitempty"`
	Description   string       `json:"description,omitempty"`
	Tags          []string     `json:"tags"`
	Attachments   []Attachment `json:"attachments"`
	BookmarkCount int          `json:"bookmark_count"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// CreateSiteReq 创建站点请求
type CreateSiteReq struct {
	Title       string   `json:"title"`
	Domain      string   `json:"domain"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateSiteReq 更新站点请求
type UpdateSiteReq struct {
	ID          string        `json:"id"`
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Tags        *[]string     `json:"tags,omitempty"`
	Icon        *string       `json:"icon,omitempty"`
	Cover       *string       `json:"cover,omitempty"`
	Attachments *[]Attachment `json:"attachments,omitempty"`
}

// SiteListReq 站点列表请求
type SiteListReq struct {
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Search   string   `json:"search,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

// SiteListResult 站点列表响应
type SiteListResult struct {
	Items   []Site `json:"items"`
	Total   int    `json:"total"`
	HasMore bool   `json:"has_more"`
}

// LookupSiteByURLReq 根据 URL 查询对应站点请求
type LookupSiteByURLReq struct {
	URL string `json:"url"`
}

// LookupSiteResult 站点查询结果
type LookupSiteResult struct {
	Found  bool   `json:"found"`
	SiteID string `json:"site_id,omitempty"`
	Domain string `json:"domain"`
}
