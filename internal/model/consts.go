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

// fileExtToMediaType 文件扩展名 → 媒体类型映射（包级私有，防止外部修改）
var fileExtToMediaType = map[string]string{
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

// LookupMediaType 根据文件扩展名查找媒体类型，未知扩展名返回 ("", false)
func LookupMediaType(ext string) (string, bool) {
	t, ok := fileExtToMediaType[ext]
	return t, ok
}

// IsSupportedMediaExt 判断文件扩展名是否为支持的媒体类型
func IsSupportedMediaExt(ext string) bool {
	_, ok := fileExtToMediaType[ext]
	return ok
}
