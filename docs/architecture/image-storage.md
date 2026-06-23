# 图片存储架构

## 目的

定义图片管理模块的存储结构、文件变化检测机制和元数据管理方案，作为开发实现的依据。

## 读者

参与本项目开发的人。

## 设计目标

| 目标 | 说明 |
|------|------|
| 文件夹位置无关 | 用户移动图片文件夹后，元数据不受影响，只需更新路径映射 |
| 快速变化检测 | 无变化时零开销，有变化时只处理变化部分 |
| 不污染用户目录 | 不在用户图片文件夹中写入任何文件 |
| 可独立管理 | 添加/移除文件夹的操作轻量，不影响其他文件夹的数据 |
| 内存可控 | 图片元数据规模可能很大，不应全部常驻内存 |

---

## 存储结构

### 总体布局

```text
persist/
├── main.db                  ← BuntDB：URL、笔记、文件夹注册表
├── search.bleve/            ← 全局搜索索引（含图片标签索引）
└── folders/
    └── {folder_id}/
        ├── images.json      ← 该文件夹所有图片的元数据
        ├── tree.json        ← Merkle Tree 快照（上次扫描结果）
        └── thumbnails/      ← 该文件夹图片的缩略图
```

### 为什么图片元数据独立于 BuntDB

| 考量 | 全放 BuntDB | 每文件夹独立存储 |
|------|-----------|------------------|
| 移除文件夹 | 遍历删除该 folder_id 下所有 key | 直接删除 `folders/{id}/` 目录 |
| 内存占用 | 全部图片元数据常驻内存 | 当前方案全加载，后续可按需加载 |
| 文件夹移动 | 只改注册表一条 | 只改注册表一条 |
| 备份粒度 | 全在一个文件 | 可单独备份/恢复某文件夹的数据 |
| 可调试性 | 需工具查看 BuntDB 内容 | 直接打开 JSON 查看 |

BuntDB（main.db）只存储文件夹注册表等轻量数据：

```text
BuntDB Key 结构（图片相关）:
└── folder:{folder_id}  → { "path": "D:\\Photos", "name": "照片", "added_at": "..." }
```

---

## 核心设计：folder_id + 相对路径

### 图片标识

每张图片的唯一标识由两部分组成：

```text
folder_id    = 用户添加文件夹时生成的 UUID（永不改变）
relative_path = 从文件夹根目录算起的相对路径（如 "子目录A/img1.jpg"）
```

全局唯一 ID：`{folder_id}/{relative_path}`

### 文件夹移动处理

用户将文件夹从 `D:\Photos` 移动到 `E:\Backup\Photos` 时：

```text
1. 用户在设置中修改路径
2. 验证新路径存在且可读
3. 更新 main.db 中的 folder:{folder_id}.path
4. 触发扫描（大概率 tree hash 一致，零变更）
```

只修改一条注册表记录，所有 images.json、tree.json、缩略图均无需改动。

---

## 元数据文件：images.json

### 结构

```json
{
  "schema_version": 1,
  "folder_id": "550e8400-e29b-41d4-a716-446655440000",
  "images": {
    "photo1.jpg": {
      "tags": ["风景", "2026"],
      "thumbnail": "a3f2b8c1.jpg",
      "added_at": "2026-06-22T10:00:00Z",
      "file_size": 2048576,
      "dimensions": [1920, 1080]
    },
    "子目录A/img1.jpg": {
      "tags": ["人物::家人"],
      "thumbnail": "7e4d9f02.jpg",
      "added_at": "2026-06-20T08:00:00Z",
      "file_size": 3145728,
      "dimensions": [4032, 3024]
    }
  }
}
```

Key 是相对路径，与文件夹绝对位置无关。

### 写入策略

采用"写临时文件 → rename"原子写入：

```text
1. 将完整 JSON 写入 images.json.tmp
2. os.Rename("images.json.tmp", "images.json")
3. rename 是原子操作，崩溃安全
```

### 数据规模估算

| 规模 | 文件大小 | SSD 重写耗时 |
|------|---------|-------------|
| 1000 张 | ~300 KB | < 2ms |
| 3000 张 | ~900 KB | < 5ms |
| 10000 张 | ~3 MB | < 10ms |

### 内存加载策略

当前方案：**应用启动时全量加载所有文件夹的 images.json 到内存**，便于搜索和统计计算。

