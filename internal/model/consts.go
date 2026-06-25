package model

// 书签状态
const (
	BookmarkStatusAlive = "alive"
	BookmarkStatusDead  = "dead"
)

// 媒体文件类型（media_type 字段）
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
)

// Merkle Tree 节点类型
const (
	TreeNodeFile = "file"
	TreeNodeDir  = "dir"
)

// FileExtToMediaType 文件扩展名 → 媒体类型映射
var FileExtToMediaType = map[string]string{
	".jpg":  MediaTypeImage,
	".jpeg": MediaTypeImage,
	".png":  MediaTypeImage,
	".gif":  MediaTypeImage,
	".bmp":  MediaTypeImage,
	".webp": MediaTypeImage,
	".avif": MediaTypeImage,
	".svg":  MediaTypeImage,
	".mp4":  MediaTypeVideo,
	".mkv":  MediaTypeVideo,
	".avi":  MediaTypeVideo,
	".mov":  MediaTypeVideo,
	".wmv":  MediaTypeVideo,
	".flv":  MediaTypeVideo,
	".webm": MediaTypeVideo,
	".mp3":  MediaTypeAudio,
	".flac": MediaTypeAudio,
	".wav":  MediaTypeAudio,
	".aac":  MediaTypeAudio,
	".ogg":  MediaTypeAudio,
	".m4a":  MediaTypeAudio,
}
