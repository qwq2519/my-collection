package store

import (
	"encoding/json"
	"fmt"

	"collections/internal/model"

	"github.com/tidwall/buntdb"
)

// GetSiteByDomain 通过域名查找站点，利用 idx:site_domain 索引。
// 未找到返回 (nil, nil)。
func (s *Store) GetSiteByDomain(domain string) (*model.Site, error) {
	var site *model.Site

	pivot, err := json.Marshal(map[string]string{"domain": domain})
	if err != nil {
		return nil, fmt.Errorf("marshal pivot: %w", err)
	}

	err = s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendEqual("idx:site_domain", string(pivot), func(key, value string) bool {
			var s model.Site
			if err := json.Unmarshal([]byte(value), &s); err == nil {
				site = &s
			}
			return false
		})
	})
	if err != nil {
		return nil, fmt.Errorf("lookup site by domain: %w", err)
	}

	return site, nil
}