后续优化方向：按需加载（只加载用户正在浏览的文件夹），配合虚拟滚动减少内存占用。

---

## 缩略图

### 命名规则

缩略图文件名 = `hash(folder_id + "/" + relative_path)` 的前 16 位十六进制 + 原始扩展名。

```text
示例：
  folder_id = "550e8400..."
  relative_path = "子目录A/img1.jpg"
  thumbnail = sha256("550e8400.../子目录A/img1.jpg")[:16] + ".jpg"
           = "7e4d9f02ab31c8e5.jpg"
```

images.json 中同时保存缩略图文件名，供前端直接使用。原始相对路径保留在 images.json 的 key 中，用于展示和定位源文件。

### 存储位置

每个文件夹的缩略图存放在各自目录下：

```text
persist/folders/{folder_id}/thumbnails/7e4d9f02ab31c8e5.jpg
```

### 生成时机

- 新图片首次扫描时生成
- 图片文件被修改时（检测到 mtime/size 变化）重新生成
- 图片被删除时同步删除对应缩略图

---

## 文件变化检测：Merkle Tree

### 触发时机

- 应用启动时
- 用户手动点击"扫描"

不做实时监听（不使用 fsnotify）。

### 树结构

按目录层级构建，叶节点为图片文件，内部节点为目录：

```text
           root_hash (dir_mtime)
          /         \
    dir_hash_A     dir_hash_B       ← 子目录级（各自记录 dir_mtime）
    /    \           /    \
  f1     f2        f3     f4        ← 文件级
```

- 叶节点指纹：`hash(filename + mtime + size)`
- 目录节点指纹：`hash(sorted(child_hashes))`
- 每个目录节点额外记录 `dir_mtime`（目录本身的修改时间）

### tree.json 格式

```json
{
  "schema_version": 1,
  "built_at": "2026-06-22T10:00:00Z",
  "root": {
    "hash": "abc123...",
    "dir_mtime": 1719014400,
    "children": {
      "子目录A": {
        "hash": "def456...",
        "dir_mtime": 1719010000,
        "children": {
          "img1.jpg": { "hash": "...", "mtime": 1719010000, "size": 3145728 },
          "img2.jpg": { "hash": "...", "mtime": 1719008000, "size": 2048576 }
        }
      },
      "photo1.jpg": { "hash": "...", "mtime": 1719014400, "size": 2048576 },
      "photo2.png": { "hash": "...", "mtime": 1719012000, "size": 1048576 }
    }
  }
}
```

### 扫描算法：基于 dir_mtime 剪枝

核心原理：**操作系统保证当目录内的文件被新增、删除、重命名时，该目录的 mtime 会更新。**

利用这一点，可以跳过未变化的整棵子树，避免全量 stat：

```text
scan(dir_path, cached_node):
  1. stat(dir_path) → 获取当前 dir_mtime
  2. 若 cached_node 存在 且 dir_mtime == cached_node.dir_mtime:
     → 整棵子树无变化，直接返回 cached_node（零 IO）
  3. 否则 dir_mtime 变了 → ReadDir(dir_path) 获取子项列表:
     - 子目录：递归 scan(子目录, cached_node.children[子目录名])
       → 子目录仍可能在下一层被剪枝
     - 图片文件：entry.Info() 获取 mtime + size → 计算叶节点 hash
  4. 汇总所有子节点 hash → 计算当前目录 hash
  5. 返回新节点
```

### 各场景 IO 开销

| 场景 | IO 操作 | 耗时预估 |
|------|---------|---------|
| 什么都没变 | 1 次 stat（根目录）| < 1ms |
| 1 个子目录里改了 1 张图 | stat 根 + ReadDir 根 + stat 子目录 + ReadDir 子目录 | < 5ms |
| 新增 1 个子目录（含 100 张图）| stat 根 + ReadDir 根 + 全量扫描新子目录 | < 20ms |
| 全量首次扫描（万级文件）| 全量 ReadDir + stat | 100-300ms (SSD) |

### 扫描结果比较（Diff）

拿到新树后，与旧 tree.json 对比，产出变更集：

