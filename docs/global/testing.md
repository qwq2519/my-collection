# 测试指南

## 运行测试

```bash
# 运行所有后端测试
go test ./internal/...

# 运行指定包
go test ./internal/model/
go test ./internal/util/
go test ./internal/store/
go test ./internal/service/

# 详细输出
go test -v ./internal/...

# 单个测试函数
go test -v -run TestValidateURL ./internal/util/

# Race 检测
go test -race ./internal/...

# 覆盖率
go test -cover ./internal/...
go test -coverprofile=coverage.out ./internal/... && go tool cover -html=coverage.out
```

## Store 查询调试工具

`internal/store/query_test.go` 提供一组 `TestQuery_*` 函数，可直接查询项目真实数据库（`persist/` 目录），类似 Postman / curl 的调试体验。

```bash
# 查看所有站点
go test -run TestQuery_AllSites -v ./internal/store/

# 查看所有书签
go test -run TestQuery_AllBookmarks -v ./internal/store/

# 查看所有笔记
go test -run TestQuery_AllNotes -v ./internal/store/

# 查看所有标签
go test -run TestQuery_AllTags -v ./internal/store/

# 查看所有媒体文件夹
go test -run TestQuery_AllFolders -v ./internal/store/

# 按域名查站点
go test -run TestQuery_SiteByDomain -v ./internal/store/ -domain=github.com

# Bleve 全文搜索
go test -run TestQuery_BleveSearch -v ./internal/store/ -search="react"

# 按前缀扫描原始 KV（调试利器）
go test -run TestQuery_RawKeys -v ./internal/store/ -prefix=site:
go test -run TestQuery_RawKeys -v ./internal/store/ -prefix=bm:
go test -run TestQuery_RawKeys -v ./internal/store/ -prefix=url_tag:

# 查看脏索引队列
go test -run TestQuery_DirtyItems -v ./internal/store/

# 指定其他 persist 目录（如备份恢复后的目录）
go test -run TestQuery_AllSites -v ./internal/store/ -persist-dir=/path/to/backup/persist
```

默认读取项目根目录下的 `persist/`，目录不存在时自动 skip。所有查询函数以只读方式打开数据库（SyncPolicy=Never）。

## Fetcher 调试工具

`internal/service/fetcher_test.go` 提供一组 `TestFetch_*` 函数，可手动调用来抓取页面元数据、下载 icon，类似 curl 的调试体验。

```bash
# 抓取页面完整元数据（title、description、icon、og:image），icon 自动保存
go test -run TestFetch_Metadata -v ./internal/service/ -fetch-url=https://github.com

# 仅解析 HTML meta 信息（不下载 icon，不需要磁盘写入）
go test -run TestFetch_PageMeta -v ./internal/service/ -fetch-url=https://react.dev

# 下载指定 URL 的图片/icon
go test -run TestFetch_Icon -v ./internal/service/ -icon-url="https://www.google.com/s2/favicons?domain=github.com&sz=64"

# 保存到指定目录（默认用临时目录，测试完自动清理）
go test -run TestFetch_Metadata -v ./internal/service/ -fetch-url=https://github.com -save-dir=./tmp
go test -run TestFetch_Icon -v ./internal/service/ -icon-url=https://react.dev/favicon.ico -save-dir=./tmp
```

无 flag 时自动 skip，不影响 `go test ./internal/...` 全量运行。

## 测试覆盖情况

### internal/model/

| 测试文件 | 被测文件 | 覆盖场景 |
|---------|---------|---------|
| `consts_test.go` | `consts.go` | `LookupMediaType` 各扩展名映射、大小写、未知扩展名；`IsSupportedMediaExt` 正向和反向 |
| `model_test.go` | `bookmark.go`, `site.go` | `Bookmark.EnsureSlices` nil→空切片；`Site.EnsureSlices` nil→空切片；已有值时不覆盖 |

### internal/util/

