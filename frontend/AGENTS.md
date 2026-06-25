# 前端 AI 开发指引

编写前端代码时必须遵守本文件及 `docs/global/frontend-design-guide.md` 中的全部规则。

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
├── lib/              # 工具函数
└── types/            # 前端专用类型
```

- `bindings/` 由 Wails 自动生成，不手动编辑
- 路由用 Zustand store 管理当前页面，不用 React Router

## 代码风格

- 组件用函数式组件 + TypeScript
- 样式写在 className 中（Tailwind），不写单独 CSS 文件
- 可复用逻辑提取为自定义 hook，放在对应 feature 目录的 hooks.ts
- 跨模块组件放 `src/components/`，模块内组件放 `src/features/{模块}/`
- Zustand store 每模块独立，放 `src/stores/`
- 后端调用通过 `bindings/` 自动生成的函数，不手写 API 层

## UI 自查清单

每生成一个组件，逐条检查：

1. 是否使用了阴影？→ 只有浮层（下拉、弹窗）可以用阴影，其他用 `border`
2. 颜色是否超过灰阶 + 1 个强调色 + 1 个危险色？→ 多余的颜色删掉
3. 标签是否统一用 `Badge variant="secondary"`？→ 禁止多彩标签
4. 列表项是否被卡片包裹？→ 去掉 border + shadow，改用 `hover:bg-muted`
5. 字号是否在 text-xs 到 text-base 范围内？→ 用字重和灰度区分，不加大字号
6. 间距是否使用 gap-1/2、gap-3/4、gap-6/8 三档？→ 保持全局一致
7. 动效是否超过 200ms？→ 缩短到 150ms
8. 是否给每行文字都加了图标？→ 仅导航和操作按钮用图标
9. 是否有过多留白？→ 桌面工具应紧凑，不是手机 app
10. 图标是否为 lucide-react 线性风格？→ 禁止填充图标

## 组件示例

### 列表项

```tsx
<div className="px-3 py-2 cursor-pointer rounded-md hover:bg-muted">
  <div className="text-sm font-medium">标题</div>
  <div className="text-xs text-muted-foreground">次要信息</div>
</div>
```

### 标签

```tsx
<Badge variant="secondary">标签名</Badge>
```

### 区域分隔

```tsx
<div className="border-b my-4" />
```

### 操作按钮组

```tsx
<div className="flex justify-end gap-2">
  <Button variant="ghost">取消</Button>
  <Button>保存</Button>
</div>
```

### 空状态

```tsx
<div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
  <InboxIcon size={32} className="mb-2" />
  <p className="text-sm">暂无数据</p>
</div>
```
