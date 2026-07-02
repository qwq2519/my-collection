package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"collections/internal/model"

	"github.com/tidwall/buntdb"
)

// reindexURLEntities 重建站点和书签的 Bleve 索引（tag 变更事务提交后调用）
func (s *Store) reindexURLEntities(sites []model.Site, bms []model.Bookmark) {
	for i := range sites {
		s.IndexDoc("site:"+sites[i].ID, siteBleveFields(&sites[i]))
	}
	for i := range bms {
		s.IndexDoc("bm:"+bms[i].ID, bmBleveFields(&bms[i]))
	}
}

// --- URL 标签编排事务（跨实体联动：遍历 site + bookmark） ---

// tagMutator 接收当前 tags 列表，返回修改后的 tags 和是否有变化。
type tagMutator func(tags []string) (newTags []string, changed bool)

type entityKV struct {
	key, value string
}

// mutateURLEntityTags 遍历所有站点和书签，对每个实体的 tags 应用 mutator。
// 先收集所有 KV 快照，再逐条修改写回（避免在 AscendKeys 遍历中修改导致不一致）。
// 变化的实体在事务内写回 BuntDB，并收集到返回值供事务后重建 Bleve 索引。
func mutateURLEntityTags(tx *buntdb.Tx, mutate tagMutator) (sites []model.Site, bms []model.Bookmark) {
	var siteKVs, bmKVs []entityKV
	tx.AscendKeys("site:*", func(key, value string) bool {
		siteKVs = append(siteKVs, entityKV{key, value})
		return true
	})
	tx.AscendKeys("bm:*", func(key, value string) bool {
		bmKVs = append(bmKVs, entityKV{key, value})
		return true
	})

	for _, kv := range siteKVs {
		var site model.Site
		if json.Unmarshal([]byte(kv.value), &site) != nil {
			continue
		}
		newTags, changed := mutate(site.Tags)
		if !changed {
			continue
		}
		site.Tags = newTags
		site.EnsureSlices()
		v, _ := json.Marshal(&site)
		tx.Set(kv.key, string(v), nil)
		sites = append(sites, site)
	}

	for _, kv := range bmKVs {
		var bm model.Bookmark
		if json.Unmarshal([]byte(kv.value), &bm) != nil {
			continue
		}
		newTags, changed := mutate(bm.Tags)
		if !changed {
			continue
		}
		bm.Tags = newTags
		bm.EnsureSlices()
		v, _ := json.Marshal(&bm)
		tx.Set(kv.key, string(v), nil)
		bms = append(bms, bm)
	}

	return
}

// RenameURLTag 重命名 URL 标签：更新所有站点和书签的 tags 数组 + 注册表。
// 单事务内完成 BuntDB 写入，提交后逐条重建受影响实体的 Bleve 索引。
// 返回受影响的实体数量。
func (s *Store) RenameURLTag(req model.RenameTagReq) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark

	err := s.db.Update(func(tx *buntdb.Tx) error {
		if _, err := tx.Get("url_tag:" + req.NewName); err == nil {
			return fmt.Errorf("tag %q already exists, use merge instead", req.NewName)
		}
		oldVal, err := tx.Get("url_tag:" + req.OldName)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("tag %q not found", req.OldName)
		}
		if err != nil {
			return err
		}

		sitesReindex, bmsReindex = mutateURLEntityTags(tx, func(tags []string) ([]string, bool) {
			idx := slices.Index(tags, req.OldName)
			if idx < 0 {
				return tags, false
			}
			tags[idx] = req.NewName
			return tags, true
		})

		var tag model.Tag
		json.Unmarshal([]byte(oldVal), &tag)
		tag.Name = req.NewName
		v, _ := json.Marshal(&tag)
		tx.Delete("url_tag:" + req.OldName)
		tx.Set("url_tag:"+req.NewName, string(v), nil)
		return nil
	})
	if err != nil {
		return 0, err
	}

	s.reindexURLEntities(sitesReindex, bmsReindex)

	affected := len(sitesReindex) + len(bmsReindex)
	slog.Info("url tag renamed", "old", req.OldName, "new", req.NewName, "affected", affected)
	return affected, nil
}

