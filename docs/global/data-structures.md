# 数据结构

集中描述所有存储层的数据结构，便于对比和实现时参照。

设计原则详见 [技术栈与存储](./tech-stack.md)，各功能的需求和行为规则见 `features/` 下对应文档。

## persist 目录结构

```text
persist/
├── main.db                    ← BuntDB 主存储（单文件）
├── search.bleve/              ← Bleve 搜索索引目录（可重建）
├── url-assets/                ← URL 模块资源（站点 + 书签相关文件）
│   ├── icons/                 ← 站点/书签图标（按域名命名，可重新抓取）
│   │   └── {domain}.{ext}
│   ├── covers/                ← 封面图（用户上传）
│   │   └── {entity_id}.{ext}
│   └── attachments/           ← 附件（用户上传的图片、文件等）
│       └── {entity_id}/
│           └── {filename}
├── note-images/               ← 笔记中粘贴的图片
│   └── {note_id}/
│       └── {hash}.{ext}
└── media-folders/                   ← 媒体管理数据（每文件夹独立）
    └── {folder_id}/
        ├── media_meta.json         ← 媒体元数据（图片+视频）
        ├── tree_hash.json          ← Merkle Tree 快照
        └── thumbnails/             ← 缩略图 + 动画预览
            ├── {hash}.jpg          ← 静态缩略图（所有文件）
            └── {hash}.preview.webp ← 动画预览（仅视频，ffmpeg 可用时）
```

- `main.db` + `media-folders/*/media_meta.json` 是核心数据，丢失不可恢复
- `search.bleve/` 可从 main.db + media_meta.json 全量重建
- `url-assets/icons/` 可通过重新抓取恢复
- `url-assets/covers/` 是用户上传的封面图，删除则封面丢失
- `url-assets/attachments/` 是用户上传的附件，删除则附件丢失
- `thumbnails/` 可从源文件重新生成
- `note-images/` 是笔记引用的图片实体，删除则笔记中图片丢失

**删除清理规则**：删除站点或书签时，同步删除 `persist/url-assets/covers/{entity_id}.*` 和 `persist/url-assets/attachments/{entity_id}/` 整个目录。

## BuntDB Key Schema

BuntDB 以 `前缀:id` 作为 key，value 为 JSON 字符串。

### 总览

| Key 格式 | 实体 | 说明 |
|----------|------|------|
| `site:{id}` | 站点 | 手动创建的网站 |
| `bm:{id}` | 书签 | 具体 URL，必须归属于某个站点 |
| `queue:{id}` | 临时队列 | 仅存 URL，添加书签时无对应站点则暂存于此 |
| `note:{id}` | 笔记 | Markdown 短笔记 |
| `folder:{id}` | 媒体文件夹 | 注册表，实际媒体数据在 JSON 文件中 |
| `url_tag:{name}` | URL 标签 | 标签注册表，`{name}` 为小写标签字符串 |
| `media_tag:{name}` | 媒体标签 | 标签注册表，`{name}` 为小写标签字符串 |
| `meta:{key}` | 元信息 | 应用级标志位 |

### ID 生成规则

所有实体 ID 使用 UUID v4（如 `550e8400-e29b-41d4-a716-446655440000`），创建时生成，永不改变。

---

### site — 站点

```text
Key:   site:{site_id}
```

