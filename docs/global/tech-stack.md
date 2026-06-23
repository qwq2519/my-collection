# 技术栈与存储

## 技术选型

| 层 | 选择 | 理由 |
|----|------|------|
| 框架 | Wails v3 | Go + Web 前端，桌面应用，纯 Go 无 CGO |
| 后端 | Go 1.25 | 高性能、编译简单、单二进制部署 |
| 前端 | React + TypeScript + Vite | 生态成熟、类型安全、构建快 |
| 主存储 | BuntDB | 内存 KV，自动持久化，纯 Go |
| 搜索 | Bleve v2 | 全文搜索引擎，倒排索引，纯 Go |
| 目标平台 | Windows 桌面 | 纯本地单机，不考虑多设备同步 |

## 存储架构

```text
┌─────────────────────────────────────────────────────────┐
│                   Go 应用 (Wails v3)                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  BuntDB (main.db)          每文件夹 JSON         Bleve  │
│  - URL/笔记/站点/队列     - 图片标签与元数据    - 搜索索引│
│  - 文件夹注册表           - Merkle Tree 快照    - 可重建  │
│  - 内存常驻               - 启动时全量加载               │
│  - 自动持久化到单文件      - 原子写入 JSON 文件           │
│                                                         │
│  persist/main.db     persist/image-folders/{id}/  search.bleve/│
└─────────────────────────────────────────────────────────┘
```

| 组件 | 包 | 定位 |
|------|----|------|
| BuntDB | `github.com/tidwall/buntdb` | 主存储，内存 KV，自动持久化 |
| Bleve v2 | `github.com/blevesearch/bleve/v2` | 全文搜索，倒排索引 |

两者均为纯 Go 实现，无 CGO 依赖。

## 为什么选 BuntDB + Bleve

### 需求约束

| 需求 | 说明 |
|------|------|
| 高性能读 | 收藏查询、标签检索需要毫秒级响应 |
| 减少磁盘 IO | 避免频繁读写磁盘，尽量内存操作 |
| 标签前缀匹配 | `前端::React` 查询时能匹配所有 `前端::*` |
| 灵活搜索 | 图片标签细粒度，接近全文检索 |
| 笔记内容搜索 | 支持对 note 正文全文搜索 |
| 部署简单 | 桌面应用，不依赖外部服务 |

### 候选方案对比

| 方案 | 类型 | 特点 |
|------|------|------|
| SQLite + FTS5 | 嵌入式关系数据库 | 生态成熟、SQL 查询、FTS5 全文搜索 |
| **BuntDB + Bleve** | 内存 KV + 搜索引擎 | 纯 Go、内存快查、倒排索引 |
| Badger + Bleve | 磁盘 KV + 搜索引擎 | 纯 Go、大数据量、LSM-tree |
| go-memdb + JSON 刷盘 | 内存数据库 + 手动持久化 | 极快、多索引、需自己处理刷盘 |

**不选 SQLite**：Go 调用需 CGO（或纯 Go 翻译版性能降 20-30%），增加编译复杂度；SQL 表达力对本项目超配。

**不选 Badger**：数据量万级以下，全内存方案更优；LSM-tree 有写放大。

### KV/JSON vs 关系型

**获得的优势：**

| 方面 | 说明 |
|------|------|
| 无需 migration | 加字段直接写入 |
| 半结构化友好 | 不同实体可有不同字段 |
| 性能可预测 | 无 SQL 解析开销，内存直读 |
| 部署简单 | 一个文件就是整个数据库 |
| 和 Go struct 对应 | JSON ↔ struct 直接序列化 |

**承担的代价：**

| 方面 | 应对 |
|------|------|
| 多条件查询要自己写 | 提前设计索引，复杂查询走 Bleve |
| 关联查询是手动的 | 用 key 前缀模拟层级关系 |
| 数据完整性靠代码 | Store 层封装校验逻辑 |
| 无标准查询语言 | 抽象 Repository 接口隔离存储细节 |

**为什么可接受**：实体类型少（3 种），查询模式有限且可预知，标签用 JSON 数组比关联表更自然，数据量不大。

