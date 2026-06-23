# 数据结构

集中描述所有存储层的数据结构，便于对比和实现时参照。

设计原则详见 [技术栈与存储](./tech-stack.md)，各功能的需求和行为规则见 `features/` 下对应文档。

## persist 目录结构

```text
persist/
├── main.db                    ← BuntDB 主存储（单文件）
├── search.bleve/              ← Bleve 搜索索引目录（可重建）
├── note-images/               ← 笔记中粘贴的图片
│   └── {note_id}/
│       └── {hash}.{ext}
└── image-folders/                   ← 图片管理数据（每文件夹独立）
    └── {folder_id}/
        ├── images_meta.json        ← 图片元数据
        ├── tree_hash.json          ← Merkle Tree 快照
        └── thumbnails/        ← 缩略图文件
            └── {hash}.{ext}
```

- `main.db` + `image-folders/*/images_meta.json` 是核心数据，丢失不可恢复
- `search.bleve/` 可从 main.db + images_meta.json 全量重建
- `thumbnails/` 可从源图片重新生成
- `note-images/` 是笔记引用的图片实体，删除则笔记中图片丢失

## BuntDB Key Schema

BuntDB 以 `前缀:id` 作为 key，value 为 JSON 字符串。

### 总览

| Key 格式 | 实体 | 说明 |
|----------|------|------|
| `site:{id}` | 站点 | 手动创建的网站 |
| `bm:{id}` | 书签 | 具体 URL，通过 `site_id` 字段关联站点 |
| `note:{id}` | 笔记 | Markdown 短笔记 |
| `folder:{id}` | 图片文件夹 | 注册表，实际图片数据在 JSON 文件中 |
| `url_tag:{name}` | URL 标签 | 标签注册表，`{name}` 为小写标签字符串 |
| `img_tag:{name}` | 图片标签 | 标签注册表，`{name}` 为小写标签字符串 |
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
  "icon": "github.png",
  "description": "代码托管平台",
  "created_at": "2026-06-22T10:00:00Z",
  "updated_at": "2026-06-22T10:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID，同 key 中的 `{site_id}` |
| title | string | 是 | 显示名称 |
| domain | string | 是 | 归一化后的完整域名（去 www），用于自动归组 |
| icon | string | 否 | 图标文件名（存储方式待定） |
| description | string | 否 | 站点描述 |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

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
  "domain": "github.com",
  "site_id": "a1b2c3d4-...",
  "title": "golang/go",
  "icon": "",
  "description": "Go 语言主仓库",
  "tags": ["Go", "开源"],
  "status": "alive",
  "created_at": "2026-06-22T12:00:00Z",
  "updated_at": "2026-06-22T12:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID |
| url | string | 是 | 完整 URL |
| domain | string | 是 | 从 URL 提取并归一化的域名 |
| site_id | string | 否 | 所属站点 ID；**空字符串表示在待归组队列中** |
| title | string | 是 | 页面标题 |
| icon | string | 否 | 页面图标 |
| description | string | 否 | 页面描述 |
| tags | string[] | 否 | 标签列表，独立于站点 |
| status | string | 否 | `"alive"` / `"dead"`，默认 `"alive"` |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

**查询模式：**

- 按 ID 精确查询：`bm:{id}`
- 按站点列出书签：BuntDB 自定义索引 `idx:bm_site`，索引 `site_id` 字段
- 按域名查找书签：BuntDB 自定义索引 `idx:bm_domain`，索引 `domain` 字段
- 待归组队列：查询 `site_id == ""` 的书签（通过索引 `idx:bm_site` 扫描空值）
- 标签精确/前缀匹配：BuntDB 自定义索引 `idx:bm_tags`
- 全文搜索 / 多标签组合：走 Bleve

**待归组队列说明：**

队列不使用独立 key 前缀，而是复用 `bm:` 前缀，以 `site_id` 为空来标识。当用户为队列中的书签创建或指定站点后，只需更新 `site_id` 字段，无需迁移 key。

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
| body | string | 否 | Markdown 正文，图片引用指向 `persist/note-images/` |
| created_at | string | 是 | ISO 8601 |
| updated_at | string | 是 | ISO 8601 |

**查询模式：**

- 按 ID 精确查询：`note:{id}`
- 列出所有笔记：前缀扫描 `note:*`
- 全文搜索（title + body）：走 Bleve

---

### folder — 图片文件夹注册表

```text
Key:   folder:{folder_id}
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "path": "D:\\Photos",
  "name": "照片",
  "added_at": "2026-06-18T08:00:00Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | UUID，即 folder_id |
| path | string | 是 | 文件夹绝对路径 |
| name | string | 是 | 显示名称 |
| added_at | string | 是 | ISO 8601 |

实际图片数据存储在 `persist/image-folders/{folder_id}/` 下的 JSON 文件中，不在 BuntDB 内。

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
| count | int | 是 | 关联书签数量，增删标签时同步维护 |
| created_at | string | 是 | ISO 8601，首次使用时自动创建 |

**查询模式：**

- 列出所有 URL 标签：前缀扫描 `url_tag:*`
- 按名称查找：`url_tag:{name}`

---

### img_tag — 图片标签注册表

```text
Key:   img_tag:{name}
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
| count | int | 是 | 关联图片数量，增删标签时同步维护 |
| created_at | string | 是 | ISO 8601，首次使用时自动创建 |

