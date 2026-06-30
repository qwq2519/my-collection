# 文档导航

## 功能文档

每个功能一个文件，从需求到方案到实现自包含。

| 文档 | 说明 |
|------|------|
| [URL 收藏](./features/url-bookmarks.md) | 站点、书签、域名匹配、存活检测 |
| [笔记](./features/notes.md) | Markdown 短笔记 |
| [媒体管理](./features/media-manager.md) | 图片+视频、文件夹扫描、Merkle Tree、元数据、缩略图、ffmpeg |
| [应用设置](./features/settings.md) | 数据目录、媒体文件夹管理、索引重建、ffmpeg 状态 |
| [小说管理](./features/novel.md) | **暂不实现**，本地小说导入、阅读器、进度管理 |
| [LLM 对话](./features/llm-chat.md) | **暂不实现**，大模型多轮对话、重新生成、隐藏轮次 |

## 全局文档

跨功能的选型、架构和约定。

| 文档 | 说明 |
|------|------|
| [技术栈与存储](./global/tech-stack.md) | Wails + BuntDB + Bleve 选型、前端技术栈、主题系统、persist 目录、事务策略、备份 |
| [数据结构](./global/data-structures.md) | BuntDB Key Schema、JSON 格式、Bleve 索引、persist 目录结构 |
| [整体布局](./global/ui-layout.md) | 侧边栏、导航、搜索框、列表排序 |
| [标签系统](./global/tag-system.md) | `::` 层级规则、生命周期、管理页面 |
| [开发指南](./global/dev-guide.md) | WSL + Windows 开发环境与构建流程 |

## 前端文档

按使用时机分为三类：写代码前参考、做界面前参考、写完后自查。

| 文档 | 定位 | 何时读 |
|------|------|--------|
| [前端代码开发规范](./global/frontend-code-conventions.md) | 编码规范、工具函数、Go 友好模式 | 写代码前 |
| [前端 UI/UX 设计规范](./global/frontend-design-guide.md) | 颜色、字体、间距、组件规范 | 做界面前 |
| [前端代码审查清单](./global/frontend-review-checklist.md) | 代码 + UI 逐条自查 | 写完后 |
| [React 概念速查](./global/react-for-go-devs.md) | React/Zustand 对照 Go 解释 | 遇到困惑时 |

## 待讨论

| 文档 | 说明 |
|------|------|
| [待讨论 & 待补充](./pending-plan.md) | 需进一步讨论或补充的设计点，完成后归入对应文档 |

## 约定

- 功能文档统一结构：需求 → 设计决策 → 实现方案 → UI
- 新增功能直接在 `features/` 下创建新文件
- 跨功能的全局决策放 `global/`
