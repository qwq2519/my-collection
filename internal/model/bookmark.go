package model

import "time"

// Bookmark 书签实体
type Bookmark struct {
	ID          string       `json:"id"`
	URL         string       `json:"url"`
	Domain      string       `json:"domain"`
	SiteID      string       `json:"site_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Tags        []string     `json:"tags"`
	Attachments []Attachment `json:"attachments"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// EnsureSlices 确保切片字段非 nil，避免 JSON 序列化为 null。
// Store 层写入前调用。
func (b *Bookmark) EnsureSlices() {
	if b.Tags == nil {
		b.Tags = []string{}
	}
	if b.Attachments == nil {
		b.Attachments = []Attachment{}
	}
}

// CreateBookmarkReq 创建书签请求。
// SiteID 由后端从 URL 自动匹配，前端无需传递。
type CreateBookmarkReq struct {
	URL         string   `json:"url"`
	SiteID      string   `json:"-"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateBookmarkReq 更新书签请求
type UpdateBookmarkReq struct {
	ID          string        `json:"id"`
	Title       *string       `json:"title,omitempty"`
	Description *string       `json:"description,omitempty"`
	Tags        *[]string     `json:"tags,omitempty"`
	Attachments *[]Attachment `json:"attachments,omitempty"`
}

// BookmarkListReq 书签列表请求（按站点加载书签）
type BookmarkListReq struct {
	SiteID   string `json:"site_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// BookmarkListResult 书签列表响应
type BookmarkListResult struct {
	Items   []Bookmark `json:"items"`
	Total   int        `json:"total"`
	HasMore bool       `json:"has_more"`
}
