# AI 开发指引

个人资料收藏夹桌面应用。技术栈：Wails v3 + Go + React + TypeScript + Vite。

## 文档

修改功能前先阅读 `docs/` 下对应文档：

- `docs/doc-index.md` — 所有文档入口
- `docs/global/tech-stack.md` — 技术选型、接口约定、存储架构
- `docs/global/data-structures.md` — BuntDB Key Schema、Bleve 索引、persist 目录结构
- `docs/global/ui-layout.md` — 整体布局、搜索筛选、列表渲染
- `docs/global/tag-system.md` — 标签规则、生命周期、管理操作
- `docs/features/` — 各功能模块的需求与实现方案

## 前端

前端代码在 `frontend/` 目录。前端开发规范见 `frontend/AGENTS.md`。

## 后端

后端代码在 `internal/` 目录。后端开发规范见 `internal/AGENTS.md`。
