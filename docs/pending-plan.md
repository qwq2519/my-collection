# 待讨论 & 待补充

跟踪需要进一步讨论或后续补充的设计点。完成后移至对应文档或标记删除。

## 待讨论

### ~~1. 标签系统完善~~ → 已完成

已重写 [global/tag-system.md](./global/tag-system.md)，补充 [global/data-structures.md](./global/data-structures.md) 中的 `url_tag` / `img_tag` 注册表结构。

### ~~2. 数据结构文档~~ → 已完成

已创建 [global/data-structures.md](./global/data-structures.md)。

### 3. URL 元数据获取方式

`url-bookmarks.md` 提到 icon、title、description 作为元数据，但未定义来源：

- 用户手动填写？
- 自动抓取（Open Graph / meta 标签）？
- 自动抓取失败时的降级策略？

## 待补充（已有结论，需写入文档）

### 4. 应用启动流程

各文档分散提到了启动时的行为，需要集中整理：

- BuntDB 加载
- 检测 `index_dirty` 标志，按需重建 Bleve
- images.json 全量加载到内存
- 是否自动触发图片文件夹扫描
- 异常恢复策略

### 5. 编辑功能

各功能文档只描述了"添加"和"删除"，以下编辑行为需补充：

- URL/站点：编辑 title、description、icon、tags
- 笔记：编辑保存策略（自动保存 vs 手动保存）
- 笔记：删除功能
- 图片：标签编辑流程

### 6. 错误处理与空状态 UX

- 图片文件夹不可访问时的提示
- 搜索无结果的空状态
- 首次使用的引导
- 加载中状态

### 7. 并发模型

Go 后端涉及的异步操作需明确：

- 图片扫描、缩略图生成、URL 存活检测的并发策略
- goroutine 并发数限制
- 扫描进行中的读写竞态处理

### 8. 数据迁移策略

- images.json 已有 `schema_version`，BuntDB 数据也需要版本管理
- 应用升级后数据格式变化的迁移机制
