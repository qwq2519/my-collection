package model

// Attachment 附件信息（站点/书签共用）
type Attachment struct {
	Filename string `json:"filename"`
	Label    string `json:"label,omitempty"`
}

// UploadFileReq 统一文件上传请求
type UploadFileReq struct {
	Scene    string `json:"scene"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

// UploadFileResult 上传结果，返回存储后的相对路径
type UploadFileResult struct {
	Path string `json:"path"`
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
