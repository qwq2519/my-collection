# React 概念速查（Go 开发者版）

> 遇到不熟的 React/前端概念时查阅。用 Go 类比解释核心机制。

---

## 核心概念映射

| React 概念 | Go 类比 | 一句话解释 |
|-----------|---------|-----------|
| 组件函数 | handler 函数 | 每次渲染 = 调用一次函数，返回 UI 描述 |
| JSX | `text/template` | 声明式 UI 模板，编译为函数调用 |
| props | 函数参数（struct） | 父组件传入，只读 |
| `useState(init)` | 带 setter 的局部变量 | 修改后触发组件重新执行 |
| `useEffect(fn, [deps])` | `sync.Once` / watcher | deps 变化时执行副作用 |
| `useRef(init)` | `*T` 指针 | 修改不触发重渲染，跨渲染保持引用 |
| `key` prop | map 的 key | 告诉 React 哪些元素是同一个实例 |
| 条件渲染 `{cond && <X/>}` | `if cond { render(X) }` | JSX 中没有 if 语句，用表达式 |

---

## 状态管理（Zustand）

| Zustand 概念 | Go 类比 | 说明 |
|-------------|---------|------|
| store | 全局 struct + mutex | 存放共享状态和操作方法 |
| `set({...})` | `mu.Lock(); s.field = v; mu.Unlock()` | 修改状态，自动通知订阅者 |
| `get()` | 读取当前快照 | 在 action 内获取最新状态 |
| `useStore((s) => s.field)` | 订阅特定字段 | 字段变化时组件自动重渲染 |
| selector `(s) => s.xxx` | 字段访问器 | 精确订阅，避免无关变化触发渲染 |

```typescript
// 等价于 Go 的：
// type URLStore struct { sites []Site; loading bool }
// func (s *URLStore) LoadSites() { ... }

const useURLStore = create<URLState>((set, get) => ({
  sites: [],           // struct field
  loading: false,      // struct field
  loadSites: async () => {  // struct method
    set({ loading: true })
    const [result] = await callService(...)
    set({ sites: result, loading: false })
  },
}))
```

---

## 生命周期

React 组件没有 Go struct 的 `New()` / `Close()` 显式生命周期。用 `useEffect` 模拟：

```typescript
// 等价于 Go 的 init() + defer cleanup()
useEffect(() => {
  // init：组件挂载时执行
  const sub = subscribe(...)

  return () => {
    // cleanup：组件卸载时执行（等价于 defer）
    sub.Close()
  }
}, [])  // 空依赖 = 只执行一次
```

**依赖数组规则：**

| 写法 | 含义 | Go 类比 |
|------|------|---------|
| `useEffect(fn, [])` | 挂载执行一次 | `init()` |
| `useEffect(fn, [a, b])` | a 或 b 变化时执行 | `Watch(a, b, fn)` |
| `useEffect(fn)` | 每次渲染都执行（少用） | 无（通常是 bug） |

---

## 异步模式对比

| 场景 | Go 写法 | 本项目写法 |
|------|---------|-----------|
| 调用 + 错误处理 | `result, err := fn()` | `const [result, err] = await callService(fn)` |
| 并发请求 | `go func(){}()` | `Promise.all([...])` |
| 防抖 | `time.AfterFunc` + `timer.Reset` | `setTimeout` + `clearTimeout` |
| 竞态保护 | `context.WithCancel` | `versionRef` 递增对比 |
| 带超时 | `ctx, cancel := context.WithTimeout(...)` | `AbortController` + `signal`（少用） |

---

## 渲染机制

Go 开发者最需要理解的一点：**React 组件函数每次状态变化都会重新执行**。

```typescript
function SiteList() {
  // ↓ 每次 sites 变化，这个函数整体重新执行
  const sites = useURLStore((s) => s.sites)

  // ↓ 每次执行都重新创建这个数组（但 React 会 diff，不会重建 DOM）
  return (
    <div>
      {sites.map(site => <SiteItem key={site.id} site={site} />)}
    </div>
  )
}
```

**为什么不需要担心性能：**
- React 的 virtual DOM diff 很快（类似 Git 的 tree diff）
- 只有实际变化的 DOM 节点才会更新
- Zustand selector 确保只有订阅的字段变化才触发重执行

---

## 常见困惑解答

### 为什么变量不需要 mut？

Go 中变量修改是直接赋值。React 中必须通过 `setState` 触发重渲染：

```typescript
// ✗ 错误：直接赋值不会触发 UI 更新
let count = 0
count = 1  // UI 不变

// ✓ 正确：通过 setter 触发重渲染
const [count, setCount] = useState(0)
setCount(1)  // UI 更新
```

### 为什么 hook 不能放在 if 里？

React 用调用顺序追踪 hook 状态（类似数组索引）。如果放在条件里，顺序会变，状态错乱：

```typescript
// ✗ 错误
if (showDetail) {
  const [data, setData] = useState(null)  // 顺序不稳定
}

// ✓ 正确：始终调用，用条件控制行为
const [data, setData] = useState(null)
if (showDetail) { /* 用 data */ }
```

### Store action vs 回调 prop？

本项目约定：子组件直接调 store action，不层层传递回调：

```typescript
// ✗ 差：prop 透传链
<Parent onDelete={handleDelete}>
  <Child onDelete={props.onDelete}>
    <Button onClick={props.onDelete} />

// ✓ 好：子组件直接调 store
function DeleteButton() {
  const deleteSite = useURLStore((s) => s.deleteSite)
  return <Button onClick={() => deleteSite(id)} />
}
```
