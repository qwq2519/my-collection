# 图片管理

## 需求

### 管理方式

- **用户手动指定**要管理的图片文件夹（支持多个）
- 每个文件夹可包含多个子文件夹，组织粒度为"一个文件夹"
- 多个文件夹之间**不能存在父子嵌套关系**（如不能同时添加 `D:\Photos` 和 `D:\Photos\2026`）
- 添加后由后台**异步扫描处理**，不阻塞 UI
- **不在用户图片文件夹中写入任何文件**，避免污染用户目录

### 支持格式

JPG/JPEG、PNG、GIF、WebP、BMP。

### 文件变化检测

- **触发时机**：应用启动时 + 用户手动触发扫描
- **不做实时监听**（不使用 fsnotify）

### 文件删除处理

用户在文件系统中删除图片后，下次扫描时自动从元数据和搜索索引中移除。

### 数据规模

- 图片：万级（可能更多）

---

## 设计决策

### 设计目标

| 目标 | 说明 |
|------|------|
| 文件夹位置无关 | 移动文件夹后，元数据不受影响，只需更新路径映射 |
| 快速变化检测 | 无变化时零开销，有变化时只处理变化部分 |
| 不污染用户目录 | 不在用户图片文件夹中写入任何文件 |
| 可独立管理 | 添加/移除文件夹轻量，不影响其他文件夹 |

### 为什么图片元数据独立于 BuntDB

| 考量 | 全放 BuntDB | 每文件夹独立存储 |
|------|-----------|------------------|
| 移除文件夹 | 遍历删除所有 key | 直接删除目录 |
| 内存占用 | 全部常驻内存 | 当前全加载，后续可按需 |
| 文件夹移动 | 只改注册表 | 只改注册表 |
| 备份粒度 | 全在一个文件 | 可单独备份某文件夹 |
| 可调试性 | 需工具查看 | 直接打开 JSON |

---

## 实现方案

### 存储结构

