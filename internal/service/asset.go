package service

import (
	"net/http"
	"os"
	"strings"

	"collections/internal/util"
)

// NewAssetHandler 创建组合式资源处理器：
//   - /persist/* 路径映射到本地 persistDir（白名单 + 目录遍历防护 + 缓存头）
//   - 其余路径委托给 embedded handler（前端打包产物）
func NewAssetHandler(embedded http.Handler, persistDir string) http.Handler {
	return &assetHandler{
		embedded:   embedded,
		persistDir: persistDir,
	}
}

type assetHandler struct {
	embedded   http.Handler
	persistDir string
}

func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/persist/") {
		h.servePersist(w, r)
		return
	}
	h.embedded.ServeHTTP(w, r)
}

var allowedPersistPrefixes = []string{
	"url-assets/",
	"note-images/",
	"media-folders/",
}

func (h *assetHandler) servePersist(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/persist/")

	allowed := false
	for _, prefix := range allowedPersistPrefixes {
		if strings.HasPrefix(relPath, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}

	absPath, err := util.SafePath(h.persistDir, relPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, absPath)
}
