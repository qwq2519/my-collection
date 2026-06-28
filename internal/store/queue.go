package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"collections/internal/model"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

// AddToQueue 将 URL 加入临时队列。
//
// TODO: 入队去重 —— 对 rawURL 做归一化后，同时校验：
//  1. 队列中是否已有相同归一化 URL（遍历 queue:* 逐条归一化比对）
//  2. 已有书签中是否已收藏该 URL（复用 util.NormalizeURL + 按域名查站点 + 遍历同站点书签）
//     已存在则拒绝入队并提示。
func (s *Store) AddToQueue(rawURL string) (*model.QueueItem, error) {
	item := &model.QueueItem{
		ID:      uuid.New().String(),
		URL:     rawURL,
		AddedAt: time.Now(),
	}

	val, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("marshal queue item: %w", err)
	}

	err = s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("queue:"+item.ID, string(val), nil)
		return err
	})
	if err != nil {
		return nil, err
	}

	slog.Info("url added to queue", "id", item.ID, "url", item.URL)
	return item, nil
}

// ListQueue 列出所有队列条目，按 added_at 降序排列
func (s *Store) ListQueue() ([]model.QueueItem, error) {
	var items []model.QueueItem

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.Descend("idx:queue_added", func(key, value string) bool {
			var item model.QueueItem
			if err := json.Unmarshal([]byte(value), &item); err != nil {
				slog.Warn("skip corrupted queue item", "key", key, "err", err)
				return true
			}
			items = append(items, item)
			return true
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list queue: %w", err)
	}

	return items, nil
}

// DeleteQueueItem 按 ID 删除队列条目
func (s *Store) DeleteQueueItem(id string) error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete("queue:" + id)
		if err == buntdb.ErrNotFound {
			return fmt.Errorf("队列条目不存在")
		}
		return err
	})
}

// ClearQueue 清空整个临时队列
func (s *Store) ClearQueue() error {
	return s.db.Update(func(tx *buntdb.Tx) error {
		var keys []string
		tx.AscendKeys("queue:*", func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
		for _, k := range keys {
			if _, err := tx.Delete(k); err != nil && err != buntdb.ErrNotFound {
				return fmt.Errorf("delete queue item %s: %w", k, err)
			}
		}
		return nil
	})
}
