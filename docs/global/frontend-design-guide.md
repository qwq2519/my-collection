# 前端设计指南

本项目的 UI 设计原则和开发规范。目标：**简洁、克制、信息密度高、工具感强**。

参考产品风格：Linear、Notion、Raycast、Things 3、Obsidian。

---

## 核心原则

1. **内容是主角** — UI 框架（侧边栏、搜索栏、按钮）应尽可能"隐形"，不争夺注意力
2. **做减法** — 能去掉的视觉元素就去掉，留下的每一个都有功能意义
3. **一致性** — 间距、字号、颜色、圆角在全局保持统一，不逐个组件微调

---

## 颜色

只用一个强调色，其余用灰阶。

| 用途 | 色彩策略 |
|------|---------|
| 背景 | 白 / 极浅灰 |
| 正文 | 深灰（不用纯黑） |
| 次要文字 | 中灰（`text-muted-foreground`） |
| 占位符 | 浅灰 |
| 强调色 | 1 个（按钮、选中态、链接），使用 shadcn `primary` |
| 危险操作 | 红色系（删除确认） |
| 边框 | 极浅灰（shadcn `border`，几乎看不见但能感知） |

**禁止**：

- 每个模块分配不同颜色
- 渐变背景
- 多彩标签（标签统一用 `Badge variant="secondary"`）
- 大面积饱和色块

---

## 字体与层级

用**字重和灰度**区分层级，不靠字号跨度。桌面工具应用字号集中在 12-16px。

| 场景 | 大小 | Tailwind class | 字重 |
|------|------|---------------|------|
| 列表项标题 / 正文 | 14px | `text-sm` | `font-medium`（标题）/ `font-normal`（正文） |
| 次要信息（时间、域名、标签数） | 12px | `text-xs` | `font-normal` + `text-muted-foreground` |
| 区域标题 | 16px | `text-base` | `font-semibold` |
| 页面大标题（极少用） | 18px | `text-lg` | `font-semibold` |

```tsx
// 好：字重 + 颜色区分主次
<div>
  <span className="text-sm font-medium">GitHub</span>
  <span className="text-xs text-muted-foreground ml-2">github.com</span>
</div>

// 差：字号跨度过大
<div>
  <span className="text-xl font-bold">GitHub</span>
  <span className="text-xs text-gray-400 ml-2">github.com</span>
</div>
```

---

## 间距

使用 Tailwind 4px 递增系统，全局只用 3-4 个间距档位：

| 档位 | 值 | Tailwind | 场景 |
|------|-----|---------|------|
| 紧凑 | 4-8px | `gap-1`, `gap-2` | 列表项内部、图标与文字 |
| 常规 | 12-16px | `gap-3`, `gap-4` | 区块内部元素间 |
| 宽松 | 24-32px | `gap-6`, `gap-8` | 区块之间、表单分组之间 |

**间距一致是"看起来舒服"的最大贡献因素。** 不要在不同地方随意用不同间距。

---

## 边框与阴影

**边框优于阴影。** 阴影只用在"浮起来"的元素上。

| 元素 | 样式 |
|------|------|
| 侧边栏与内容区分割 | `border-r` |
| 列表项之间 | `border-b` 或无边框（靠间距分隔） |
| 区域分割 | `border-b` 细线 |
| 下拉菜单、弹窗 | `shadow-md`（仅浮层） |
| 卡片 | 通常不需要，如必须则 `border` 无阴影 |

```tsx
// 好
<aside className="border-r h-full">

// 差
<aside className="shadow-lg h-full">
```

---

## 列表

列表项不用卡片包裹，用 hover 反馈区分交互区域：

```tsx
// 好：干净的列表项
<div className="px-3 py-2 cursor-pointer rounded-md hover:bg-muted">
  <div className="text-sm font-medium">GitHub</div>
  <div className="text-xs text-muted-foreground">3 个书签</div>
</div>

// 差：每条都是卡片
<div className="rounded-lg border shadow-sm p-4 mb-3">
  <div className="text-base font-bold">GitHub</div>
  <div className="text-sm text-gray-500">3 个书签</div>
</div>
```

选中态使用 `bg-muted` 或 `bg-accent`，不用左侧色条、高亮边框等装饰。

---

## 标签

所有标签统一样式，不按内容分配颜色：

```tsx
// 好：统一 secondary 样式
<Badge variant="secondary">go</Badge>
<Badge variant="secondary">开源</Badge>

// 差：每个标签一个颜色
<Badge className="bg-blue-100 text-blue-800">go</Badge>
<Badge className="bg-green-100 text-green-800">开源</Badge>
```

---

## 图标

- 使用 lucide-react 线性图标（outline 风格），不用填充图标
- 图标大小与文字匹配：正文旁 `size={16}`，按钮内 `size={14}`
- **不要给每行文字都加图标**，只在需要视觉锚点的地方使用（侧边栏导航、操作按钮）

---

## 动效

桌面工具追求即时响应感，动效要**快且微妙**：

| 交互 | 时长 | 说明 |
|------|------|------|
| hover 变色 | 150ms | `transition-colors duration-150` |
| 折叠/展开 | 150-200ms | shadcn Collapsible 默认 |
| 弹窗出现 | 150ms | fade + 微缩放 |
| 页面切换 | 0ms | 直接切换，无过渡动画 |

**禁止**：300ms+ 的缓动动画、元素进场动画、hover 放大效果。

---

## 组件规范

### 侧边栏

- 宽度 220-240px，折叠后约 48px
- 浅灰底色 + 右侧细 `border-r`，不用阴影
- 导航项：lucide 线性图标 + 文字，选中态 `bg-muted font-medium`
- 设置项用分割线与功能导航分开

### 详情区

- 不加外边框，靠留白与列表区分隔
- 信息按重要程度分层：标题 → 域名/次要信息 → 正文描述 → 标签 → 分割线 → 元信息（时间、状态）
- 操作按钮（编辑、删除）放在合理位置，不要浮动固定

### 媒体网格

- 缩略图间距统一 `gap-2`
- 鼠标悬停：微妙的叠加层或边框变化，不放大
- 文件名 `truncate` 截断
- 标签不在网格中显示，选中后在详情面板编辑

### 表单

- 表单项垂直排列，标签在输入框上方（`flex flex-col gap-2`）
- 必填标记用 `*`，不用红色感叹号
- 错误信息在对应输入框下方，`text-xs text-destructive`
- 保存/取消按钮右对齐，主操作用 `Button`，取消用 `Button variant="ghost"`

### 空状态

- 居中显示，一个灰色图标 + 一行说明文字
- 不要大面积插画或 emoji
- 有操作引导时加一个 ghost 或 outline 按钮

---

## 反模式清单

以下做法在本项目中**禁止使用**：

| 反模式 | 正确做法 |
|--------|---------|
| 每个列表项都包裹卡片（border + shadow + rounded） | 无框列表，hover 反馈 |
| 多彩标签 / 按模块分配颜色 | 统一灰阶标签，单一强调色 |
| 字号跨度超过 12-18px 范围 | 字重 + 灰度区分层级 |
| hover 时元素放大（scale） | hover 时背景色变化 |
| 300ms+ 的慢动画 | 150ms 快速过渡 |
| 大面积留白（手机 app 风格） | 紧凑信息密度（桌面工具风格） |
| 渐变背景、发光效果、装饰线条 | 纯色背景，极浅灰边框分隔 |
| 每行文字前都放图标 | 仅导航和操作按钮用图标 |
| 填充风格图标 | 线性（outline）图标 |
| 操作按钮用鲜艳颜色 | 默认色或 ghost，仅危险操作用红色 |
