package store

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"collections/internal/model"

	"github.com/blevesearch/bleve/v2"
	"github.com/tidwall/buntdb"
)

var (
	flagPersistDir = flag.String("persist-dir", "../../persist", "path to persist directory")
	flagDomain     = flag.String("domain", "", "domain for TestQuery_SiteByDomain")
	flagSearch     = flag.String("search", "", "search query for TestQuery_BleveSearch / TestQuery_SearchURL")
	flagPrefix     = flag.String("prefix", "", "key prefix for TestQuery_RawKeys")
	flagID         = flag.String("id", "", "entity ID for TestQuery_GetByID")
	flagType       = flag.String("type", "", "entity type (site/bookmark/note) for TestQuery_GetByID")
	flagTag        = flag.String("tag", "", "tag name for TestQuery_ByTag")
)

// openQueryStore 打开真实 persist 目录的 Store（只读查询用途）。
// 将 main.db 复制到临时目录再打开，确保不会误写真实数据。
// Bleve 在临时目录新建空索引（BleveSearch 等搜索工具基于空索引，
// KV 查询类工具不受影响）。persistDir 指回原目录以便文件路径解析。
// persist 目录不存在时自动 skip。
func openQueryStore(t *testing.T) *Store {
	t.Helper()
	srcDir := *flagPersistDir
	srcDB := filepath.Join(srcDir, "main.db")
	if _, err := os.Stat(srcDB); os.IsNotExist(err) {
		t.Skipf("persist dir %q has no main.db, skipping query test", srcDir)
	}

	tmpDir := t.TempDir()
	if err := copyFile(srcDB, filepath.Join(tmpDir, "main.db")); err != nil {
		t.Fatalf("copy main.db to temp dir: %v", err)
	}

	s, err := New(tmpDir)
	if err != nil {
		t.Fatalf("open store at %q: %v", tmpDir, err)
	}
	s.db.SetConfig(buntdb.Config{
		SyncPolicy:         buntdb.Never,
		AutoShrinkDisabled: true,
	})
	s.persistDir = srcDir
	t.Cleanup(func() { s.Close() })
	return s
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func dumpJSON(t *testing.T, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	t.Log(string(data))
}

// --- Query Functions ---

func TestQuery_AllSites(t *testing.T) {
	s := openQueryStore(t)
	result, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	t.Logf("total: %d", result.Total)
	dumpJSON(t, result.Items)
}

func TestQuery_AllBookmarks(t *testing.T) {
	s := openQueryStore(t)

	sites, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}

	total := 0
	for _, site := range sites.Items {
		result, err := s.ListBookmarks(model.BookmarkListReq{
			SiteID: site.ID, Page: 1, PageSize: 10000,
		})
		if err != nil {
			t.Fatalf("list bookmarks for site %s: %v", site.Domain, err)
		}
		total += result.Total
		if result.Total > 0 {
			t.Logf("site %s (%d bookmarks):", site.Domain, result.Total)
			dumpJSON(t, result.Items)
		}
	}
	t.Logf("total bookmarks: %d", total)
}