每文件夹独立存储在 `persist/image-folders/{folder_id}/` 下，BuntDB 只存文件夹注册表。完整目录结构和 KV/JSON 格式详见 [数据结构](../global/data-structures.md#persist-目录结构)。

### 核心设计：folder_id + 相对路径

每张图片的唯一标识：

```text
folder_id     = 用户添加文件夹时生成的 UUID（永不改变）
relative_path = 从文件夹根目录算起的相对路径（如 "子目录A/img1.jpg"）
全局 ID       = "{folder_id}/{relative_path}"
```

文件夹移动时只需更新 BuntDB 中一条注册表记录，所有元数据文件无需改动。

### images_meta.json 格式

完整字段定义详见 [数据结构 - images_meta.json](../global/data-structures.md#images_metajson--图片元数据)。

写入策略："写临时文件 → rename"原子写入。内存策略：启动时全量加载。

> **扩展性备注**：当前方案下，单文件夹图片量过大（数千张以上）时 images_meta.json 文件会较大，每次修改需重写整个文件。当前数据规模可接受，后续若出现性能瓶颈再考虑分片或增量写入策略。

> **批量写入策略**：批量打标签等操作在内存中完成所有修改后，**一次性写入** images_meta.json + 批量更新 Bleve，避免逐张保存触发多次全量重写。

### 缩略图

命名规则：`sha256(folder_id + "/" + relative_path)[:16]` + 原始扩展名。

```text
示例：sha256("550e8400.../子目录A/img1.jpg")[:16] = "7e4d9f02ab31c8e5"
文件：persist/image-folders/{folder_id}/thumbnails/7e4d9f02ab31c8e5.jpg
```

images_meta.json 中保存缩略图文件名用于前端展示，原始相对路径作为 key 用于定位源文件。

生成时机：首次扫描 / 文件修改时重新生成 / 文件删除时同步删除。

### Merkle Tree 变化检测

**树结构：**

```text
           root_hash (dir_mtime)
          /         \
    dir_hash_A     dir_hash_B       ← 子目录（各自记录 dir_mtime）
    /    \           /    \
  f1     f2        f3     f4        ← 文件：hash(filename + mtime + size)
```

- 叶节点指纹：`hash(filename + mtime + size)`
- 目录节点指纹：`hash(sorted(child_hashes))`
- 每个目录节点额外记录 `dir_mtime`

**扫描算法（基于 dir_mtime 剪枝）：**

```text
scan(dir_path, cached_node):
  1. stat(dir_path) → 获取当前 dir_mtime
  2. 若 dir_mtime == cached_node.dir_mtime:
     → 整棵子树无变化，直接复用 cached_node（零 IO）
  3. 否则 → ReadDir 获取子项:
     - 子目录：递归 scan（可能在下层被剪枝）
     - 图片文件：获取 mtime + size → 计算叶节点 hash
  4. 汇总子节点 hash → 计算目录 hash
  5. 返回新节点
```

**IO 开销：**

| 场景 | 耗时 |
|------|------|
| 无变化 | < 1ms（1 次 stat） |
| 1 个子目录改了 1 张图 | < 5ms |
| 新增子目录含 100 张图 | < 20ms |
| 全量首次扫描（万级）| 100-300ms (SSD) |

**Diff 结果处理：**

| 变更类型 | 判定 | 处理 |
|---------|------|------|
| 新增 | 新树有、旧树无 | 生成缩略图 + 写 images_meta.json + 更新 Bleve |
| 删除 | 旧树有、新树无 | 删缩略图 + 从 images_meta.json 移除 + 更新 Bleve |
| 修改 | hash 不同 | 重新生成缩略图 + 更新 images_meta.json |
| 子目录重命名 | "A 消失 + B 出现"且 hash 相同 | 批量更新 relative_path + 重算缩略图名 |

**扁平目录分桶**（可选）：目录下 > 500 文件时按 100 个一组分桶，首版可不做。

### 搜索索引

Bleve 文档 ID = `{folder_id}/{relative_path}`，索引 tags 和 filename。可从 images_meta.json 全量重建。

### 文件夹生命周期

**添加：**

```text
1. 验证路径存在、可读、不与已有文件夹嵌套
2. 生成 folder_id (UUID)
3. 写入 main.db: folder:{folder_id}
4. 创建 persist/image-folders/{folder_id}/
5. 触发全量扫描
```

**移除：**

```text
1. 从 main.db 删除 folder:{folder_id}
2. 从 Bleve 删除该 folder_id 下所有文档
3. 删除 persist/image-folders/{folder_id}/ 整个目录
4. 释放内存
```

**移动（修改路径）：**

```text
1. 验证新路径存在且可读
2. 更新 main.db: folder:{folder_id}.path
3. 触发扫描（大概率无变更）
```

### 缓存失效与重建

| 场景 | 处理 |
|------|------|
| tree_hash.json 不存在/损坏 | 全量扫描重建 |
| images_meta.json 不存在/损坏 | 全量扫描重建（标签数据丢失） |
| thumbnails/ 缺失 | 从源图片重新生成 |
| schema_version 不匹配 | 格式迁移或全量重建 |
| 用户"强制重建" | 删除 `image-folders/{id}/` 目录后重新扫描 |

---

## UI

缩略图网格展示：

```text
🔍 [搜索...]  [文件夹筛选 ▼] [扫描]
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│      │ │      │ │      │ │      │
│  🖼  │ │  🖼  │ │  🖼  │ │  🖼  │
│      │ │      │ │      │ │      │
│photo1│ │photo2│ │photo3│ │photo4│
│#风景  │ │#美食  │ │#旅行  │ │#日落  │
└──────┘ └──────┘ └──────┘ └──────┘
```

- 点击图片展开详情面板（标签编辑、原图预览、文件信息）
- 文件夹筛选、手动扫描触发作为内容区顶部控件

### 标签编辑

**单张编辑：**

- 点击图片展开详情面板，面板中可直接增删标签
- 标签修改即时保存，无需手动点保存按钮

**批量打标签：**

- 支持多选/全选图片
- 选中后顶部出现批量操作栏，可为选中图片统一添加标签
- 批量添加的标签追加到每张图片的已有标签中（不覆盖）
