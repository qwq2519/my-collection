package model

import "time"

// Bookmark 书签实体
type Bookmark struct {
	ID          string       `json:"id"`
	URL         string       `json:"url"`
	Domain      string       `json:"domain"`
	SiteID      string       `json:"site_id"`
	Title       string       `json:"title"`
	Cover       string       `json:"cover,omitempty"`
	Description string       `json:"description,omitempty"`
	Tags        []string     `json:"tags"`
	Attachments []Attachment `json:"attachments"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// CreateBookmarkReq 创建书签请求
type CreateBookmarkReq struct {
	URL         string   `json:"url"`
	SiteID      string   `json:"site_id"`
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
	Cover       *string       `json:"cover,omitempty"`
	Attachments *[]Attachment `json:"attachments,omitempty"`
}

// BookmarkListReq 书签列表请求
type BookmarkListReq struct {
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Search   string   `json:"search,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	SiteID   string   `json:"site_id,omitempty"`
}

// BookmarkListResult 书签列表响应
type BookmarkListResult struct {
	Items   []Bookmark `json:"items"`
	Total   int        `json:"total"`
	HasMore bool       `json:"has_more"`
}
