package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"

	"github.com/tidwall/buntdb"
)

// --- 标签注册表原子操作（url_tag / media_tag 共用） ---

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