| 测试文件 | 被测文件 | 覆盖场景 |
|---------|---------|---------|
| `urlutil_test.go` | `urlutil.go` | `ValidateURL` 合法/非法 URL；`NormalizeURL` 去 www/fragment/排序参数；`ExtractDomain` 域名提取 |
| `tagutil_test.go` | `tagutil.go` | `ValidateTagName` 合法/非法标签名；`NormalizeTagName` trim/小写/压缩空格 |
| `mdutil_test.go` | `mdutil.go` | `StripMarkdown` 标题/加粗/链接/图片/代码块/引用 |
| `sliceutil_test.go` | `sliceutil.go` | `StringIndex` 存在/不存在/空切片；`StringRemove` 移除/重复/不存在 |
| `fileutil_test.go` | `fileutil.go` | `AtomicWrite` 写入正确性/目录自动创建；`SafePath` 正常路径/`../` 遍历拒绝 |

### internal/store/

集成测试，每个测试用 `t.TempDir()` 创建独立的 BuntDB + Bleve 实例，测试后自动清理。

| 测试文件 | 被测文件 | 覆盖场景 |
|---------|---------|---------|
| `testhelper_test.go` | `store.go` | `newTestStore` 辅助函数：创建临时 Store 实例 |
| `query_test.go` | 全部 | 查询调试工具：AllSites/AllBookmarks/AllNotes/AllTags/AllFolders/SiteByDomain/BleveSearch/RawKeys/DirtyItems |
| `site_test.go` | `site.go`, `site_list.go` | 创建字段校验、域名唯一性、Get+Update、Delete、有书签时拒绝删除、分页列表、GetSiteByDomain |
| `bookmark_test.go` | `bookmark.go`, `bookmark_list.go` | 创建+bookmark_count 递增、URL 去重、Get+Update、Delete+count 递减、BatchDelete、分页列表、NotFound |
| `note_test.go` | `note.go` | 创建、Get+Update、Delete、NotFound、分页列表、Bleve 标题搜索、中文 body 搜索 |
| `tag_test.go` | `tag.go`, `tag_ops.go` | Set/Get、NotFound、List、Delete、AdjustCount 自动创建/递增/递减/clamp、BatchAdjust、Rename+实体更新、Rename 冲突、Merge 去重、DeleteFromEntities、Recount |
| `media_test.go` | `media.go` | 文件夹 CRUD、ListFolders、media_meta.json 读写 round-trip、tree_hash.json 读写、ReadMeta 文件不存在 |
| `search_test.go` | `search.go`, `index.go` | IndexDoc+Search、DeleteDoc、文本搜索、标签搜索、dirty 队列生命周期、ClearDirtyByType、RebuildIndexByType 选择性重建 |

### internal/service/

集成测试，注入真实 Store 实例测试业务逻辑。AssetHandler 使用 `httptest` 测试 HTTP 行为。

| 测试文件 | 被测文件 | 覆盖场景 |
|---------|---------|---------|
| `url_test.go` | `url.go` | CreateSite 校验（空 title/URL）、自动提取 domain、标签归一化去重、非法标签拒绝；UpdateSite 校验、标签替换；DeleteSite 标签清理；CreateBookmark 校验（非法 URL、空 title）、自动匹配站点、无站点拒绝；Update/DeleteBookmark 校验；LookupSiteByURL found/not-found/校验；NormalizeURL |
| `note_test.go` | `note.go` | Create/Get/Update/Delete 校验；DeleteNote 清理图片目录；DetectOrphanImages 比对 body 引用找孤儿文件、目录不存在；DeleteOrphanImages 删除文件、路径遍历跳过 |
| `upload_test.go` | `upload.go` | 输入校验（空 scene/filename/data）；10MB 大小限制；不支持扩展名拒绝；未知 scene 拒绝；site-icon 路由和持久化；note-image 内容 hash 命名；attachment 保存和允许格式；DeleteAttachment 文件删除和校验 |
| `asset_test.go` | `asset.go` | 白名单路径正常服务；Cache-Control 设置；非白名单路径 404；路径遍历拒绝；POST/PUT/DELETE 405；非 persist 路径委托给 embedded handler；三个白名单前缀全覆盖 |
| `fetcher_test.go` | `fetcher.go` | 调试工具：Metadata 完整抓取、PageMeta 仅解析 HTML、Icon 下载保存（手动调用，无 flag 时 skip） |