| 变更类型 | 判定条件 | 处理 |
|---------|---------|------|
| 新增文件 | 新树有、旧树无 | 生成缩略图，写入 images.json，更新 Bleve |
| 删除文件 | 旧树有、新树无 | 删除缩略图，从 images.json 移除，更新 Bleve |
| 修改文件 | 两边都有但 hash 不同 | 重新生成缩略图，更新 images.json 中的 file_size 等 |
| 子目录重命名 | 旧树"A 消失 + B 出现"且 A.hash == B.hash | 更新 images.json 中受影响图片的 relative_path |

### 子目录重命名检测

```text
旧树: 子目录A (hash=xxx) 包含 [img1, img2, img3]
新树: 子目录B (hash=xxx) 包含 [img1, img2, img3]

判定：hash 相同 → 是重命名，不是删除+新增
处理：批量更新 images.json 中 "子目录A/..." → "子目录B/..."
      缩略图文件名需要重新计算（因为 relative_path 变了）
```

### 扁平目录分桶（可选优化）

当某个目录下直接包含超过 500 个图片文件且无子目录时，Merkle Tree 退化为二层结构，失去剪枝能力。

此时可对文件列表按排序后的固定大小分桶，人为创建中间层：

```text
root_hash
├── bucket_0 (第 1~100 个文件的聚合 hash)
├── bucket_1 (第 101~200 个文件的聚合 hash)
...
└── bucket_N
```

这样即使目录 mtime 变了，也只需定位到变化的 bucket 再比较 100 个文件，而非全量比较。

**实现时机**：首版可不做，当检测到纯扁平目录文件数 > 500 时再启用。

---

## 缓存失效与重建

| 场景 | 处理 |
|------|------|
| tree.json 不存在或损坏 | 全量扫描文件系统，重建 tree.json |
| images.json 不存在或损坏 | 全量扫描，重建元数据（标签数据丢失，需用户重新标记） |
| thumbnails/ 缺失 | 从源图片重新生成，不影响元数据 |
| schema_version 不匹配 | 触发格式迁移或全量重建 |
| 用户手动"强制重建" | 删除该文件夹的整个 `folders/{id}/` 目录，重新全量扫描 |

---

## 搜索索引（Bleve）集成

图片标签搜索走全局 Bleve 索引，文档 ID 使用全局唯一标识：

```text
Bleve 文档:
  ID:       "{folder_id}/{relative_path}"
  tags:     ["人物", "人物::家人"]
  filename: "img1.jpg"
```

Bleve 索引是派生数据，可从所有 `folders/{id}/images.json` 全量重建。

写入时序：

```text
1. 更新 images.json（原子写入）
2. 更新 Bleve 索引
3. 若 Bleve 更新失败 → 数据已持久化，标记需重建索引，不影响核心数据
```

---

## 完整扫描流程

```text
应用启动 / 用户点击扫描
        │
        ▼
遍历文件夹注册表（main.db 中所有 folder:* 记录）
        │
        ▼ 对每个文件夹：
读取 tree.json（旧快照）
        │
        ├── 不存在 → 全量扫描，构建完整树
        │
        └── 存在 → 基于 dir_mtime 剪枝扫描，构建新树
                    │
                    ▼
              新树 vs 旧树 Diff
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
    新增文件     删除文件     修改文件
        │           │           │
        ▼           ▼           ▼
  生成缩略图    删除缩略图   重新生成缩略图
  更新 images.json（原子写入）
  更新 Bleve 索引
        │
        ▼
  保存新的 tree.json（原子写入）
```

---

## 文件夹生命周期

### 添加文件夹

```text
1. 验证路径存在、可读、不与已有文件夹嵌套
2. 生成 folder_id (UUID)
3. 写入 main.db: folder:{folder_id} → { path, name, added_at }
4. 创建 persist/folders/{folder_id}/ 目录
5. 触发全量扫描
```

### 移除文件夹

```text
1. 从 main.db 删除 folder:{folder_id}
2. 从 Bleve 中删除该 folder_id 下所有文档
3. 删除 persist/folders/{folder_id}/ 整个目录（含 images.json、tree.json、thumbnails/）
4. 释放内存中该文件夹的数据
```

### 移动文件夹（用户修改路径）

```text
1. 验证新路径存在且可读
2. 更新 main.db: folder:{folder_id}.path = 新路径
3. 触发扫描（利用已有 tree.json 对比，大概率无变更）
```

---

## 相关文档

- [功能需求 § 图片管理](../product/requirements.md#3-图片管理)
- [存储技术栈决策](./storage-decision.md)
- [文档索引](../INDEX.md)
