# 待讨论 & 待补充

跟踪需要进一步讨论或后续补充的设计点。完成后移至对应文档或标记删除。

## 待讨论

### ~~1. 标签系统完善~~ → 已完成

已重写 [global/tag-system.md](./global/tag-system.md)，补充 [global/data-structures.md](./global/data-structures.md) 中的 `url_tag` / `img_tag` 注册表结构。

### ~~2. 数据结构文档~~ → 已完成

已创建 [global/data-structures.md](./global/data-structures.md)。

### ~~3. URL 元数据获取方式~~ → 已完成

已写入 [features/url-bookmarks.md](./features/url-bookmarks.md#元数据获取)。方案：表单旁"抓取"按钮，后端解析 HTML 返回 JSON，用户手动选择字段填入。

## 待补充（已有结论，需写入文档）

### 4. 应用启动流程

各文档分散提到了启动时的行为，需要集中整理：

- BuntDB 加载
- 检测 `index_dirty` 标志，按需重建 Bleve
- images_meta.json 全量加载到内存
- 是否自动触发图片文件夹扫描
- 异常恢复策略

### ~~5. 编辑功能~~ → 已完成

已补充到各功能文档：
- URL/站点：详情区"编辑"按钮 → 表单态 → 保存/取消
- 笔记：手动保存（Ctrl+S），切换时自动保存，详情区删除按钮
- 图片：点击展开详情编辑标签（即时保存），支持多选/全选批量打标签

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

- images_meta.json 已有 `schema_version`，BuntDB 数据也需要版本管理
- 应用升级后数据格式变化的迁移机制

### ~~9. 前端状态管理选型~~ → 已完成

已选定 Zustand，写入 [tech-stack.md](./global/tech-stack.md#技术选型)。每模块独立 store，职责隔离。

### 10. Service 接口概览

各 Service 的方法列表和大致职责，后续在实现过程中补充。约定见 [tech-stack.md - 接口约定](./global/tech-stack.md#接口约定)。
