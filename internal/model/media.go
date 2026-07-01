package model

import "time"

// MediaFolder 媒体文件夹注册表（存储在 BuntDB）
type MediaFolder struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	FileCount  int       `json:"file_count"`
	AddedAt    time.Time `json:"added_at"`
	LastScanAt time.Time `json:"last_scan_at,omitempty"`
}

// MediaFile 单个媒体文件的元数据（存储在 media_meta.json 的 files map 中）
type MediaFile struct {
	MediaType   string    `json:"media_type"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description,omitempty"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	Preview     string    `json:"preview,omitempty"`
	ScannedAt   time.Time `json:"scanned_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FileSize    int64     `json:"file_size"`
	Width       *int      `json:"width,omitempty"`
	Height      *int      `json:"height,omitempty"`
	Duration    *float64  `json:"duration,omitempty"`
}

// MediaMeta media_meta.json 的完整结构
type MediaMeta struct {
	SchemaVersion int                  `json:"schema_version"`
	FolderID      string               `json:"folder_id"`
	Files         map[string]MediaFile `json:"files"`
}

// TreeNode Merkle Tree 中的节点（文件或目录）。
// Mtime 含义由 Type 决定：文件为文件修改时间，目录为目录修改时间（用于剪枝）。
type TreeNode struct {
	Type     string               `json:"type"`
	Hash     string               `json:"hash"`
	Mtime    *time.Time           `json:"mtime,omitempty"`
	Size     *int64               `json:"size,omitempty"`
	Children map[string]*TreeNode `json:"children,omitempty"`
}

// TreeHashFile tree_hash.json 的完整结构
type TreeHashFile struct {
	SchemaVersion int       `json:"schema_version"`
	FolderID      string    `json:"folder_id"`
	Root          *TreeNode `json:"root"`
}

// MediaListReq 媒体文件列表请求
type MediaListReq struct {
	FolderID  string   `json:"folder_id"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
	Search    string   `json:"search,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	MediaType string   `json:"media_type,omitempty"`
}

// MediaListResult 媒体文件列表响应
type MediaListResult struct {
	Items   []MediaFileItem `json:"items"`
	Total   int             `json:"total"`
	HasMore bool            `json:"has_more"`
}

// MediaFileItem 媒体文件列表项（内嵌 MediaFile，附加文件夹定位信息）
type MediaFileItem struct {
	MediaFile
	FolderID string `json:"folder_id"`
	RelPath  string `json:"rel_path"`
}
