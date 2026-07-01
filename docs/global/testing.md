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

默认读取项目根目录下的 `persist/`，目录不存在时自动 skip。所有查询函数以只读方式打开数据库。

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

> 待补充（阶段二）

### internal/service/

> 待补充（阶段三）