// MergeURLTag 将 source 标签合并到 target：实体已有 target 则仅移除 source，
// 否则替换 source 为 target。删除 source 注册表，重算 target count。
// 单事务内完成 BuntDB 写入，提交后逐条重建受影响实体的 Bleve 索引。
func (s *Store) MergeURLTag(req model.MergeTagReq) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark
	targetCount := 0

	err := s.db.Update(func(tx *buntdb.Tx) error {
		if _, err := tx.Get("url_tag:" + req.Source); err == buntdb.ErrNotFound {
			return fmt.Errorf("source tag %q not found", req.Source)
		}

		sitesReindex, bmsReindex = mutateURLEntityTags(tx, func(tags []string) ([]string, bool) {
			hasSource := slices.Contains(tags, req.Source)
			hasTarget := slices.Contains(tags, req.Target)

			if !hasSource {
				if hasTarget {
					targetCount++
				}
				return tags, false
			}

			if hasTarget {
				tags = slices.DeleteFunc(tags, func(s string) bool { return s == req.Source })
			} else {
				tags[slices.Index(tags, req.Source)] = req.Target
			}
			targetCount++
			return tags, true
		})

		tx.Delete("url_tag:" + req.Source)

		targetTag := model.Tag{Name: req.Target, Count: targetCount, CreatedAt: time.Now()}
		if tv, err := tx.Get("url_tag:" + req.Target); err == nil {
			var existing model.Tag
			json.Unmarshal([]byte(tv), &existing)
			targetTag.CreatedAt = existing.CreatedAt
		}
		v, _ := json.Marshal(&targetTag)
		tx.Set("url_tag:"+req.Target, string(v), nil)

		return nil
	})
	if err != nil {
		return 0, err
	}

	s.reindexURLEntities(sitesReindex, bmsReindex)

	affected := len(sitesReindex) + len(bmsReindex)
	slog.Info("url tag merged", "source", req.Source, "target", req.Target, "affected", affected)
	return affected, nil
}

// DeleteURLTagFromEntities 从所有站点和书签中移除指定标签并删除注册表条目。
// 单事务内完成 BuntDB 写入，提交后逐条重建受影响实体的 Bleve 索引。
// 返回受影响的实体数量。
func (s *Store) DeleteURLTagFromEntities(name string) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark

	err := s.db.Update(func(tx *buntdb.Tx) error {
		sitesReindex, bmsReindex = mutateURLEntityTags(tx, func(tags []string) ([]string, bool) {
			if !slices.Contains(tags, name) {
				return tags, false
			}
			return slices.DeleteFunc(tags, func(s string) bool { return s == name }), true
		})

		tx.Delete("url_tag:" + name)
		return nil
	})
	if err != nil {
		return 0, err
	}

	s.reindexURLEntities(sitesReindex, bmsReindex)

	affected := len(sitesReindex) + len(bmsReindex)
	slog.Info("url tag deleted from entities", "name", name, "affected", affected)
	return affected, nil
}

// RecountURLTags 重新统计所有 URL 标签的 count（遍历站点 + 书签）
func (s *Store) RecountURLTags() error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		counts := make(map[string]int)

		tx.AscendKeys("site:*", func(key, value string) bool {
			var site model.Site
			if json.Unmarshal([]byte(value), &site) != nil {
				return true
			}
			for _, t := range site.Tags {
				counts[t]++
			}
			return true
		})
		tx.AscendKeys("bm:*", func(key, value string) bool {
			var bm model.Bookmark
			if json.Unmarshal([]byte(value), &bm) != nil {
				return true
			}
			for _, t := range bm.Tags {
				counts[t]++
			}
			return true
		})

		tx.AscendKeys("url_tag:*", func(key, value string) bool {
			var tag model.Tag
			if json.Unmarshal([]byte(value), &tag) != nil {
				return true
			}
			tag.Count = counts[tag.Name]
			delete(counts, tag.Name)
			v, _ := json.Marshal(&tag)
			tx.Set(key, string(v), nil)
			return true
		})

		for name, count := range counts {
			tag := model.Tag{Name: name, Count: count, CreatedAt: time.Now()}
			v, _ := json.Marshal(&tag)
			tx.Set("url_tag:"+name, string(v), nil)
		}

		slog.Info("url tags recounted")
		return nil
	})
}
