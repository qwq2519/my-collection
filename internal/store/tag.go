package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"
	"collections/internal/util"

	"github.com/tidwall/buntdb"
)

// --- 标签注册表通用操作（url_tag / media_tag 共用） ---

// GetTag 按名称查询标签。未找到返回 (nil, nil)。
func (s *Store) GetTag(prefix, name string) (*model.Tag, error) {
	var tag model.Tag
	err := s.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(prefix + ":" + name)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &tag)
	})
	if err == buntdb.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// ListTags 全量返回指定前缀的标签列表，按名称字典序排序
func (s *Store) ListTags(prefix string) (*model.TagListResult, error) {
	var tags []model.Tag

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(prefix+":*", func(key, value string) bool {
			var tag model.Tag
			if err := json.Unmarshal([]byte(value), &tag); err != nil {
				slog.Warn("skip corrupted tag", "key", key, "err", err)
				return true
			}
			tags = append(tags, tag)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	return &model.TagListResult{Items: tags, Total: len(tags)}, nil
}

// SetTag 写入标签注册表条目（创建或覆盖）
func (s *Store) SetTag(prefix string, tag *model.Tag) error {
	val, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("marshal tag: %w", err)
	}
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(prefix+":"+tag.Name, string(val), nil)
		return err
	})
}

// DeleteTagEntry 删除标签注册表条目
func (s *Store) DeleteTagEntry(prefix, name string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(prefix + ":" + name)
		return err
	})
}

// AdjustTagCount 调整标签 count。标签不存在且 delta > 0 时自动创建。
func (s *Store) AdjustTagCount(prefix, name string, delta int) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		return adjustTagCountTx(tx, prefix, name, delta)
	})
}

func adjustTagCountTx(tx *buntdb.Tx, prefix, name string, delta int) error {
	key := prefix + ":" + name
	val, err := tx.Get(key)
	if err == buntdb.ErrNotFound {
		if delta <= 0 {
			return nil
		}
		tag := model.Tag{Name: name, Count: delta, CreatedAt: time.Now()}
		v, _ := json.Marshal(&tag)
		_, _, err = tx.Set(key, string(v), nil)
		return err
	}
	if err != nil {
		return err
	}

	var tag model.Tag
	if err := json.Unmarshal([]byte(val), &tag); err != nil {
		return err
	}
	tag.Count += delta
	if tag.Count < 0 {
		tag.Count = 0
	}
	v, _ := json.Marshal(&tag)
	_, _, err = tx.Set(key, string(v), nil)
	return err
}