```json
{
  "id": "a1b2c3d4-...",
  "title": "GitHub",
  "domain": "github.com",
  "icon": "github.com.png",
  "cover": "a1b2c3d4.png",
  "description": "代码托管平台",
  "tags": ["开发", "工具::代码托管"],
  "attachments": [
    { "filename": "screenshot.png", "label": "首页截图" }
  ],
  "bookmark_count": 3,
  "created_at": "2026-06-22T10:00:00Z",
  "updated_at": "2026-06-22T10:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID，同 key 中的 `{site_id}` |
| title | string | 是 | 显示名称 |
| domain | string | 是 | 归一化后的完整域名（去 www），用于自动归组，**站点间唯一，创建后不可修改** |
| icon | string | 否 | 图标文件名，存储在 `persist/url-assets/icons/`，以域名命名（如 `github.com.png`） |
| cover | string | 否 | 封面图文件名，存储在 `persist/url-assets/covers/{entity_id}.{ext}`，列表页主展示图 |
| description | string | 否 | 站点描述 |
| tags | string[] | 否 | 标签列表，与书签共享同一套 URL 标签体系 |
| attachments | object[] | 否 | 附件列表，文件存储在 `persist/url-assets/attachments/{entity_id}/` |
| attachments[*].filename | string | 是 | 附件文件名 |
| attachments[*].label | string | 否 | 附件描述/标注 |
| bookmark_count | int | 是 | 该站点下的书签数量，增删书签时同步维护 |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

**唯一性约束：** 同一 domain 只能创建一个站点，创建时校验 `idx:site_domain` 索引，已存在则拒绝并提示。

**bookmark_count 维护规则：**

| 操作 | 维护动作 |
|------|---------|
| 添加书签并关联到站点 | 对应站点 `bookmark_count + 1` |
| 删除书签（单条或批量） | 对应站点 `bookmark_count - 1`（批量时按站点汇总后一次性更新） |
| 书签从临时队列分配到站点 | 目标站点 `bookmark_count + 1` |

**updated_at 联动规则：** 书签的增删改操作会同步更新所属站点的 `updated_at` 为当前时间，使站点列表按 `updated_at` 排序时反映最新书签活动。

**删除约束：** 仅允许删除 `bookmark_count == 0` 的空站点。尝试删除非空站点时返回错误，拒绝执行。

**查询模式：**

- 按 ID 精确查询：`site:{id}`
- 按域名查找站点：BuntDB 自定义索引 `idx:site_domain`，索引 `domain` 字段
- 列出所有站点：前缀扫描 `site:*`

---

### bm — 书签（URL）

```text
Key:   bm:{bm_id}
```

```json
{
  "id": "e5f6a7b8-...",
  "url": "https://github.com/golang/go",
  "normalized_url": "github.com/golang/go",
  "domain": "github.com",
  "site_id": "a1b2c3d4-...",
  "title": "golang/go",
  "cover": "e5f6a7b8.gif",
  "description": "Go 语言主仓库",
  "tags": ["go", "开源"],
  "attachments": [
    { "filename": "demo.png", "label": "效果演示" }
  ],
  "status": "alive",
  "created_at": "2026-06-22T12:00:00Z",
  "updated_at": "2026-06-22T12:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID |
| url | string | 是 | 用户输入的原始 URL，**创建后不可修改** |
| normalized_url | string | 是 | 归一化后的 URL，用于去重比对（后端自动生成，随 URL 固定不变） |
| domain | string | 是 | 从 URL 提取并归一化的域名（随 URL 固定不变） |
| site_id | string | 是 | 所属站点 ID，书签必须归属于已有站点 |
| title | string | 是 | 页面标题 |
| cover | string | 否 | 封面图文件名，存储在 `persist/url-assets/covers/{entity_id}.{ext}`，支持静态图和 GIF |
| description | string | 否 | 页面描述 |
| tags | string[] | 否 | 标签列表，独立于站点 |
| attachments | object[] | 否 | 附件列表，文件存储在 `persist/url-assets/attachments/{entity_id}/` |
| attachments[*].filename | string | 是 | 附件文件名 |
| attachments[*].label | string | 否 | 附件描述/标注 |
| status | string | 否 | `"alive"` / `"dead"`，默认 `"alive"` |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

**URL 归一化规则（`normalized_url`）：**

后端提供 `NormalizeURL(rawURL) → normalizedURL` 方法，前端添加书签时自动调用并展示归一化结果供用户确认。

| 处理项 | 规则 | 示例 |
|--------|------|------|
| 协议 | 去除 `http://` / `https://` | `https://github.com/go` → `github.com/go` |
| www 前缀 | 去除 | `www.example.com/page` → `example.com/page` |
| 尾部斜杠 | 去除 | `github.com/go/` → `github.com/go` |
| fragment | 去除 `#` 及之后内容 | `page.html#section` → `page.html` |
| 查询参数 | 保留原序 | `?a=1&b=2` 保持不变 |
| domain 大小写 | 转小写 | `GitHub.COM/Go` → `github.com/Go` |
| path 大小写 | 保持原样 | path 部分区分大小写 |

**去重**：添加书签时，用 `normalized_url` 在已有书签中查找，存在则拒绝。需要 BuntDB 自定义索引支持。

**添加书签流程：**

1. 用户输入 URL，后端提取域名并归一化
2. 通过 `idx:site_domain` 查找是否存在对应站点
3. 若站点存在 → 创建书签，`site_id` 指向该站点
4. 若站点不存在 → **拒绝创建书签**，改为存入临时队列（`queue:{id}`），仅保存 URL

**查询模式：**

- 按 ID 精确查询：`bm:{id}`
- 按站点列出书签：BuntDB 自定义索引 `idx:bm_site`，索引 `site_id` 字段
- 按域名查找书签：BuntDB 自定义索引 `idx:bm_domain`，索引 `domain` 字段
- 按归一化 URL 查重：BuntDB 自定义索引 `idx:bm_normalized_url`，索引 `normalized_url` 字段
- 标签筛选（单标签 / 多标签组合）：走 Bleve keyword 精确匹配（tags 为数组，BuntDB 不支持数组字段索引）
- 全文搜索：走 Bleve

---

### queue — 临时队列

```text
Key:   queue:{queue_id}
```

```json
{
  "id": "d1e2f3a4-...",
  "url": "https://example.com/some-page",
  "added_at": "2026-06-22T14:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID |
| url | string | 是 | 用户输入的原始 URL |
| added_at | string | 是 | ISO 8601，加入队列的时间 |

临时队列是一个极简的 URL 暂存区。添加书签时若域名无对应站点，URL 自动存入此队列。队列条目只保存 URL 本身，不含 title、tags 等元数据，不参与搜索索引（不进 Bleve）。

用户可在队列中查看和删除条目。如需正式收藏，用户先创建对应站点，再手动添加书签。

**查询模式：**

- 列出所有队列条目：前缀扫描 `queue:*`
- 按 ID 删除：`queue:{id}`

---

### note — 笔记

```text
Key:   note:{note_id}
```

```json
{
  "id": "c9d0e1f2-...",
  "title": "学习笔记",
  "body": "今天学了 Go 的接口...\n\n![图片](note-images/c9d0e1f2-.../ab3f.png)",
  "created_at": "2026-06-20T09:00:00Z",
  "updated_at": "2026-06-22T15:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID |
| title | string | 是 | 标题 |
| body | string | 否 | Markdown 正文，图片引用存储相对路径（如 `note-images/{note_id}/{hash}.png`），前端渲染时加 `/persist/` 前缀 |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

**查询模式：**

- 按 ID 精确查询：`note:{id}`
- 列出所有笔记：前缀扫描 `note:*`
- 全文搜索（title + body）：走 Bleve

---

### folder — 媒体文件夹注册表

```text
Key:   folder:{folder_id}
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "path": "D:\\Photos",
  "name": "照片",
  "file_count": 128,
  "added_at": "2026-06-18T08:00:00Z",
  "last_scan_at": "2026-06-22T10:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID，即 folder_id |
| path | string | 是 | 文件夹绝对路径 |
| name | string | 是 | 显示名称 |
| file_count | int | 是 | 该文件夹下的媒体文件数量，扫描完成后同步更新 |
| added_at | string | 是 | ISO 8601 |
| last_scan_at | string | 否 | 上次扫描完成时间（ISO 8601），未扫描过为空 |

实际媒体数据存储在 `persist/media-folders/{folder_id}/` 下的 JSON 文件中，不在 BuntDB 内。

---

### url_tag — URL 标签注册表

```text
Key:   url_tag:{name}
```

```json
{
  "name": "前端::react",
  "count": 12,
  "created_at": "2026-06-22T10:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 标签字符串（小写），同 key 中的 `{name}` |
| count | int | 是 | 关联实体数量（站点 + 书签），增删标签时同步维护 |
| created_at | string | 是 | ISO 8601，首次使用时自动创建 |

站点和书签共享同一套 URL 标签体系，对 count 的贡献相同（各计 1）。

**查询模式：**

- 列出所有 URL 标签：前缀扫描 `url_tag:*`
- 按名称查找：`url_tag:{name}`

---

### media_tag — 媒体标签注册表

```text
Key:   media_tag:{name}
```

```json
{
  "name": "风景",
  "count": 5,
  "created_at": "2026-06-20T08:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 标签字符串（小写），同 key 中的 `{name}` |
| count | int | 是 | 关联媒体文件数量，增删标签时同步维护 |
| created_at | string | 是 | ISO 8601，首次使用时自动创建 |

**查询模式：**

- 列出所有媒体标签：前缀扫描 `media_tag:*`
- 按名称查找：`media_tag:{name}`

**标签规则**：大小写不敏感，存储时统一转小写。`::` 为逻辑层级分隔符，后端不解析层级关系。详见 [标签系统](./tag-system.md)。

**标签名字符规则**：

| 允许 | 说明 |
|------|------|
| 中文字符 | 主要使用场景 |
| 英文字母 | 存储时转小写 |
| 数字 | 如年份标签 `2026` |
| `::` | 层级分隔符，允许出现在标签中间 |
| 空格 | 允许，自动 trim 首尾，连续空格归一化为单空格 |
| `-`、`_` | 常用连接符 |

| 禁止 | 理由 |
|------|------|
| 单独的 `:` | 与 `::` 混淆，与 BuntDB key 分隔符混淆 |
| 前导/尾随 `::` | 如 `::react` 或 `react::` 没有语义 |
| 空字符串 | 无意义 |

后端在创建/重命名标签时做校验，不符合规则的输入拒绝并提示。

---

### meta — 元信息

```text
Key:   meta:{key}
Value: 字符串（非 JSON）
```

| Key | Value | 说明 |
|-----|-------|------|
| `meta:index_dirty` | `"true"` / `"false"` | Bleve 索引是否需要重建 |

后续可扩展其他元信息（如数据版本号等）。

---

## BuntDB 自定义索引

BuntDB 支持基于 JSON 字段创建自定义索引，用于加速非主键查询。

| 索引名 | 目标 Key 前缀 | 索引字段 | 用途 |
|--------|-------------|---------|------|
| `idx:site_domain` | `site:*` | `.domain` | 按域名查找站点（添加 URL 时自动归组） |
| `idx:bm_site` | `bm:*` | `.site_id` | 按站点列出书签 |
| `idx:bm_domain` | `bm:*` | `.domain` | 按域名查找书签 |
| `idx:bm_normalized_url` | `bm:*` | `.normalized_url` | URL 去重校验 |

---

## 每文件夹 JSON 文件

媒体管理的元数据和变化检测快照以 JSON 文件存储在 `persist/media-folders/{folder_id}/` 下，独立于 BuntDB。

### Hash 算法选型

| 用途 | 算法 | 包 | 理由 |
|------|------|---|------|
| Merkle Tree 变化检测 | xxHash (xxh64) | `github.com/cespare/xxhash/v2` | 非密码学场景，速度优先 |
| 缩略图文件命名 | SHA-256 取前 16 字符 | Go 标准库 `crypto/sha256` | 需要稳定、低碰撞的文件名 |

**决策记录（Merkle Tree hash）：**

候选方案：FNV-1a（Go 标准库 `hash/fnv`）、xxHash（`github.com/cespare/xxhash/v2`）。

当前 Merkle Tree 的输入是短字符串（`filename + mtime + size`），两者速度差异不大。但后续可能扩展为对文件内容做 hash（重复检测、更精确的变化检测），输入为 MB-GB 级数据，此时 xxHash 比 FNV-1a 快约 7 倍（~10 GB/s vs ~1.5 GB/s），差距显著。

选 xxHash 一步到位，避免后续混用两套 hash 或做迁移。多一个外部依赖，但 `cespare/xxhash` 是成熟稳定的小库，可接受。

### media_meta.json — 媒体元数据

```json
{
  "schema_version": 1,
  "folder_id": "550e8400-e29b-41d4-a716-446655440000",
  "files": {
    "photo1.jpg": {
      "media_type": "image",
      "tags": ["风景", "2026"],
      "thumbnail": "a3f2b8c1e5d7f9ab.jpg",
      "added_at": "2026-06-22T10:00:00Z",
      "file_size": 2048576,
      "dimensions": [1920, 1080]
    },
    "子目录A/trip.mp4": {
      "media_type": "video",
      "tags": ["旅行::日本"],
      "thumbnail": "9c1d4e7f20a3b6d8.jpg",
      "preview": "9c1d4e7f20a3b6d8.preview.webp",
      "added_at": "2026-06-21T14:00:00Z",
      "file_size": 104857600,
      "dimensions": [1920, 1080],
      "duration": 182.5
    }
  }
}
```

| 字段 | 说明 |
|------|------|
| `schema_version` | 格式版本号，用于后续迁移 |
| `folder_id` | 冗余记录，与目录名一致 |
| `files` | Map，key 为相对路径，value 为媒体元数据 |
| `files[*].media_type` | `"image"` 或 `"video"` |
| `files[*].tags` | 标签数组 |
| `files[*].thumbnail` | 静态缩略图文件名（`sha256(folder_id/rel_path)[:16].jpg`） |
| `files[*].preview` | 动画预览文件名（仅视频，`{hash}.preview.webp`），无则为空 |
| `files[*].added_at` | 首次扫描发现的时间 |
| `files[*].file_size` | 文件大小（字节） |
| `files[*].dimensions` | `[width, height]`，获取失败时为 `null` |
| `files[*].duration` | 仅视频，时长（秒），ffmpeg 不可用时为 null |

写入策略：写临时文件 → rename 原子替换。

### tree_hash.json — Merkle Tree 快照

```json
{
  "schema_version": 1,
  "folder_id": "550e8400-e29b-41d4-a716-446655440000",
  "root": {
    "hash": "a1b2c3d4e5f6...",
    "dir_mtime": "2026-06-22T10:00:00Z",
    "children": {
      "photo1.jpg": {
        "type": "file",
        "hash": "f1a2b3c4...",
        "mtime": "2026-06-20T08:00:00Z",
        "size": 2048576
      },
      "子目录A": {
        "type": "dir",
        "hash": "d5e6f7a8...",
        "dir_mtime": "2026-06-21T15:00:00Z",
        "children": {
          "img1.jpg": {
            "type": "file",
            "hash": "b9c0d1e2...",
            "mtime": "2026-06-20T08:00:00Z",
            "size": 3145728
          }
        }
      }
    }
  }
}
```

| 字段 | 说明 |
|------|------|
| `root` | 根目录节点 |
| `*.type` | `"file"` 或 `"dir"` |
| `*.hash` | 文件：`xxh64(relative_path + mtime + size)`，relative_path 为从文件夹根到该文件的相对路径（如 `子目录A/img1.jpg`）；目录：`xxh64(sorted(child_hashes))` |
| `*.dir_mtime` | 仅目录节点，用于 dir_mtime + children 双重剪枝优化（详见 [媒体管理 - 扫描算法](../features/media-manager.md#merkle-tree-变化检测)） |
| `*.mtime` | 仅文件节点，文件修改时间 |
| `*.size` | 仅文件节点，文件大小 |
| `*.children` | 仅目录节点，Map，key 为文件/目录名 |

扫描时对比新旧 tree_hash.json 生成 diff，详见 [媒体管理 - Merkle Tree 变化检测](../features/media-manager.md#merkle-tree-变化检测)。

---

## Bleve 索引结构

Bleve 索引目录：`persist/search.bleve/`

所有实体共用一个 Bleve 索引，通过 `_type` 字段区分实体类型。

### 中文分词

使用 gse（`github.com/go-ego/gse`）作为 Bleve 的中文 analyzer，纯 Go 实现，无 CGO 依赖。

- 自定义 analyzer 名称：`"gse"`，注册到 Bleve index mapping
- 所有 `text（分词）` 类型的字段统一使用该 analyzer
- `keyword` 类型字段（tags、domain、url 等）不经过分词，不受影响
- gse 自带词典约 10-15MB，会嵌入二进制或随应用分发

### 文档结构

**站点（Site）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"site:{site_id}"` | — |
| `_type` | `"site"` | keyword |
| `title` | site.title | text（分词） |
| `description` | site.description | text（分词） |
| `domain` | site.domain | keyword |
| `domain_text` | site.domain | text（分词） |
| `tags` | site.tags | keyword（多值） |
| `updated_at` | site.updated_at | datetime |

**书签（Bookmark）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"bm:{bm_id}"` | — |
| `_type` | `"bookmark"` | keyword |
| `title` | bookmark.title | text（分词） |
| `description` | bookmark.description | text（分词） |
| `domain` | bookmark.domain | keyword |
| `domain_text` | bookmark.domain | text（分词） |
| `tags` | bookmark.tags | keyword（多值） |
| `url` | bookmark.url | keyword |
| `updated_at` | bookmark.updated_at | datetime |

**笔记（Note）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"note:{note_id}"` | — |
| `_type` | `"note"` | keyword |
| `title` | note.title | text（分词） |
| `body` | note.body 经 strip Markdown 后的纯文本 | text（分词，全文搜索） |
| `updated_at` | note.updated_at | datetime |

**媒体（Media）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"{folder_id}/{relative_path}"` | — |
| `_type` | `"media"` | keyword |
| `media_type` | `"image"` 或 `"video"` | keyword |
| `filename` | 从 relative_path 提取文件名 | text（分词） |
| `tags` | file.tags | keyword（多值） |
| `added_at` | file.added_at | datetime |

**笔记索引预处理**：笔记 body 索引前由后端 strip Markdown 语法标记（`#`、`[]`、` ``` ` 等），只保留纯文本内容。BuntDB 中存储原始 Markdown，Bleve 中存储过滤后的纯文本。

**时间排序**：各实体的 `updated_at`（媒体为 `added_at`）以 datetime 类型索引，支持搜索结果按时间排序。

### 搜索与筛选策略

搜索框输入和结构化筛选同时生效时取**交集（AND）**。

**搜索框（文本搜索）**：用户输入关键词，匹配所有 text 类型字段：

| 模块 | 搜索框匹配字段 |
|------|--------------|
| URL（站点 + 书签） | `title` + `description` + `domain_text` |
| 笔记 | `title` + `body` |
| 媒体 | `filename` |

`domain` 同时索引为 keyword（`domain`）和 text（`domain_text`）两种方式：搜索框走 `domain_text` 实现模糊匹配（输入"github"能命中 `github.com`），站点归组和精确过滤走 `domain` keyword。

**结构化筛选**：

| 筛选方式 | 走哪个字段 | 索引方式 |
|---------|-----------|---------|
| 标签多选 | `tags` | keyword 精确匹配（AND 语义） |
| 站点筛选 | BuntDB `idx:bm_site` | 精确匹配 site_id |
| 域名精确过滤 | `domain` | keyword |
| 类型筛选（媒体） | `media_type` | keyword |

`status` 字段（`"alive"` / `"dead"`）仅用于详情页状态展示，**不参与搜索、筛选或排序**，因此不纳入 Bleve 索引。

不支持 `前端::*` 前缀模糊查询。

**重建来源**：Bleve 索引可从 BuntDB (main.db) + 各 media_meta.json 全量重建，不含不可恢复的数据。

---

## 实体关系

```text
Site 1 ←——→ N Bookmark        (通过 bookmark.site_id 关联，书签必须归属站点)

Queue                          (独立，仅存 URL，不关联站点或书签)

Folder 1 ←——→ N Media         (通过 media_meta.json 内的 key)

Note                           (独立，无关联)

url_tag   ←——→ N Site/Bookmark (标签名内嵌于 tags 数组，注册表独立存储)
media_tag ←——→ N Media         (标签名内嵌于 file.tags 数组，注册表独立存储)
URL 标签与媒体标签完全独立，互不影响
```