func TestQuery_AllNotes(t *testing.T) {
	s := openQueryStore(t)

	result, err := s.ListNotes(model.NoteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	t.Logf("total: %d", result.Total)
	dumpJSON(t, result.Items)
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
		t.Log(string(data))
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

// --- 按 ID 查详情 ---
//
//	go test -run TestQuery_GetByID -v ./internal/store/ -type=site -id=xxx
//	go test -run TestQuery_GetByID -v ./internal/store/ -type=bookmark -id=xxx
//	go test -run TestQuery_GetByID -v ./internal/store/ -type=note -id=xxx
func TestQuery_GetByID(t *testing.T) {
	typ := *flagType
	id := *flagID
	if typ == "" || id == "" {
		t.Skip("use -type=site|bookmark|note -id=xxx to query")
	}
	s := openQueryStore(t)

	switch typ {
	case "site":
		site, err := s.GetSite(id)
		if err != nil {
			t.Fatalf("GetSite(%s): %v", id, err)
		}
		dumpJSON(t, site)
	case "bookmark":
		bm, err := s.GetBookmark(id)
		if err != nil {
			t.Fatalf("GetBookmark(%s): %v", id, err)
		}
		dumpJSON(t, bm)
	case "note":
		note, err := s.GetNote(id)
		if err != nil {
			t.Fatalf("GetNote(%s): %v", id, err)
		}
		dumpJSON(t, note)
	default:
		t.Fatalf("unknown type %q, use site|bookmark|note", typ)
	}
}

// --- 按标签筛选 ---
//
//	go test -run TestQuery_ByTag -v ./internal/store/ -tag=react
//	go test -run TestQuery_ByTag -v ./internal/store/ -tag=frontend
func TestQuery_ByTag(t *testing.T) {
	tag := *flagTag
	if tag == "" {
		t.Skip("use -tag=tagname to filter")
	}
	s := openQueryStore(t)

	sites, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}

	type taggedItem struct {
		Type   string      `json:"type"`
		Entity interface{} `json:"entity"`
	}
	var matched []taggedItem

	for _, site := range sites.Items {
		if containsTag(site.Tags, tag) {
			matched = append(matched, taggedItem{Type: "site", Entity: site})
		}
		bms, err := s.ListBookmarks(model.BookmarkListReq{
			SiteID: site.ID, Page: 1, PageSize: 10000,
		})
		if err != nil {
			continue
		}
		for _, bm := range bms.Items {
			if containsTag(bm.Tags, tag) {
				matched = append(matched, taggedItem{Type: "bookmark", Entity: bm})
			}
		}
	}

	t.Logf("entities with tag %q: %d", tag, len(matched))
	dumpJSON(t, matched)
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

// --- 统一搜索（走 SearchURL 接口） ---
//
//	go test -run TestQuery_SearchURL -v ./internal/store/ -search=github
//	go test -run TestQuery_SearchURL -v ./internal/store/ -search=react -tag=frontend
func TestQuery_SearchURL(t *testing.T) {
	query := *flagSearch
	tag := *flagTag
	if query == "" && tag == "" {
		t.Skip("use -search=keyword and/or -tag=tagname to query")
	}
	s := openQueryStore(t)

	req := model.SearchURLReq{Page: 1, PageSize: 50, Search: query}
	if tag != "" {
		req.Tags = []string{tag}
	}

	result, err := s.SearchURL(req)
	if err != nil {
		t.Fatalf("SearchURL: %v", err)
	}
	t.Logf("total site groups: %d, has_more: %v", result.Total, result.HasMore)
	for _, item := range result.Items {
		t.Logf("  site: %s (%s) — %d bookmarks matched",
			item.Site.Title, item.Site.Domain, len(item.Bookmarks))
	}
	dumpJSON(t, result)
}

// --- 数据诊断：Tag Count 一致性检查 ---
//
//	go test -run TestQuery_TagConsistency -v ./internal/store/
func TestQuery_TagConsistency(t *testing.T) {
	s := openQueryStore(t)

	actual := make(map[string]int)

	sites, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	for _, site := range sites.Items {
		for _, tag := range site.Tags {
			actual[tag]++
		}
		bms, err := s.ListBookmarks(model.BookmarkListReq{
			SiteID: site.ID, Page: 1, PageSize: 10000,
		})
		if err != nil {
			continue
		}
		for _, bm := range bms.Items {
			for _, tag := range bm.Tags {
				actual[tag]++
			}
		}
	}

	registered, err := s.ListTags("url_tag")
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	regMap := make(map[string]int)
	for _, tag := range registered.Items {
		regMap[tag.Name] = tag.Count
	}

	type mismatch struct {
		Tag         string `json:"tag"`
		Registered  int    `json:"registered"`
		ActualCount int    `json:"actual"`
	}
	var issues []mismatch

	allTags := make(map[string]bool)
	for k := range actual {
		allTags[k] = true
	}
	for k := range regMap {
		allTags[k] = true
	}

	for tag := range allTags {
		reg := regMap[tag]
		act := actual[tag]
		if reg != act {
			issues = append(issues, mismatch{Tag: tag, Registered: reg, ActualCount: act})
		}
	}

	if len(issues) == 0 {
		t.Logf("all %d tags consistent", len(allTags))
	} else {
		t.Logf("found %d tag count mismatches:", len(issues))
		dumpJSON(t, issues)
	}
}

// --- 数据诊断：孤儿文件检测 ---
//
//	go test -run TestQuery_OrphanAssets -v ./internal/store/
func TestQuery_OrphanAssets(t *testing.T) {
	s := openQueryStore(t)
	persistDir := s.PersistDir()

	sites, err := s.ListSites(model.SiteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}

	type orphan struct {
		Category string `json:"category"`
		Path     string `json:"path"`
	}
	var orphans []orphan

	// 1. icon 文件 vs site.Icon 引用
	iconDir := filepath.Join(persistDir, "url-assets", "icons")
	if entries, err := os.ReadDir(iconDir); err == nil {
		iconRefs := make(map[string]bool)
		for _, site := range sites.Items {
			if site.Icon != "" {
				iconRefs[site.Icon] = true
			}
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !iconRefs[e.Name()] {
				orphans = append(orphans, orphan{"icon", e.Name()})
			}
		}
	}

	// 2. attachment 目录 vs site/bookmark ID
	entityIDs := make(map[string]bool)
	for _, site := range sites.Items {
		entityIDs[site.ID] = true
		bms, err := s.ListBookmarks(model.BookmarkListReq{
			SiteID: site.ID, Page: 1, PageSize: 10000,
		})
		if err != nil {
			continue
		}
		for _, bm := range bms.Items {
			entityIDs[bm.ID] = true
		}
	}

	attachDir := filepath.Join(persistDir, "url-assets", "attachments")
	if entries, err := os.ReadDir(attachDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if !entityIDs[e.Name()] {
				orphans = append(orphans, orphan{"attachment-dir", e.Name()})
			}
		}
	}

	// 3. note-images 目录 vs note ID
	notes, err := s.ListNotes(model.NoteListReq{Page: 1, PageSize: 10000})
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	noteIDs := make(map[string]bool)
	for _, n := range notes.Items {
		noteIDs[n.ID] = true
	}

	noteImgDir := filepath.Join(persistDir, "note-images")
	if entries, err := os.ReadDir(noteImgDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if !noteIDs[e.Name()] {
				orphans = append(orphans, orphan{"note-images-dir", e.Name()})
			}
		}
	}

	if len(orphans) == 0 {
		t.Log("no orphan assets found")
	} else {
		t.Logf("found %d orphan assets:", len(orphans))
		dumpJSON(t, orphans)
	}
}
