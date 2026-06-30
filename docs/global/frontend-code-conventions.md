# 前端代码开发规范

> 目标：让前端代码对 Go 后端开发者友好 — 线性流程、少嵌套、少魔法、机器可检查。

---

## 编码原则

- 单层三元 `a ? b : c` 允许，禁止嵌套三元（多分支用 `if + return` 或提取函数）
- 用 `[result, err]` 元组替代 try-catch（类比 Go 的 `result, err := fn()`）
- `?.` `??` 收敛到 `lib/` 工具函数，业务代码不直接写
- 子组件直接调 store action，减少回调 prop 透传
- ESLint 强制执行语法约束，Prettier 统一格式

---

## ESLint 规则速查

规则详情及注释见 `frontend/eslint.config.js`。

| 规则 | 级别 | 说明 |
|------|------|------|
| `no-nested-ternary` | error | 禁止嵌套三元运算符，单层三元允许 |
| `no-restricted-syntax[TryStatement]` | error | 禁止 try-catch，用 `callService()` |
| `no-restricted-syntax[useCallback]` | error | 禁止 useCallback，直接写普通函数 |
| `no-restricted-syntax[useMemo]` | error | 禁止 useMemo（有实测性能需求时加 eslint-disable） |
| `no-restricted-syntax[JSX BlockStatement]` | error | JSX 属性中禁止多行箭头函数 |
| `max-depth` | error (4) | 嵌套不超过 4 层 |
| `max-lines-per-function` | warn (60) | 函数不超过 60 行 |
| `complexity` | warn (12) | 圈复杂度不超过 12 |
| `@typescript-eslint/no-explicit-any` | error | 禁止 any |
| `@typescript-eslint/no-unused-vars` | error | 未使用变量（`_` 前缀豁免） |
| `react-hooks/rules-of-hooks` | error | Hook 调用规则 |
| `react-hooks/exhaustive-deps` | warn | Hook 依赖完整性 |
| `no-console` | warn | 禁止 console.log（允许 warn/error） |

**豁免文件：**

| 文件 | 豁免 | 原因 |
|------|------|------|
| `src/lib/async.ts` | `no-restricted-syntax` | 唯一的 try-catch 封装点 |
| `src/components/ui/**` | `max-lines-per-function` | shadcn CLI 生成 |

---

## 工具函数

| 函数 | 位置 | 用途 | Go 类比 |
|------|------|------|---------|
| `callService(fn)` | `lib/async.ts` | 异步调用 → `[result, err]` 元组 | `result, err := fn()` |
| `runAsync(fn)` | `lib/async.ts` | 包装任意异步回调 → `err \| null` | `if err := fn(); err != nil` |
| `useLoading()` | `lib/async.ts` | loading 状态 + callService | 带进度的 `result, err` |
| `str()` / `arr()` / `num()` / `bool()` | `lib/safe.ts` | null → 零值 | Go 零值语义 |
| `unpackList(result)` | `lib/safe.ts` | 分页响应拆包 | 指针字段 → 值字段 |
| `extractError(e)` | `lib/utils.ts` | Wails 错误解析 | `err.Error()` |
| `FormField` | `components/FormField.tsx` | 表单字段布局 | — |

### 用法示例

```typescript
// 异步调用：Go 风格错误处理
const [result, err] = await callService(() => URLService.CreateSite(req))
if (err) {
  toast.error(err)
  return
}

// 带 loading 状态
const saving = useLoading()
const [result, err] = await saving.run(() => URLService.UpdateSite(req))

// 条件值：单层三元运算符
const title = isEdit ? "编辑站点" : "新建站点"

// 空值安全：替代 ?? 散落
const defaults = {
  title: str(site?.title),
  tags: arr(site?.tags),
}

// 分页拆包：替代 result?.items ?? []
const { items, total, hasMore } = unpackList(result)
```

---

## Go 友好编码规范

### useEffect 必须注释触发条件

Go 开发者习惯显式调用链，`useEffect` 的执行时机不直观。每个 `useEffect` 上方加一行注释说明何时触发：

```tsx
// 触发：siteId 变化时（用户切换站点），重置为查看模式
useEffect(() => {
  setMode("view")
}, [siteId])

// 触发：组件首次挂载，加载站点列表（等价于 Go 的 init）
useEffect(() => {
  loadSites()
}, [loadSites])

// 触发：外部 value 变化时同步本地状态（父组件清除搜索）
useEffect(() => {
  setLocalValue(value)
}, [value])
```

