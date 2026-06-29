# 前端代码规范化改造计划

> 目标：让前端代码对 Go 后端开发者友好 — 线性流程、少嵌套、少魔法、机器可检查。

## 背景

当前前端代码的四个痛点：

1. **try-catch 样板泛滥** — 12 处重复的 try-catch-finally，部分删除操作甚至缺失错误处理
2. **三元运算符嵌套** — JSX 中 `a ? x : b ? y : z` 链式写法可读性差
3. **函数开头大量 const** — Zustand selector 每个字段一行，表单 useState 七八个
4. **回调函数层层透传** — `onSave / onCancel / onChange` 多层传递，不符合 Go 直接调用的习惯

## 改造原则

- 用 `if + return` 替代三元运算符（Go 没有三元运算符）
- 用 `[result, err]` 元组替代 try-catch（类比 Go 的 `result, err := fn()`）
- `?.` `??` 收敛到 `lib/` 工具函数，业务代码不直接写
- 子组件直接调 store action，减少回调 prop 透传
- ESLint 强制执行语法约束，Prettier 统一格式

---

## 计划总览

| 阶段 | 内容 | 状态 |
|------|------|------|
| [P0](#p0安装-eslint--prettier) | 安装 ESLint + Prettier | 待执行 |
| [P1](#p1创建工具函数) | 创建工具函数 `lib/async.ts` + `lib/safe.ts` | 待执行 |
| [P2](#p2消灭-try-catch) | 全部 try-catch 改为 `callService` | 待执行 |
| [P3](#p3消灭三元运算符) | 全部三元运算符改为 `if + return` 或 `pick()` | 待执行 |
| [P4](#p4收敛--) | `?.` `??` 收敛到工具函数 | 待执行 |
| [P5](#p5减少回调和-const-行数) | 减少回调透传 + selector 预组合 | 待执行 |
| [P6](#p6更新规范文档) | 更新 `AGENTS.md` 和本文档 | 待执行 |

---

## P0：安装 ESLint + Prettier

### P0.1 安装依赖

```bash
cd frontend
npm install -D eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-config-prettier prettier
```

### P0.2 创建 Prettier 配置

新建 `frontend/.prettierrc`：

```jsonc
{
  // 语句末尾不加分号：const x = 1（不写 const x = 1;）
  "semi": false,
  // 字符串用双引号："hello"（不写 'hello'）
  "singleQuote": false,
  // 最后一项后面也加逗号（和 Go struct 初始化一致，减少 git diff 噪音）
  "trailingComma": "all",
  // 一行最多 100 字符，超了自动换行
  "printWidth": 100,
  // 缩进用 2 个空格（前端惯例，Go 用 tab）
  "tabWidth": 2,
  // 箭头函数参数总加括号：(x) => x + 1（不写 x => x + 1，更一致）
  "arrowParens": "always",
  // 换行符用 \n（Linux/Mac 风格，不用 Windows 的 \r\n）
  "endOfLine": "lf"
}
```

### P0.3 创建 ESLint 配置

新建 `frontend/eslint.config.js`，完整规则：

```javascript
import js from "@eslint/js"              // JavaScript 基础规则集
import ts from "typescript-eslint"        // TypeScript 专用规则集
import reactHooks from "eslint-plugin-react-hooks"  // React Hooks 使用规则
import prettier from "eslint-config-prettier"        // 关掉和 Prettier 冲突的格式规则

// 规则从上到下叠加，后面的覆盖前面的（类似 golangci-lint 的配置继承）
export default ts.config(
  js.configs.recommended,       // 第 1 层：JavaScript 社区推荐规则
  ...ts.configs.recommended,    // 第 2 层：TypeScript 社区推荐规则
  prettier,                     // 第 3 层：关掉格式类规则（格式交给 Prettier，不要 ESLint 管）
  {
    plugins: {
      "react-hooks": reactHooks,
    },
    // 规则级别：
    //   "error" = 红色报错，lint 不通过（类比 Go 编译错误）
    //   "warn"  = 黄色警告，lint 仍通过（类比 go vet 建议）
    //   "off"   = 关闭
    rules: {
      // ── Go 风格约束 ──

      // 禁止三元运算符 a ? b : c（Go 没有三元运算符，用 if-else）
      "no-ternary": "error",

      // no-restricted-syntax: 万能禁令，用 AST 选择器匹配任意语法结构
      // 类比 golangci-lint 的 forbidigo（禁止调用特定函数）
      "no-restricted-syntax": ["error",
        {
          // 匹配 try { } catch { } 语句 → 写了 try 就报错
          selector: "TryStatement",
          message: "禁止 try-catch，用 callService() 返回 [result, err] 元组，见 lib/async.ts",
        },
        {
          // 匹配 useCallback(...) 函数调用 → 调了就报错
          selector: "CallExpression[callee.name='useCallback']",
          message: "禁止 useCallback，直接写普通函数",
        },
        {
          // 匹配 useMemo(...) 函数调用 → 调了就报错
          selector: "CallExpression[callee.name='useMemo']",
          message: "禁止 useMemo，除非有实测性能问题（加 eslint-disable 并注释原因）",
        },
        {
          // 匹配 JSX 属性中带 {} 大括号体的箭头函数
          // 报错：onClick={() => { 多行代码 }}
          // 通过：onClick={() => setOpen(true)}  （单表达式没有 {}）
          selector: "JSXAttribute ArrowFunctionExpression[body.type='BlockStatement']",
          message: "JSX 属性中禁止多行箭头函数，请提取为命名函数",
        },
      ],

      // ── 复杂度控制 ──

      // 代码缩进嵌套不超过 4 层（类比 Go 的 nestif linter）
      "max-depth": ["error", 4],
      // 单个函数不超过 60 行（不算空行和注释），超了警告，鼓励拆分
      "max-lines-per-function": ["warn", { max: 60, skipBlankLines: true, skipComments: true }],
      // 圈复杂度不超过 12（= 函数里 if/for/switch 分支数，类比 Go 的 gocyclo）
      "complexity": ["warn", 12],

      // ── 类型安全 ──

      // 禁止 any 类型（Go 里没有 any，TypeScript 也应该写明确类型）
      "@typescript-eslint/no-explicit-any": "error",
      // 未使用的变量报错，_ 开头的豁免（和 Go 的 _ 忽略变量一致）
      // 例：const [_result, err] = ... → _result 不报错
      "@typescript-eslint/no-unused-vars": ["error", {
        argsIgnorePattern: "^_",
        varsIgnorePattern: "^_",
      }],

      // ── React ──

      // React Hooks 两条铁律：(1) 只能在函数顶层调用 (2) 只能在组件/hook 中调用
      "react-hooks/rules-of-hooks": "error",
      // useEffect 依赖数组是否写全（只警告，有时故意不写全）
      "react-hooks/exhaustive-deps": "warn",
      // 禁止 console.log（生产代码不应有），允许 console.warn 和 console.error
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },
  // ── 文件级豁免 ──
  {
    // lib/async.ts 是封装 try-catch 的唯一文件，允许它使用 try-catch
    files: ["src/lib/async.ts"],
    rules: {
      "no-restricted-syntax": "off",
    },
  },
  {
    // shadcn/ui 组件由 CLI 自动生成，不是手写的，放宽规则
    files: ["src/components/ui/**"],
    rules: {
      "max-lines-per-function": "off",  // 生成的组件可能超 60 行
      "no-ternary": "off",              // 生成的组件可能用三元
    },
  },
)
```

### P0.4 加严 tsconfig.json

```diff
- "noImplicitAny": false,
+ "noImplicitAny": true,
  // 改前：函数参数不写类型，TypeScript 默默当作 any，不报错
  // 改后：不写类型就报编译错误（和 Go 一样，函数参数必须写类型）

- "noUnusedParameters": false,
+ "noUnusedParameters": true
  // 改前：函数参数声明了没用，不报错
  // 改后：未使用的参数报编译错误（和 Go 的 "declared and not used" 一致）
```

### P0.5 添加 npm scripts

在 `frontend/package.json` 的 `scripts` 中追加：

```jsonc
{
  // 格式化所有代码（直接改文件）— 等价于 go fmt ./...
  "fmt": "prettier --write src/",
  // 只检查格式不改文件（CI 用）— 等价于 gofmt -l .
  "fmt:check": "prettier --check src/",
  // 静态检查，列出所有违规 — 等价于 go vet ./...
  "lint": "eslint src/",
  // 自动修复能修的问题 — 等价于 golangci-lint run --fix
  "lint:fix": "eslint src/ --fix",
  // 全套检查：类型 + lint + 格式（CI 跑这一条）— 等价于 go build && go vet && gofmt -l
  // tsc --noEmit = 只做类型检查不生成文件（等价于 go build 只检查编译通过）
  "check": "tsc --noEmit && eslint src/ && prettier --check src/"
}
```

### P0.6 编辑器保存自动格式化

确认 `.vscode/settings.json` 包含：

```jsonc
{
  // 指定 Prettier 插件作为代码格式化工具
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  // 按 Ctrl+S 保存时自动格式化（和 GoLand 保存时跑 go fmt 一样）
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    // 保存时同时自动修复 ESLint 能自动修复的问题
    "source.fixAll.eslint": "explicit"
  }
}
```

### P0 验收

- [ ] `npm run fmt` 能格式化全部代码
- [ ] `npm run lint` 能检测到当前代码中的违规（预期大量报错，这是正常的）
- [ ] `npm run check` 能跑通类型检查

---

## P1：创建工具函数

### P1.1 创建 `lib/async.ts`

```typescript
// lib/async.ts — 异步调用封装，项目中唯一允许 try-catch 的文件

import { useState } from "react"
import { extractError } from "./utils"

/**
 * 异步调用结果元组，类比 Go 的 (T, error)。
 * 成功：[result, null]，失败：[null, errorMessage]
 */
export type Result<T> = [T, null] | [null, string]

/**
 * 包装异步调用，将 try-catch 转换为 Go 风格的 [result, err] 元组。
 *
 * Go 类比：
 *   result, err := callService(fn)
 *   if err != nil { ... }
 */
export async function callService<T>(fn: () => Promise<T>): Promise<Result<T>> {
  try {
    const result = await fn()
    return [result, null]
  } catch (e: unknown) {
    return [null, extractError(e)]
  }
}

/**
 * 管理 loading 状态的 hook，内置 callService 调用。
 *
 * 用法：
 *   const saving = useLoading()
 *   const [result, err] = await saving.run(() => URLService.CreateSite(req))
 *   if (err) { setError(err); return }
 */
export function useLoading() {
  const [loading, setLoading] = useState(false)

  async function run<T>(fn: () => Promise<T>): Promise<Result<T>> {
    setLoading(true)
    const result = await callService(fn)
    setLoading(false)
    return result
  }

  return { loading, run }
}
```

### P1.2 创建 `lib/safe.ts`

```typescript
// lib/safe.ts — 零值工具函数，模仿 Go 的零值语义
// 目标：业务代码中不直接写 ?. 和 ??，统一在此收口

/**
 * 条件选择，替代三元运算符。
 * Go 类比：无（Go 用 if-else 赋值）
 */
export function pick<T>(cond: boolean, ifTrue: T, ifFalse: T): T {
  if (cond) return ifTrue
  return ifFalse
}

/** 空字符串安全：null/undefined → "" */
export function str(v: string | null | undefined): string {
  return v ?? ""
}

/** 空数组安全：null/undefined → [] */
export function arr<T>(v: T[] | null | undefined): T[] {
  return v ?? []
}

/** 空数字安全：null/undefined → 0 */
export function num(v: number | null | undefined): number {
  return v ?? 0
}

/** 空布尔安全：null/undefined → false */
export function bool(v: boolean | null | undefined): boolean {
  return v ?? false
}

/**
 * 统一拆包分页列表响应，消灭 store 中重复的 result?.items ?? [] 模式。
 *
 * Go 类比：把 *ListResult 的指针字段转为值类型
 */
export function unpackList<T>(
  result: { items?: T[] | null; total?: number; has_more?: boolean } | null,
): { items: T[]; total: number; hasMore: boolean } {
  return {
    items: result?.items ?? [],
    total: result?.total ?? 0,
    hasMore: result?.has_more ?? false,
  }
}
```

### P1.3 创建 `components/FormField.tsx`

```tsx
// components/FormField.tsx — 表单字段布局组件，减少表单嵌套层数

import type { ReactNode } from "react"

interface FormFieldProps {
  label: string
  required?: boolean
  error?: string
  children: ReactNode
}

export function FormField({ label, required, error, children }: FormFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-sm">
        {label}
        {required && <span className="text-destructive ml-0.5">*</span>}
      </label>
      {children}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}
```

### P1 验收

- [ ] `lib/async.ts` 中 `callService` 和 `useLoading` 可正常导入使用
- [ ] `lib/safe.ts` 中各工具函数可正常导入使用
- [ ] `components/FormField.tsx` 可正常渲染
- [ ] 以上三个文件通过 `npm run lint`

---

## P2：消灭 try-catch

所有异步调用从 try-catch 改为 `callService` 的 `[result, err]` 元组模式。

### P2.1 改造 `stores/url.ts`

涉及 8 个 async action：

| action | 当前模式 | 改造内容 |
|--------|---------|---------|
| `loadSites` | try-finally | `callService` + 合并 set |
| `loadMoreSites` | try-finally | 同上 |
| `selectSite` | try-finally | 同上 |
| `loadMoreBookmarks` | try-finally | 同上 |
| `selectBookmark` | try-catch（静默） | `callService`，err 时保持空态 |
| `search` | try-finally | 同上 |
| `loadMoreSearch` | try-finally | 同上 |
| `refreshCurrentSite` | 无错误处理 | 加上 `callService` |

改造示例（`loadSites`）：

```typescript
// 改造前
loadSites: async () => {
  set({ sitesLoading: true })
  try {
    const result = await URLService.ListSites({ page: 1, page_size: PAGE_SIZE })
    set({ sites: result?.items ?? [], sitesTotal: result?.total ?? 0, ... })
  } finally {
    set({ sitesLoading: false })
  }
},

// 改造后
loadSites: async () => {
  set({ sitesLoading: true })
  const [result] = await callService(() => URLService.ListSites({ page: 1, page_size: PAGE_SIZE }))
  const { items, total, hasMore } = unpackList(result)
  set({ sites: items, sitesTotal: total, sitesHasMore: hasMore, sitesPage: 1, sitesLoading: false })
},
```

### P2.2 改造 `features/url/SiteForm.tsx`

| 函数 | 改造内容 |
|------|---------|
| `handleFetch` | `useLoading` + `callService`，删除 `useCallback` |
| `onSubmit` | `useLoading` + `callService`，删除手动 try-catch-finally |

### P2.3 改造 `features/url/BookmarkForm.tsx`

| 函数 | 改造内容 |
|------|---------|
| `onSubmit` | `useLoading` + `callService` |

### P2.4 改造 `features/url/SiteDetail.tsx`

| 函数 | 改造内容 |
|------|---------|
| `SiteHeader.handleDelete` | 加 `callService`（当前无错误处理，报错会崩） |

### P2.5 改造 `features/url/BookmarkDetail.tsx`

| 函数 | 改造内容 |
|------|---------|
| `BookmarkNav.handleDelete` | 加 `callService`（当前无错误处理，报错会崩） |

### P2.6 改造 `features/url/BatchToolbar.tsx`

| 函数 | 改造内容 |
|------|---------|
| 批量删除/移动 handler | `callService` 替代 try-catch |

### P2.7 改造 `features/url/hooks.ts`

| 函数 | 改造内容 |
|------|---------|
| `useURLNormalize` | 内部 try-catch → `callService` |
| `useSiteLookup` | 3 层嵌套 try-catch → 线性 `callService` 链 |

### P2 验收

- [ ] 全局搜索 `try` 只在 `lib/async.ts` 和 `components/ui/` 中出现
- [ ] `npm run lint` 不报 try-catch 违规
- [ ] 所有删除操作都有错误处理（不再有 unhandled rejection）

---

## P3：消灭三元运算符

### P3.1 改造 `features/url/SiteDetail.tsx`

`BookmarkSection` 内 4 层三元链 → 提取 `BookmarkContent` 子组件，用 if + return：

```tsx
function BookmarkContent({ bookmarks, loading, viewMode, onSelect, batchMode, selectedIds }) {
  if (loading && bookmarks.length === 0) return <LoadingSpinner />
  if (bookmarks.length === 0) return <EmptyState icon={Bookmark} message="暂无书签" />
  if (viewMode === "grid") return <BookmarkGrid ... />
  return <BookmarkListView ... />
}
```

### P3.2 改造 `features/url/BookmarkForm.tsx`

站点匹配提示 4 段条件 → 提取 `SiteLookupHint` 子组件：

```tsx
function SiteLookupHint({ state, onCreateSite }) {
  if (state.status === "idle") return null
  if (state.status === "checking") return <p>...<Loader2 /> 检查站点...</p>
  if (state.status === "found") return <p>...<CheckCircle2 /> 已匹配</p>
  if (state.status === "not_found") return <div>...<AlertCircle /> 无对应站点...</div>
  return null
}
```

### P3.3 改造 `features/url/URLPage.tsx`

`detailView.type` 三元链 → 提取 `DetailPanel` 子组件用 if + return。

### P3.4 简单值三元 → `pick()`

全局搜索替换所有简单值三元表达式：

```typescript
// 改造前
isEdit ? "编辑站点" : "新建站点"
countLabel ?? `${count} 书签`
variant={viewMode === "grid" ? "secondary" : "ghost"}

// 改造后
pick(isEdit, "编辑站点", "新建站点")
countLabel ?? `${count} 书签`       // 这个是 ??，不是三元，在 P4 处理
pick(viewMode === "grid", "secondary" as const, "ghost" as const)
```

### P3 验收

- [ ] 全局搜索 `?` 不出现三元运算符（排除 `?.` 可选链）
- [ ] `npm run lint` 的 `no-ternary` 规则零报错
- [ ] 所有条件渲染走 if + return 或 `pick()`

---

## P4：收敛 `?.` `??`

目标：`?.` `??` 只出现在 `lib/` 的工具函数中，业务代码和 store 里不直接写。

### P4.1 改造 `stores/url.ts`

所有 `result?.items ?? []` 模式 → `unpackList(result)`：

```typescript
// 改造前
set({
  sites: result?.items ?? [],
  sitesTotal: result?.total ?? 0,
  sitesHasMore: result?.has_more ?? false,
})

// 改造后
const { items, total, hasMore } = unpackList(result)
set({ sites: items, sitesTotal: total, sitesHasMore: hasMore })
```

约 6 处分页拆包需要改造。

### P4.2 提取表单初始值函数

`SiteForm.tsx`：

```typescript
function siteDefaults(site?: Site | null) {
  return {
    title: str(site?.title),
    url: str(site?.url),
    description: str(site?.description),
    tags: arr(site?.tags),
    icon: str(site?.icon),
  }
}
```

`BookmarkForm.tsx`：

```typescript
function bookmarkDefaults(bm?: Bookmark | null) {
  return {
    url: str(bm?.url),
    title: str(bm?.title),
    description: str(bm?.description),
    tags: arr(bm?.tags),
  }
}
```

### P4.3 检查剩余 `??` 用法

逐文件排查，将散落的 `??` 替换为对应的 `str()` / `arr()` / `num()` 调用，或移入相应的 defaults 函数。

### P4 验收

- [ ] `src/stores/` 和 `src/features/` 中搜索 `??` 数量降至个位数
- [ ] 剩余的 `??` 有合理理由（如 `??` 用于非 null 默认值场景）

---

## P5：减少回调和 const 行数

### P5.1 Store selector 预组合 hook

在 `stores/url.ts` 底部导出预组合 hook：

```typescript
/** 站点列表所需的全部状态（替代组件内 7 行 selector） */
export function useSiteListState() {
  return {
    sites: useURLStore((s) => s.sites),
    loading: useURLStore((s) => s.sitesLoading),
    hasMore: useURLStore((s) => s.sitesHasMore),
    detailView: useURLStore((s) => s.detailView),
    loadSites: useURLStore((s) => s.loadSites),
    loadMore: useURLStore((s) => s.loadMoreSites),
    selectSite: useURLStore((s) => s.selectSite),
  }
}

/** 书签区域所需的全部状态 */
export function useBookmarkSectionState() {
  return {
    bookmarks: useURLStore((s) => s.bookmarks),
    loading: useURLStore((s) => s.bookmarksLoading),
    viewMode: useURLStore((s) => s.bookmarkViewMode),
    setViewMode: useURLStore((s) => s.setBookmarkViewMode),
    selectBookmark: useURLStore((s) => s.selectBookmark),
    refreshCurrentSite: useURLStore((s) => s.refreshCurrentSite),
  }
}
```

### P5.2 改造组件使用预组合 hook

| 文件 | 改造 |
|------|------|
| `SiteList.tsx` `DefaultSiteList` | 7 行 selector → `useSiteListState()` |
| `SiteList.tsx` `SearchResultList` | 6 行 selector → `useSearchResultState()` |
| `SiteDetail.tsx` `BookmarkSection` | 6 行 selector → `useBookmarkSectionState()` |

### P5.3 减少回调 prop 透传

| 文件 | 改造 |
|------|------|
| `SiteDetail.tsx` | `SiteForm` 保存后直接调 `useURLStore.getState().refreshCurrentSite()`，不通过 `onSave` 回调链 |
| `BookmarkDetail.tsx` | 同上，`BookmarkForm` 直接调 store |
| `URLPage.tsx` | `CreateFormDialog` 的回调简化为单一 `onDone`（只管关弹窗） |

改造模式：

```typescript
// 改造前：回调链
// 父：onSave={() => { setMode("view"); refreshCurrentSite() }}
// 子：onSave()

// 改造后：子组件直接调 store
// 父：onDone={() => setMode("view")}
// 子：
//   useURLStore.getState().refreshCurrentSite()
//   onDone()
```

### P5.4 删除 useCallback

| 文件 | 位置 |
|------|------|
| `SiteForm.tsx` | `handleFetch` 的 `useCallback` → 改为普通 `async function` |

### P5.5 表单使用 FormField 组件

| 文件 | 改造 |
|------|------|
| `SiteForm.tsx` | 4 个字段的 `<div><label>...<Controller>...{error}</div>` → `<FormField>` |
| `BookmarkForm.tsx` | 3 个字段同上 |

### P5 验收

- [ ] 搜索 `useCallback` 零结果
- [ ] 各组件开头的 selector 行数不超过 3 行（用预组合 hook）
- [ ] 回调 prop 不超过 1 个 `onDone`（弹窗/表单场景）
- [ ] 表单字段统一使用 `FormField` 包装

---

## P6：更新规范文档

### P6.1 更新 `frontend/AGENTS.md`

在"代码风格"部分增加以下规则（分为机器检查和人工审查两类）：

**机器检查（ESLint 强制）：**

- 禁止三元运算符 — 用 `if + return` 或 `pick()` 替代
- 禁止 try-catch — 用 `callService()` 返回 `[result, err]` 元组
- 禁止 useCallback / useMemo — 此项目规模不需要
- JSX 属性中禁止多行箭头函数 — 提取为命名函数
- 嵌套不超过 4 层，函数不超过 60 行
- 禁止 any，未使用变量报错（`_` 前缀豁免）

**人工 / AI 审查：**

- `?.` `??` 只写在 `lib/` 的工具函数中，业务代码不直接写
- 每个 `.tsx` 文件只 `export` 1 个主组件，辅助组件不导出
- 子组件直接调 store action，回调 prop 最多 1 个 `onDone`
- 条件渲染超过 2 个分支必须提取为独立子组件

### P6.2 更新 `docs/doc-index.md`

在全局文档表中增加本文档的链接。

### P6.3 工具函数使用指南

在 `frontend/AGENTS.md` 的工具函数部分增加：

| 函数 | 位置 | 用途 | Go 类比 |
|------|------|------|---------|
| `callService(fn)` | `lib/async.ts` | 异步调用 → `[result, err]` | `result, err := fn()` |
| `useLoading()` | `lib/async.ts` | loading 状态 + callService | 带进度的 `result, err` |
| `pick(cond, a, b)` | `lib/safe.ts` | 条件值选择 | `if cond { a } else { b }` |
| `str()` / `arr()` / `num()` | `lib/safe.ts` | null → 零值 | Go 零值语义 |
| `unpackList(result)` | `lib/safe.ts` | 分页响应拆包 | 指针字段 → 值字段 |
| `extractError(e)` | `lib/utils.ts` | Wails 错误解析 | `err.Error()` |
| `FormField` | `components/FormField.tsx` | 表单字段布局 | — |

### P6 验收

- [ ] `AGENTS.md` 包含完整的 lint 规则说明和工具函数清单
- [ ] `doc-index.md` 包含本文档链接
- [ ] 新 session 的 AI 能根据文档正确生成符合规范的代码

---

## ESLint 规则速查表

| ESLint 规则 | 级别 | 对应约束 |
|-------------|------|---------|
| `no-ternary` | error | 禁止三元运算符 |
| `no-restricted-syntax[TryStatement]` | error | 禁止 try-catch |
| `no-restricted-syntax[useCallback]` | error | 禁止 useCallback |
| `no-restricted-syntax[useMemo]` | error | 禁止 useMemo |
| `no-restricted-syntax[JSXAttribute BlockStatement]` | error | JSX 内禁止多行箭头 |
| `max-depth` | error (4) | 嵌套不超过 4 层 |
| `max-lines-per-function` | warn (60) | 函数不超过 60 行 |
| `complexity` | warn (12) | 圈复杂度不超过 12 |
| `@typescript-eslint/no-explicit-any` | error | 禁止 any |
| `@typescript-eslint/no-unused-vars` | error | 未使用变量（`_` 豁免） |
| `react-hooks/rules-of-hooks` | error | Hook 调用规则 |
| `react-hooks/exhaustive-deps` | warn | Hook 依赖完整性 |
| `no-console` | warn | 禁止 console.log |

**豁免文件：**

| 文件 | 豁免 | 原因 |
|------|------|------|
| `src/lib/async.ts` | `no-restricted-syntax` | 唯一的 try-catch 封装点 |
| `src/components/ui/**` | `no-ternary` + `max-lines-per-function` | shadcn CLI 生成 |