**查询模式：**

- 列出所有图片标签：前缀扫描 `img_tag:*`
- 按名称查找：`img_tag:{name}`

**标签规则**：大小写不敏感，存储时统一转小写。`::` 为逻辑层级分隔符，后端不解析层级关系。详见 [标签系统](./tag-system.md)。

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
| `idx:bm_site` | `bm:*` | `.site_id` | 按站点列出书签 / 查询待归组队列 |
| `idx:bm_domain` | `bm:*` | `.domain` | 按域名查找书签 |

---

## 每文件夹 JSON 文件

图片管理的元数据和变化检测快照以 JSON 文件存储在 `persist/image-folders/{folder_id}/` 下，独立于 BuntDB。

### images_meta.json — 图片元数据

```json
{
  "schema_version": 1,
  "folder_id": "550e8400-e29b-41d4-a716-446655440000",
  "images": {
    "photo1.jpg": {
      "tags": ["风景", "2026"],
      "thumbnail": "a3f2b8c1e5d7f9ab.jpg",
      "added_at": "2026-06-22T10:00:00Z",
      "file_size": 2048576,
      "dimensions": [1920, 1080]
    },
    "子目录A/img1.jpg": {
      "tags": ["人物::家人"],
      "thumbnail": "7e4d9f02ab31c8e5.jpg",
      "added_at": "2026-06-20T08:00:00Z",
      "file_size": 3145728,
      "dimensions": [4032, 3024]
    }
  }
}
```

| 字段 | 说明 |
|------|------|
| `schema_version` | 格式版本号，用于后续迁移 |
| `folder_id` | 冗余记录，与目录名一致 |
| `images` | Map，key 为相对路径，value 为图片元数据 |
| `images[*].tags` | 标签数组 |
| `images[*].thumbnail` | 缩略图文件名（`sha256(folder_id/rel_path)[:16].ext`） |
| `images[*].added_at` | 首次扫描发现的时间 |
| `images[*].file_size` | 文件大小（字节） |
| `images[*].dimensions` | `[width, height]` |

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
| `*.hash` | 文件：`hash(filename + mtime + size)`；目录：`hash(sorted(child_hashes))` |
| `*.dir_mtime` | 仅目录节点，用于 dir_mtime 剪枝优化 |
| `*.mtime` | 仅文件节点，文件修改时间 |
| `*.size` | 仅文件节点，文件大小 |
| `*.children` | 仅目录节点，Map，key 为文件/目录名 |

扫描时对比新旧 tree_hash.json 生成 diff，详见 [图片管理 - Merkle Tree 变化检测](../features/image-manager.md#merkle-tree-变化检测)。

---

## Bleve 索引结构

Bleve 索引目录：`persist/search.bleve/`

所有实体共用一个 Bleve 索引，通过 `_type` 字段区分实体类型。

### 文档结构

**书签（Bookmark）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"bm:{bm_id}"` | — |
| `_type` | `"bookmark"` | keyword |
| `title` | bookmark.title | text（分词） |
| `description` | bookmark.description | text（分词） |
| `domain` | bookmark.domain | keyword |
| `tags` | bookmark.tags | keyword（多值） |
| `url` | bookmark.url | keyword |

**笔记（Note）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"note:{note_id}"` | — |
| `_type` | `"note"` | keyword |
| `title` | note.title | text（分词） |
| `body` | note.body | text（分词，全文搜索） |

**图片（Image）：**

| Bleve 字段 | 来源 | 索引方式 |
|-----------|------|---------|
| `_id` | `"{folder_id}/{relative_path}"` | — |
| `_type` | `"image"` | keyword |
| `filename` | 从 relative_path 提取文件名 | text（分词） |
| `tags` | image.tags | keyword（多值） |

**标签筛选方式**：用户多选标签进行筛选时，查询走 Bleve keyword 精确匹配（AND 语义）。不支持 `前端::*` 前缀模糊查询。

**重建来源**：Bleve 索引可从 BuntDB (main.db) + 各 images_meta.json 全量重建，不含不可恢复的数据。

---

## 实体关系

```text
Site 1 ←——→ N Bookmark        (通过 bookmark.site_id 关联)
Bookmark (site_id="") = 待归组队列

Folder 1 ←——→ N Image         (通过 images_meta.json 内的 key)

Note                           (独立，无关联)

url_tag  ←——→ N Bookmark      (标签名内嵌于 bookmark.tags 数组，注册表独立存储)
img_tag  ←——→ N Image         (标签名内嵌于 image.tags 数组，注册表独立存储)
URL 标签与图片标签完全独立，互不影响
```
