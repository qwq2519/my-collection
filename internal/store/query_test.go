package store

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/tidwall/buntdb"
)

var (
	flagPersistDir = flag.String("persist-dir", "../../persist", "path to persist directory")
	flagDomain     = flag.String("domain", "", "domain for TestQuery_SiteByDomain")
	flagSearch     = flag.String("search", "", "search query for TestQuery_BleveSearch")
	flagPrefix     = flag.String("prefix", "", "key prefix for TestQuery_RawKeys")
)

// openQueryStore 打开真实 persist 目录的 Store（只读查询用途）。
// persist 目录不存在时 skip，避免 CI 环境报错。
// 设置 SyncPolicy=Never 和 AutoShrinkDisabled=true 避免后台写入。
func openQueryStore(t *testing.T) *Store {
	t.Helper()
	dir := *flagPersistDir
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("persist dir %q not found, skipping query test", dir)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatalf("open store at %q: %v", dir, err)
	}
	s.db.SetConfig(buntdb.Config{
		SyncPolicy:         buntdb.Never,
		AutoShrinkDisabled: true,
	})
	t.Cleanup(func() { s.Close() })
	return s
}

func dumpJSON(t *testing.T, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	fmt.Println(string(data))
}

// --- Query Functions ---

func TestQuery_AllSites(t *testing.T) {
	s := openQueryStore(t)
	result, err := s.ListSites(struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}{Page: 1, PageSize: 1000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	t.Logf("total: %d", result.Total)
	dumpJSON(t, result.Items)
}

func TestQuery_AllBookmarks(t *testing.T) {
	s := openQueryStore(t)

	var all []interface{}
	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("bm:*", func(key, value string) bool {
			var v interface{}
			json.Unmarshal([]byte(value), &v)
			all = append(all, v)
			return true
		})
	})
	if err != nil {
		t.Fatalf("scan bookmarks: %v", err)
	}
	t.Logf("total: %d", len(all))
	dumpJSON(t, all)
}

func TestQuery_AllNotes(t *testing.T) {
	s := openQueryStore(t)

	var all []interface{}
	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("note:*", func(key, value string) bool {
			var v interface{}
			json.Unmarshal([]byte(value), &v)
			all = append(all, v)
			return true
		})
	})
	if err != nil {
		t.Fatalf("scan notes: %v", err)
	}
	t.Logf("total: %d", len(all))
	dumpJSON(t, all)
}

func TestQuery_AllTags(t *testing.T) {
	s := openQueryStore(t)

	result, err := s.ListTags("url_tag")
	if err != nil {
		t.Fatalf("list url_tag: %v", err)
	}
	t.Logf("url_tag count: %d", result.Total)
	dumpJSON(t, result.Items)

	result, err = s.ListTags("media_tag")
	if err != nil {
		t.Fatalf("list media_tag: %v", err)
	}
	t.Logf("media_tag count: %d", result.Total)
	dumpJSON(t, result.Items)
}

func TestQuery_AllFolders(t *testing.T) {
	s := openQueryStore(t)
	folders, err := s.ListFolders()
	if err != nil {
		t.Fatalf("list folders: %v", err)
	}
	t.Logf("total: %d", len(folders))
	dumpJSON(t, folders)
}

func TestQuery_SiteByDomain(t *testing.T) {
	domain := *flagDomain
	if domain == "" {
		t.Skip("use -domain=example.com to query")
	}
	s := openQueryStore(t)
	site, err := s.GetSiteByDomain(domain)
	if err != nil {
		t.Fatalf("get site by domain: %v", err)
	}
	if site == nil {
		t.Logf("no site found for domain %q", domain)
		return
	}
	dumpJSON(t, site)
}

func TestQuery_BleveSearch(t *testing.T) {
	query := *flagSearch
	if query == "" {
		t.Skip("use -search=keyword to query")
	}
	s := openQueryStore(t)

	q := bleve.NewMatchQuery(query)
	req := bleve.NewSearchRequest(q)
	req.Size = 50
	req.Fields = []string{"_type", "title", "domain", "tags"}

	result, err := s.Search(req)
	if err != nil {
		t.Fatalf("bleve search: %v", err)
	}
	t.Logf("total hits: %d", result.Total)
	for _, hit := range result.Hits {
		data, _ := json.Marshal(map[string]interface{}{
			"id":     hit.ID,
			"score":  hit.Score,
			"fields": hit.Fields,
		})
		fmt.Println(string(data))
	}
}

func TestQuery_DirtyItems(t *testing.T) {
	s := openQueryStore(t)
	items := s.GetDirtyItems()
	t.Logf("dirty items: %d", len(items))
	if len(items) > 0 {
		dumpJSON(t, items)
	}

	status := s.GetDirtyIndexStatus()
	dumpJSON(t, status)
}

func TestQuery_RawKeys(t *testing.T) {
	prefix := *flagPrefix
	if prefix == "" {
		t.Skip("use -prefix=site: to scan keys")
	}
	s := openQueryStore(t)

	type kv struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	var items []kv

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys(prefix+"*", func(key, value string) bool {
			var v interface{}
			json.Unmarshal([]byte(value), &v)
			items = append(items, kv{Key: key, Value: v})
			return true
		})
	})
	if err != nil {
		t.Fatalf("scan keys: %v", err)
	}
	t.Logf("keys with prefix %q: %d", prefix, len(items))
	dumpJSON(t, items)
}