### Props 使用命名 interface

组件参数超过 3 个时，提取为命名 interface（类比 Go 的参数 struct）：

```tsx
// ✓ 好：命名 interface，一目了然
interface SiteListItemProps {
  title: string
  domain?: string
  count: number
  selected: boolean
  onClick: () => void
}

function SiteListItem(props: SiteListItemProps) {
  const { title, domain, count, selected, onClick } = props
  // ...
}

// ✗ 差：内联类型，参数列表过长
function SiteListItem({ title, domain, count, ... }: { title: string; domain?: string; ... }) {
```

3 个以内的简单 props 允许内联：

```tsx
// OK：参数少，内联清晰
function SiteIcon({ icon, size }: { icon?: string; size: number }) {
```

### 复杂异步逻辑提取为命名函数

闭包内超过 10 行的异步逻辑，提取为独立命名函数，让流程线性可读：

```tsx
// ✓ 好：步骤函数，流程清晰
async function doSiteLookup(url: string, version: number): Promise<SiteLookupState | null> {
  const [norm] = await callService(() => URLService.NormalizeURL(url))
  if (versionRef.current !== version) return null
  setNormalizedURL(str(norm))

  const [result, err] = await callService(() => URLService.LookupSiteByURL({ url }))
  if (versionRef.current !== version) return null
  if (err) return { status: "idle" }
  if (result?.found) return { status: "found", domain: result.domain }
  return { status: "not_found", domain: str(result?.domain) }
}

// useEffect 内只做调度
timerRef.current = setTimeout(async () => {
  const state = await doSiteLookup(urlValue, version)
  if (state) setLookupState(state)
}, 500)

// ✗ 差：setTimeout 内 20+ 行嵌套逻辑
```

### 文件内区域排序

每个 `.tsx` 文件按以下顺序组织（类比 Go 文件：类型声明 → 导出函数 → 内部函数）：

```
1. import 语句
2. 类型/interface 定义（Props 等）
3. 常量
4. 导出组件（export function）
5. 内部子组件（function，不导出）
6. 工具函数（纯逻辑，无 JSX）
```

区域之间用分隔注释：

```tsx
// ─── Types ──────────────────────────────────────────────────

interface SiteListItemProps { ... }

// ─── Exported ───────────────────────────────────────────────

export function SiteList() { ... }

// ─── Internal ───────────────────────────────────────────────

function DefaultSiteList() { ... }
```

---

## 预组合 Hooks

替代组件内多行 `useURLStore((s) => s.xxx)` 写法。

| Hook | 用途 |
|------|------|
| `useSiteListState()` | 站点列表：sites, loading, hasMore, loadSites, selectSite... |
| `useSearchResultState()` | 搜索结果：results, loading, hasMore, loadMore, selectResult... |
| `useBookmarkSectionState()` | 书签区域：bookmarks, loading, viewMode, selectBookmark... |

```typescript
// 替代 7 行 selector
const { sites, loading, hasMore, loadSites, selectSite } = useSiteListState()
```

---

## npm scripts

| 命令 | 用途 | Go 类比 |
|------|------|---------|
| `npm run fmt` | 格式化 src/ | `go fmt ./...` |
| `npm run fmt:check` | 检查格式（CI） | `gofmt -l .` |
| `npm run lint` | 静态检查 | `go vet ./...` |
| `npm run lint:fix` | 自动修复 | `golangci-lint run --fix` |
| `npm run check` | 类型+lint+格式 | `go build && go vet && gofmt` |

---

## TypeScript 编译选项

详情及注释见 `frontend/tsconfig.json`。

| 选项 | 值 | 说明 |
|------|-----|------|
| `strict` | true | 开启全部严格检查 |
| `noImplicitAny` | true | 参数必须写类型（类比 Go 函数签名） |
| `noUnusedParameters` | true | 未使用参数报错（`_` 前缀豁免） |
| `noUnusedLocals` | true | 未使用变量报错 |

---

## Prettier 格式化

详情及注释见 `frontend/prettier.config.js`。

| 选项 | 值 | 说明 |
|------|-----|------|
| `semi` | false | 无分号 |
| `singleQuote` | false | 双引号 |
| `trailingComma` | "all" | 尾逗号（减少 diff 噪音） |
| `printWidth` | 100 | 行宽 |
| `tabWidth` | 2 | 2 空格缩进 |
