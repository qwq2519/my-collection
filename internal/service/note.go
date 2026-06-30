package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"collections/internal/model"
	"collections/internal/store"
	"collections/internal/util"
)

// NoteService 笔记业务逻辑层。
// 公开方法即前端可调用接口（通过 Wails 绑定）。
type NoteService struct {
	Store *store.Store
}

// CreateNote 创建笔记，title 必填
func (n *NoteService) CreateNote(req model.CreateNoteReq) (_ *model.Note, err error) {
	defer logError(&err)
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("note title required")
	}
	return n.Store.CreateNote(req)
}

// GetNote 按 ID 查询笔记详情
func (n *NoteService) GetNote(id string) (_ *model.Note, err error) {
	defer logError(&err)
	if id == "" {
		return nil, fmt.Errorf("note ID required")
	}
	return n.Store.GetNote(id)
}

// UpdateNote 部分更新笔记。Title 若提供则不允许为空
func (n *NoteService) UpdateNote(req model.UpdateNoteReq) (_ *model.Note, err error) {
	defer logError(&err)
	if req.ID == "" {
		return nil, fmt.Errorf("note ID required")
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, fmt.Errorf("note title required")
	}
	return n.Store.UpdateNote(req)
}

// DeleteNote 删除笔记并清理关联图片目录
func (n *NoteService) DeleteNote(id string) (err error) {
	defer logError(&err)
	if id == "" {
		return fmt.Errorf("note ID required")
	}

	if err := n.Store.DeleteNote(id); err != nil {
		return err
	}

	n.cleanNoteImages(id)
	return nil
}

// ListNotes 分页查询笔记列表（有 Search 时走 Bleve 全文搜索）
func (n *NoteService) ListNotes(req model.NoteListReq) (_ *model.NoteListResult, err error) {
	defer logError(&err)
	return n.Store.ListNotes(req)
}

// ────────────────────── Orphan Images ──────────────────────

// noteImageRefRe 匹配 Markdown body 中的笔记图片引用路径。
// 格式：note-images/{note_id}/{filename}
var noteImageRefRe = regexp.MustCompile(`note-images/[^/]+/([^\s)]+)`)

// DetectOrphanImages 对比 body 中的图片引用与磁盘文件，返回孤儿文件列表
func (n *NoteService) DetectOrphanImages(req model.DetectOrphanImagesReq) (_ *model.DetectOrphanImagesResult, err error) {
	defer logError(&err)
	if req.NoteID == "" {
		return nil, fmt.Errorf("note ID required")
	}

	dir := filepath.Join(n.Store.PersistDir(), "note-images", req.NoteID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.DetectOrphanImagesResult{OrphanFiles: []string{}}, nil
		}
		return nil, fmt.Errorf("read note images dir: %w", err)
	}

	referenced := make(map[string]struct{})
	for _, m := range noteImageRefRe.FindAllStringSubmatch(req.Body, -1) {
		referenced[m[1]] = struct{}{}
	}

	var orphans []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if _, ok := referenced[e.Name()]; !ok {
			orphans = append(orphans, e.Name())
		}
	}

	return &model.DetectOrphanImagesResult{OrphanFiles: orphans}, nil
}

// DeleteOrphanImages 删除指定的孤儿图片文件
func (n *NoteService) DeleteOrphanImages(req model.DeleteOrphanImagesReq) (err error) {
	defer logError(&err)
	if req.NoteID == "" {
		return fmt.Errorf("note ID required")
	}
	if len(req.Files) == 0 {
		return nil
	}

	baseDir := filepath.Join(n.Store.PersistDir(), "note-images", req.NoteID)
	for _, f := range req.Files {
		safePath, err := util.SafePath(baseDir, f)
		if err != nil {
			slog.Warn("skip unsafe orphan path", "file", f, "err", err)
			continue
		}
		if err := os.Remove(safePath); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove orphan image", "path", safePath, "err", err)
		}
	}
	return nil
}

// cleanNoteImages 删除 persist/note-images/{note_id}/ 整个目录
func (n *NoteService) cleanNoteImages(noteID string) {
	dir := filepath.Join(n.Store.PersistDir(), "note-images", noteID)
	if err := os.RemoveAll(dir); err != nil {
		slog.Warn("failed to remove note images dir", "path", dir, "err", err)
	}
}
