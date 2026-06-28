package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"github.com/blevesearch/bleve/v2"
	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

func noteBleveFields(note *model.Note) map[string]interface{} {
	return map[string]interface{}{
		"_type":      "note",
		"title":      note.Title,
		"body":       util.StripMarkdown(note.Body),
		"updated_at": note.UpdatedAt,
	}
}

// CreateNote 创建笔记，写入 BuntDB 后同步索引到 Bleve
func (s *Store) CreateNote(req model.CreateNoteReq) (*model.Note, error) {
	now := time.Now()
	note := &model.Note{
		ID:        uuid.New().String(),
		Title:     req.Title,
		Body:      req.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}

	val, err := json.Marshal(note)
	if err != nil {
		return nil, fmt.Errorf("marshal note: %w", err)
	}

	err = s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("note:"+note.ID, string(val), nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("note:"+note.ID, noteBleveFields(note))
	slog.Info("note created", "id", note.ID, "title", note.Title)
	return note, nil
}

// GetNote 按 ID 查询笔记
func (s *Store) GetNote(id string) (*model.Note, error) {
	var note model.Note
	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get("note:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("笔记不存在")
		}
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &note)
	})
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// UpdateNote 部分更新笔记，body 经 StripMarkdown 后重建索引
func (s *Store) UpdateNote(req model.UpdateNoteReq) (*model.Note, error) {
	var note model.Note

	err := s.db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get("note:" + req.ID)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("笔记不存在")
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(val), &note); err != nil {
			return fmt.Errorf("unmarshal note: %w", err)
		}

		if req.Title != nil {
			note.Title = *req.Title
		}
		if req.Body != nil {
			note.Body = *req.Body
		}

		note.UpdatedAt = time.Now()

		newVal, err := json.Marshal(&note)
		if err != nil {
			return fmt.Errorf("marshal note: %w", err)
		}
		_, _, err = tx.Set("note:"+note.ID, string(newVal), nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.IndexDoc("note:"+note.ID, noteBleveFields(&note))
	return &note, nil
}

// DeleteNote 删除笔记（关联图片清理由 service 层负责）
func (s *Store) DeleteNote(id string) error {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete("note:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("笔记不存在")
		}
		return err
	})
	if err != nil {
		return err
	}

	s.DeleteDoc("note:"+id, "note")
	slog.Info("note deleted", "id", id)
	return nil
}

// ListNotes 分页查询笔记列表。无搜索时走 BuntDB，有搜索时走 Bleve（title + body 全文）。
func (s *Store) ListNotes(req model.NoteListReq) (*model.NoteListResult, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	if req.Search == "" {
		return s.listNotesFromDB(req)
	}
	return s.listNotesFromBleve(req)
}

func (s *Store) listNotesFromDB(req model.NoteListReq) (*model.NoteListResult, error) {
	skip := (req.Page - 1) * req.PageSize
	total := 0
	notes := make([]model.Note, 0, req.PageSize)

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend("idx:note_updated", func(key, value string) bool {
			total++
			if total <= skip {
				return true
			}
			if len(notes) >= req.PageSize {
				return true
			}
			var note model.Note
			if err := json.Unmarshal([]byte(value), &note); err != nil {
				slog.Warn("skip corrupted note", "key", key, "err", err)
				return true
			}
			notes = append(notes, note)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	return &model.NoteListResult{
		Items:   notes,
		Total:   total,
		HasMore: skip+len(notes) < total,
	}, nil
}

func (s *Store) listNotesFromBleve(req model.NoteListReq) (*model.NoteListResult, error) {
	typeQ := bleve.NewTermQuery("note")
	typeQ.SetField("_type")

	titleQ := bleve.NewMatchQuery(req.Search)
	titleQ.SetField("title")
	bodyQ := bleve.NewMatchQuery(req.Search)
	bodyQ.SetField("body")

	conjunction := bleve.NewConjunctionQuery(typeQ, bleve.NewDisjunctionQuery(titleQ, bodyQ))

	searchReq := bleve.NewSearchRequest(conjunction)
	searchReq.SortBy([]string{"-updated_at"})
	searchReq.From = (req.Page - 1) * req.PageSize
	searchReq.Size = req.PageSize
	searchReq.Fields = []string{}

	result, err := s.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}

	notes := make([]model.Note, 0, len(result.Hits))
	err = s.db.View(func(tx *buntdb.Tx) error {
		for _, hit := range result.Hits {
			val, err := tx.Get(hit.ID)
			if err != nil {
				slog.Warn("note in index but not in db", "id", hit.ID, "err", err)
				continue
			}
			var note model.Note
			if err := json.Unmarshal([]byte(val), &note); err != nil {
				slog.Warn("corrupted note data", "id", hit.ID, "err", err)
				continue
			}
			notes = append(notes, note)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("fetch notes: %w", err)
	}

	total := int(result.Total)
	return &model.NoteListResult{
		Items:   notes,
		Total:   total,
		HasMore: (req.Page-1)*req.PageSize+len(notes) < total,
	}, nil
}
