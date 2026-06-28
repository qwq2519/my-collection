package service

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"collections/internal/util"
)

// NewAssetHandler 创建组合式资源处理器：
//   - /persist/* 路径映射到本地 persistDir（白名单 + 目录遍历防护 + 缓存头）
//   - 其余路径委托给 embedded handler（前端打包产物）
//
// persistDir 在构造时转为绝对路径，消除运行时对工作目录的依赖。
func NewAssetHandler(embedded http.Handler, persistDir string) http.Handler {
	abs, err := filepath.Abs(persistDir)
	if err != nil {
		log.Fatalf("resolve persist dir: %v", err)
	}
	return &assetHandler{
		embedded:   embedded,
		persistDir: abs,
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

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

	http.ServeFile(&cacheWriter{ResponseWriter: w}, r, absPath)
}

// cacheWriter 拦截 WriteHeader，仅在 2xx 响应时设置 Cache-Control，
// 避免 404 等错误响应也被浏览器缓存。
type cacheWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (cw *cacheWriter) WriteHeader(code int) {
	if !cw.wroteHeader {
		cw.wroteHeader = true
		if code >= 200 && code < 300 {
			cw.ResponseWriter.Header().Set("Cache-Control", "public, max-age=86400")
		}
	}
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *cacheWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}
	return cw.ResponseWriter.Write(b)
}
