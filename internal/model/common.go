package model

import "time"

// ensureStringSlice 确保字符串切片非 nil，避免 JSON 序列化为 null
func ensureStringSlice(s *[]string) {
	if *s == nil {
		*s = []string{}
	}
}

// Attachment 附件信息（站点/书签共用）
type Attachment struct {
	Filename   string    `json:"filename"`
	Label      string    `json:"label,omitempty"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// UploadFileReq 统一文件上传请求（笔记图片、书签图标等小文件）。
// Data 在 Wails 绑定层以 base64 JSON 编码，约 33% 膨胀，
// 对当前场景（通常 <1MB）可接受。
// 媒体文件不走上传，由后端扫描文件系统获取。
type UploadFileReq struct {
	Scene    string `json:"scene"`
	EntityID string `json:"entity_id"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

// UploadFileResult 上传结果，返回存储后的相对路径
type UploadFileResult struct {
	Path string `json:"path"`
}

// DirtyItem 索引脏队列条目（Bleve 写入失败时记录）
type DirtyItem struct {
	DocID    string    `json:"doc_id"`
	DocType  string    `json:"doc_type"`
	FailedAt time.Time `json:"failed_at"`
}

// DirtyIndexStatus 索引状态摘要（返回给前端）
type DirtyIndexStatus struct {
	HasDirty   bool `json:"has_dirty"`
	URLCount   int  `json:"url_count"`
	NoteCount  int  `json:"note_count"`
	MediaCount int  `json:"media_count"`
}

// ScanProgress 媒体扫描进度事件载荷
type ScanProgress struct {
	FolderID string `json:"folder_id"`
	Scanned  int    `json:"scanned"`
	Total    int    `json:"total"`
}

// ScanComplete 媒体扫描完成事件载荷
type ScanComplete struct {
	FolderID string `json:"folder_id"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
	Modified int    `json:"modified"`
}

// FFmpegStatus ffmpeg 检测结果。
// Available 为 true 时 BinPath/Version 有值；
// 为 false 时 Guide 包含平台对应的安装指令供前端展示。
type FFmpegStatus struct {
	Available bool             `json:"available"`
	Version   string           `json:"version,omitempty"`
	BinPath   string           `json:"bin_path,omitempty"`
	Guide     []InstallCommand `json:"guide,omitempty"`
}

// InstallCommand 一条安装指令（标题 + 可复制的命令）
type InstallCommand struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

// AppSettings 应用设置摘要（设置页进入时返回）
type AppSettings struct {
	PersistDir  string           `json:"persist_dir"`
	FFmpeg      FFmpegStatus     `json:"ffmpeg"`
	IndexStatus DirtyIndexStatus `json:"index_status"`
}

// FetchMetaReq 元数据抓取请求
type FetchMetaReq struct {
	URL string `json:"url"`
}

// FetchMetaResult 元数据抓取结果。
// Icon 为已保存的文件名（如 "github.com.png"），为空表示未获取到。
// OGImage 为远程图片 URL，由前端决定是否展示或下载。
type FetchMetaResult struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	OGImage     string `json:"og_image,omitempty"`
}