// BatchAdjustTagCounts 批量调整标签 count（单事务）
func (s *Store) BatchAdjustTagCounts(prefix string, deltas map[string]int) error {
	if len(deltas) == 0 {
		return nil
	}
	return s.db.Update(func(tx *buntdb.Tx) error {
		for name, delta := range deltas {
			if err := adjustTagCountTx(tx, prefix, name, delta); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- URL 标签实体级操作（遍历 site + bookmark） ---

type entityKV struct {
	key, value string
}

func (s *Store) collectURLEntities(tx *buntdb.Tx) (sites, bms []entityKV) {
	tx.AscendKeys("site:*", func(key, value string) bool {
		sites = append(sites, entityKV{key, value})
		return true
	})
	tx.AscendKeys("bm:*", func(key, value string) bool {
		bms = append(bms, entityKV{key, value})
		return true
	})
	return
}

// RenameURLTag 重命名 URL 标签：更新所有站点和书签的 tags 数组 + 注册表。
// 返回受影响的实体数量。
func (s *Store) RenameURLTag(req model.RenameTagReq) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark

	err := s.db.Update(func(tx *buntdb.Tx) error {
		if _, err := tx.Get("url_tag:" + req.NewName); err == nil {
			return fmt.Errorf("标签 %q 已存在，请使用合并", req.NewName)
		}
		oldVal, err := tx.Get("url_tag:" + req.OldName)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("标签 %q 不存在", req.OldName)
		}
		if err != nil {
			return err
		}

		siteKVs, bmKVs := s.collectURLEntities(tx)

		for _, kv := range siteKVs {
			var site model.Site
			if json.Unmarshal([]byte(kv.value), &site) != nil {
				continue
			}
			if idx := util.StringIndex(site.Tags, req.OldName); idx >= 0 {
				site.Tags[idx] = req.NewName
				site.EnsureSlices()
				v, _ := json.Marshal(&site)
				tx.Set(kv.key, string(v), nil)
				sitesReindex = append(sitesReindex, site)
			}
		}

		for _, kv := range bmKVs {
			var bm model.Bookmark
			if json.Unmarshal([]byte(kv.value), &bm) != nil {
				continue
			}
			if idx := util.StringIndex(bm.Tags, req.OldName); idx >= 0 {
				bm.Tags[idx] = req.NewName
				bm.EnsureSlices()
				v, _ := json.Marshal(&bm)
				tx.Set(kv.key, string(v), nil)
				bmsReindex = append(bmsReindex, bm)
			}
		}

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

	for i := range sitesReindex {
		s.IndexDoc("site:"+sitesReindex[i].ID, siteBleveFields(&sitesReindex[i]))
	}
	for i := range bmsReindex {
		s.IndexDoc("bm:"+bmsReindex[i].ID, bmBleveFields(&bmsReindex[i]))
	}

	affected := len(sitesReindex) + len(bmsReindex)
	slog.Info("url tag renamed", "old", req.OldName, "new", req.NewName, "affected", affected)
	return affected, nil
}

// MergeURLTag 将 source 标签合并到 target：实体已有 target 则仅移除 source，
// 否则替换 source 为 target。删除 source 注册表，重算 target count。
func (s *Store) MergeURLTag(req model.MergeTagReq) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark

	err := s.db.Update(func(tx *buntdb.Tx) error {
		if _, err := tx.Get("url_tag:" + req.Source); err == buntdb.ErrNotFound {
			return fmt.Errorf("源标签 %q 不存在", req.Source)
		}

		siteKVs, bmKVs := s.collectURLEntities(tx)
		targetCount := 0

		for _, kv := range siteKVs {
			var site model.Site
			if json.Unmarshal([]byte(kv.value), &site) != nil {
				continue
			}
			if util.StringIndex(site.Tags, req.Source) < 0 {
				if util.StringIndex(site.Tags, req.Target) >= 0 {
					targetCount++
				}
				continue
			}
			if util.StringIndex(site.Tags, req.Target) >= 0 {
				site.Tags = util.StringRemove(site.Tags, req.Source)
			} else {
				site.Tags[util.StringIndex(site.Tags, req.Source)] = req.Target
			}
			targetCount++
			site.EnsureSlices()
			v, _ := json.Marshal(&site)
			tx.Set(kv.key, string(v), nil)
			sitesReindex = append(sitesReindex, site)
		}

		for _, kv := range bmKVs {
			var bm model.Bookmark
			if json.Unmarshal([]byte(kv.value), &bm) != nil {
				continue
			}
			if util.StringIndex(bm.Tags, req.Source) < 0 {
				if util.StringIndex(bm.Tags, req.Target) >= 0 {
					targetCount++
				}
				continue
			}
			if util.StringIndex(bm.Tags, req.Target) >= 0 {
				bm.Tags = util.StringRemove(bm.Tags, req.Source)
			} else {
				bm.Tags[util.StringIndex(bm.Tags, req.Source)] = req.Target
			}
			targetCount++
			bm.EnsureSlices()
			v, _ := json.Marshal(&bm)
			tx.Set(kv.key, string(v), nil)
			bmsReindex = append(bmsReindex, bm)
		}

		tx.Delete("url_tag:" + req.Source)

		targetTag := model.Tag{Name: req.Target, Count: targetCount, CreatedAt: time.Now()}
		if tv, err := tx.Get("url_tag:" + req.Target); err == nil {
			var existing model.Tag
			json.Unmarshal([]byte(tv), &existing)
			targetTag.CreatedAt = existing.CreatedAt
		}
		targetTag.Count = targetCount
		v, _ := json.Marshal(&targetTag)
		tx.Set("url_tag:"+req.Target, string(v), nil)

		return nil
	})
	if err != nil {
		return 0, err
	}

	for i := range sitesReindex {
		s.IndexDoc("site:"+sitesReindex[i].ID, siteBleveFields(&sitesReindex[i]))
	}
	for i := range bmsReindex {
		s.IndexDoc("bm:"+bmsReindex[i].ID, bmBleveFields(&bmsReindex[i]))
	}

	affected := len(sitesReindex) + len(bmsReindex)
	slog.Info("url tag merged", "source", req.Source, "target", req.Target, "affected", affected)
	return affected, nil
}

// DeleteURLTagFromEntities 从所有站点和书签中移除指定标签并删除注册表条目。
// 返回受影响的实体数量。
func (s *Store) DeleteURLTagFromEntities(name string) (int, error) {
	var sitesReindex []model.Site
	var bmsReindex []model.Bookmark

	err := s.db.Update(func(tx *buntdb.Tx) error {
		siteKVs, bmKVs := s.collectURLEntities(tx)

		for _, kv := range siteKVs {
			var site model.Site
			if json.Unmarshal([]byte(kv.value), &site) != nil {
				continue
			}
			if util.StringIndex(site.Tags, name) < 0 {
				continue
			}
			site.Tags = util.StringRemove(site.Tags, name)
			site.EnsureSlices()
			v, _ := json.Marshal(&site)
			tx.Set(kv.key, string(v), nil)
			sitesReindex = append(sitesReindex, site)
		}

		for _, kv := range bmKVs {
			var bm model.Bookmark
			if json.Unmarshal([]byte(kv.value), &bm) != nil {
				continue
			}
			if util.StringIndex(bm.Tags, name) < 0 {
				continue
			}
			bm.Tags = util.StringRemove(bm.Tags, name)
			bm.EnsureSlices()
			v, _ := json.Marshal(&bm)
			tx.Set(kv.key, string(v), nil)
			bmsReindex = append(bmsReindex, bm)
		}

		tx.Delete("url_tag:" + name)
		return nil
	})
	if err != nil {
		return 0, err
	}

	for i := range sitesReindex {
		s.IndexDoc("site:"+sitesReindex[i].ID, siteBleveFields(&sitesReindex[i]))
	}
	for i := range bmsReindex {
		s.IndexDoc("bm:"+bmsReindex[i].ID, bmBleveFields(&bmsReindex[i]))
	}

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