## persist 目录

所有持久化数据集中存放，是唯一需要备份的目录。完整目录结构和各文件格式详见 [数据结构](./data-structures.md#persist-目录结构)。

**备份方式**：直接拷贝整个 `persist/` 文件夹。

- `main.db` + `image-folders/*/images_meta.json` 是核心数据
- `search.bleve/` 可从上述数据重建
- `thumbnails/` 可通过重新扫描重建

## 职责分工

| 操作 | 走哪个组件 |
|------|-----------|
| 增删改收藏/笔记 | BuntDB → 同步更新 Bleve |
| 待归组队列读写 | BuntDB |
| 批量删除 URL | BuntDB 事务 + 清理 Bleve |
| 按 ID 精确查询 | BuntDB |
| 标签筛选（多选精确匹配） | Bleve keyword 精确匹配 |
| 标签注册表（列表、count） | BuntDB 前缀扫描 `url_tag:*` / `img_tag:*` |
| 笔记全文搜索 | Bleve |
| 按站点列出所有收藏 | BuntDB 自定义索引 `idx:bm_site` |
| 文件夹注册表 | BuntDB |
| 图片元数据增删改 | 每文件夹 images_meta.json → 同步更新 Bleve |
| 图片标签搜索 | Bleve |
| 图片文件变化检测 | 每文件夹 tree_hash.json |

## 写入事务策略

BuntDB 和 Bleve 不具备跨组件原子性。策略：**BuntDB 先写先提交，Bleve 后写；Bleve 失败时不回滚 BuntDB，而是标记索引脏并通知前端。**

```text
Store.Create(entity):
  1. BuntDB.Update(tx) → 写入 KV（提交）
  2. Bleve.Index(doc)  → 更新索引
  3. 若 Bleve 失败 → BuntDB 数据已持久化，标记 index_dirty，返回结果 + 索引异常警告
```

| 场景 | 处理 |
|------|------|
| BuntDB 写入失败 | 事务回滚，Bleve 不执行，返回错误 |
| BuntDB 成功、Bleve 失败 | 数据已写入，标记 `index_dirty`，返回成功 + 索引异常警告；前端展示"索引异常"状态 |
| 应用崩溃 | BuntDB AOF 保证持久化，启动时检测 `index_dirty` 标志自动重建 Bleve |

**索引脏标志（`index_dirty`）**：

- 存储在 BuntDB 中：`meta:index_dirty → true`
- Bleve 写入失败时置为 `true`
- 前端可查询该标志，在 UI 顶部展示"搜索索引异常，建议重建"提示
- 用户可在设置页手动触发重建；应用启动时若检测到该标志也自动重建
- 重建完成后清除标志

**图片元数据写入**（独立于 BuntDB 事务）：

```text
ImageStore.Update(folder_id, changes):
  1. 更新内存中的 images 数据
  2. 原子写入 images_meta.json (write tmp → rename)
  3. Bleve.Index(docs) → 更新索引
  4. 若 Bleve 失败 → images_meta.json 已持久化，标记 index_dirty
```

## 架构约束

1. **Store 层封装写入**：每次写入先完成 BuntDB 持久化，再同步更新 Bleve；Bleve 失败不阻塞写入
2. **Repository 接口隔离**：上层不直接依赖 BuntDB/Bleve API
3. **Bleve 可重建**：从 main.db + images_meta.json 全量灌入，启动时自动检测 `index_dirty` 触发重建
4. **索引状态可观测**：`index_dirty` 标志暴露给前端，用户可感知并手动触发重建
5. **数据格式 JSON**：统一使用 JSON，便于调试和导出
6. **图片数据隔离**：按文件夹独立存储，不与 URL/笔记耦合

## 部署与扩展

- **纯本地单机**桌面应用
- 后续可选：开放 HTTP 接口，支持从手机 share URL 到本机
- 始终单机部署，不考虑多设备同步

## 暂不考虑

- 浏览器书签导入/导出
- 全局快捷键唤起
- 多设备同步
