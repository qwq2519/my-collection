package model

import "time"

// QueueItem 临时队列条目（无对应站点时暂存 URL）
type QueueItem struct {
	ID      string    `json:"id"`
	URL     string    `json:"url"`
	AddedAt time.Time `json:"added_at"`
}
