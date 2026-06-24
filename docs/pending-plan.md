# 待讨论 & 待补充

跟踪需要进一步讨论或后续补充的设计点。完成后移至对应文档或标记删除。

## 待讨论

### ~~1. 标签系统完善~~ → 已完成

已重写 [global/tag-system.md](./global/tag-system.md)，补充 [global/data-structures.md](./global/data-structures.md) 中的 `url_tag` / `media_tag` 注册表结构。

### ~~2. 数据结构文档~~ → 已完成

已创建 [global/data-structures.md](./global/data-structures.md)。

### ~~3. URL 元数据获取方式~~ → 已完成

已写入 [features/url-bookmarks.md](./features/url-bookmarks.md#元数据获取)。方案：表单旁"抓取"按钮，后端解析 HTML 返回 JSON，用户手动选择字段填入。

## 待补充（已有结论，需写入文档）

### 4. 应用启动流程

各文档分散提到了启动时的行为，需要集中整理：

- BuntDB 加载
- 检测 `index_dirty` 标志，按需重建 Bleve
- media_meta.json 全量加载到内存
- 是否自动触发媒体文件夹扫描
- 异常恢复策略

### ~~5. 编辑功能~~ → 已完成

已补充到各功能文档：
- URL/站点：详情区"编辑"按钮 → 表单态 → 保存/取消
- 笔记：手动保存（Ctrl+S），切换时自动保存，详情区删除按钮
- 图片：点击展开详情编辑标签（即时保存），支持多选/全选批量打标签

### 6. 错误处理与空状态 UX

- 媒体文件夹不可访问时的提示
- 搜索无结果的空状态
- 首次使用的引导
- 加载中状态

### 7. 并发模型

Go 后端涉及的异步操作需明确：

- 媒体扫描、缩略图生成、URL 存活检测的并发策略
- goroutine 并发数限制
- 扫描进行中的读写竞态处理

### 8. 数据迁移策略

- media_meta.json 已有 `schema_version`，BuntDB 数据也需要版本管理
- 应用升级后数据格式变化的迁移机制

### ~~9. 前端状态管理选型~~ → 已完成

已选定 Zustand，写入 [tech-stack.md](./global/tech-stack.md#技术选型)。每模块独立 store，职责隔离。

### ~~10. Service 接口概览~~ → 已完成

已写入 [tech-stack.md - 接口约定](./global/tech-stack.md#接口约定)。包含 Service 列表、返回模式、分页结构、Events 事件、HTTP 扩展方案。

## 设计问题（待讨论）

### ~~11. 中文全文搜索缺少分词器方案~~ → 已完成

已选定 gse（`github.com/go-ego/gse`），纯 Go 实现，无 CGO 依赖。写入 [tech-stack.md](./global/tech-stack.md#技术选型) 和 [data-structures.md](./global/data-structures.md#中文分词)。

### ~~12. 标签 count 在实体删除时的维护未说明~~ → 已完成

已补充到 [tag-system.md - 实体删除时的标签维护](./global/tag-system.md#实体删除时的标签维护)、[url-bookmarks.md - 删除](./features/url-bookmarks.md#删除)、[media-manager.md - Diff 结果处理 & 文件夹移除](./features/media-manager.md)。

### ~~13. 笔记图片与静态资源的前端访问路径未定义~~ → 已完成

已选定 Wails AssetHandler 方案，含候选方案对比。写入 [tech-stack.md - 静态资源访问](./global/tech-stack.md#静态资源访问)。

### 14. 图标存储的命名冲突

`persist/icons/` 下图标以 `{domain}.{ext}` 命名，站点和书签共享同一目录。同一域名下，站点图标和特定页面图标可能不同，当前设计会互相覆盖。

需要明确：只保留域名级图标（站点和书签共用），还是按实体 ID 分别存储。

### ~~15. URL 去重的归一化规则不明确~~ → 已完成

新增 `normalized_url` 字段和 `idx:bm_normalized_url` 索引，归一化规则写入 [data-structures.md - bm](./global/data-structures.md#bm--书签url)。后端提供 `NormalizeURL` 方法，前端调用后展示结果供用户确认。

### ~~16. 临时队列（原待归组书签）缺少"分配到已有站点"的流程~~ → 已完成

已简化设计：临时队列定位为临时备忘，只支持查看和删除，不做站点分配。用户如需正式收藏，手动创建对应站点和书签。写入 [url-bookmarks.md - 临时队列](./features/url-bookmarks.md#临时队列) 和 [data-structures.md - bm](./global/data-structures.md#bm--书签url)。

### ~~17. 标签名中特殊字符的处理规则未定义~~ → 已完成

已定义标签名字符规则（允许/禁止字符表），写入 [data-structures.md - 标签名字符规则](./global/data-structures.md#媒体标签注册表) 和 [tag-system.md - 基本规则](./global/tag-system.md#基本规则)。
