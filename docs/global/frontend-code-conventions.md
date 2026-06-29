# 前端代码开发规范

> 目标：让前端代码对 Go 后端开发者友好 — 线性流程、少嵌套、少魔法、机器可检查。

---

## 编码原则

- 用 `if + return` 替代三元运算符（Go 没有三元运算符）
- 用 `[result, err]` 元组替代 try-catch（类比 Go 的 `result, err := fn()`）
- `?.` `??` 收敛到 `lib/` 工具函数，业务代码不直接写
- 子组件直接调 store action，减少回调 prop 透传
- ESLint 强制执行语法约束，Prettier 统一格式

---

## ESLint 规则速查

规则详情及注释见 `frontend/eslint.config.js`。

| 规则 | 级别 | 说明 |
|------|------|------|
| `no-ternary` | error | 禁止三元运算符，用 `if+return` 或 `pick()` |
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
| `src/components/ui/**` | `no-ternary` + `max-lines-per-function` | shadcn CLI 生成 |

---

## 工具函数

| 函数 | 位置 | 用途 | Go 类比 |
|------|------|------|---------|
| `callService(fn)` | `lib/async.ts` | 异步调用 → `[result, err]` 元组 | `result, err := fn()` |
| `useLoading()` | `lib/async.ts` | loading 状态 + callService | 带进度的 `result, err` |
| `pick(cond, a, b)` | `lib/safe.ts` | 条件值选择 | `if cond { a } else { b }` |
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

// 条件值：替代三元
const title = pick(isEdit, "编辑站点", "新建站点")

// 空值安全：替代 ?? 散落
const defaults = {
  title: str(site?.title),
  tags: arr(site?.tags),
}

// 分页拆包：替代 result?.items ?? []
const { items, total, hasMore } = unpackList(result)
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

## 人工 / AI 审查规则

以下规则无法由 ESLint 自动检查，需人工或 AI 审查：

1. **`?.` `??` 只写在 `lib/`** — 业务代码用 `str()`/`arr()`/`unpackList()`
2. **单组件导出** — 每个 `.tsx` 文件只 `export` 1 个主组件，辅助组件不导出
3. **回调限制** — 子组件直接调 store action，回调 prop 最多 1 个 `onDone`
4. **条件渲染** — 超过 2 个分支必须提取为独立子组件（if + return）
5. **表单字段** — 统一使用 `<FormField label="..." required error={...}>` 包裹
6. **表单初始值** — 提取 `xxxDefaults()` 函数，用 `str()`/`arr()` 拆包

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
