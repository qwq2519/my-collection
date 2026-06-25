package model

import "time"

// Note 笔记实体
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateNoteReq 创建笔记请求
type CreateNoteReq struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// UpdateNoteReq 更新笔记请求
type UpdateNoteReq struct {
	ID    string  `json:"id"`
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
}

// NoteListReq 笔记列表请求
type NoteListReq struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Search   string `json:"search,omitempty"`
}

// NoteListResult 笔记列表响应
type NoteListResult struct {
	Items   []Note `json:"items"`
	Total   int    `json:"total"`
	HasMore bool   `json:"has_more"`
}
