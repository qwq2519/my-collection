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

### 15. URL 去重的归一化规则不明确

添加 URL 时拒绝重复，但未定义 URL 归一化规则：

- 尾部斜杠：`github.com/go` vs `github.com/go/`
- 查询参数顺序：`?a=1&b=2` vs `?b=2&a=1`
- fragment（`#section`）是否去除
- 协议（http vs https）是否统一

需要在 [url-bookmarks.md](./features/url-bookmarks.md) 中补充。

### 16. 待归组书签缺少"分配到已有站点"的流程

待归组队列中用户可以"创建对应站点"或"删除"，但缺少将书签手动分配到已存在站点的能力。例如 `docs.github.com/xxx` 想归到 `github.com` 站点下，当前按完整域名匹配做不到。

需要在 [url-bookmarks.md](./features/url-bookmarks.md) 中补充手动分配站点的交互。

### 17. 标签名中特殊字符的处理规则未定义

标签用 `::` 作为层级分隔符，且标签名直接作为 BuntDB key（`url_tag:{name}`）。未定义：

- 标签名中允许哪些字符
- 标签名包含 `:` 时与 key 前缀分隔符的冲突处理
- 空格、特殊符号的处理规则

需要在 [tag-system.md](./global/tag-system.md) 中补充。
