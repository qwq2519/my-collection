# 前端 AI 开发指引

编写前端代码时必须遵守本文件中的铁律规则。

## 技术栈

| 职责 | 库 |
|------|-----|
| UI 组件 | shadcn/ui (Radix UI) |
| CSS | Tailwind CSS v4 |
| 状态管理 | Zustand（每模块独立 store） |
| 表单 | react-hook-form + zod |
| 图标 | lucide-react |
| Markdown 编辑器 | @uiw/react-md-editor |
| 虚拟滚动 | @tanstack/react-virtual |

## 目录约定

```
src/
├── components/ui/    # shadcn/ui 组件（CLI 生成，可修改）
├── components/       # 跨模块复用的业务组件
├── features/{模块}/  # 功能模块（组件 + hooks 同目录）
├── stores/           # Zustand stores，每模块独立
├── hooks/            # 全局通用 hooks
├── lib/              # 工具函数（callService, pick, str 等）
└── types/            # 前端专用类型
```

- `bindings/` 由 Wails 自动生成，不手动编辑
- 路由用 Zustand store 管理当前页面，不用 React Router
- 后端调用通过 `bindings/` 自动生成的函数，不手写 API 层

## 铁律（违反即报错）

1. **禁止 try-catch** — 用 `callService()` 返回 `[result, err]` 元组
2. **禁止三元运算符** — 用 `if + return` 或 `pick(cond, a, b)`
3. **禁止 any** — 写明确类型
4. **函数不超过 60 行，嵌套不超过 4 层**

## 文档路由

| 时机 | 文档 |
|------|------|
| 写代码前 | `docs/global/frontend-code-conventions.md` — 编码规范、工具函数、模式 |
| 做界面前 | `docs/global/frontend-design-guide.md` — UI/UX 设计规范 |
| 写完后自查 | `docs/global/frontend-review-checklist.md` — 代码 + UI 自查清单 |
| 概念不熟 | `docs/global/react-for-go-devs.md` — React 概念对照 Go 速查 |
