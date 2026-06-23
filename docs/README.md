# 文档导航

## 功能文档

每个功能一个文件，从需求到方案到实现自包含。

| 文档 | 说明 |
|------|------|
| [URL 收藏](./features/url-bookmarks.md) | 站点、书签、待归组队列、域名匹配、存活检测 |
| [笔记](./features/notes.md) | Markdown 短笔记 |
| [图片管理](./features/image-manager.md) | 文件夹扫描、Merkle Tree、元数据、缩略图 |

## 全局文档

跨功能的选型、架构和约定。

| 文档 | 说明 |
|------|------|
| [技术栈与存储](./global/tech-stack.md) | Wails + BuntDB + Bleve 选型、persist 目录、事务策略、备份 |
| [整体布局](./global/ui-layout.md) | 侧边栏、导航、搜索框、列表排序 |
| [标签系统](./global/tag-system.md) | `::` 层级规则、生命周期、管理页面 |
| [开发指南](./global/dev-guide.md) | WSL + Windows 开发环境与构建流程 |

## 待讨论

| 文档 | 说明 |
|------|------|
| [待讨论 & 待补充](./pending-plan.md) | 需进一步讨论或补充的设计点，完成后归入对应文档 |

## 约定

- 功能文档统一结构：需求 → 设计决策 → 实现方案 → UI
- 新增功能直接在 `features/` 下创建新文件
- 跨功能的全局决策放 `global/`
