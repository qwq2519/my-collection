package model

import "time"

// Tag 标签注册表条目（url_tag 和 media_tag 共用结构）
type Tag struct {
	Name      string    `json:"name"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
}

// TagListResult 标签列表响应（全量返回，无分页）
type TagListResult struct {
	Items []Tag `json:"items"`
	Total int   `json:"total"`
}

// RenameTagReq 重命名标签请求
type RenameTagReq struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// MergeTagReq 合并标签请求（将 Source 合并到 Target，Source 消失）
type MergeTagReq struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
