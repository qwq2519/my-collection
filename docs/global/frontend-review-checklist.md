# 前端代码审查清单

> 写完代码后逐条自查。适用于人工 review 和 AI 自动审查。

---

## 代码规范自查

### 错误处理

- [ ] 异步调用是否用 `callService()` 包裹，返回 `[result, err]` 元组？
- [ ] 是否在 `if (err)` 后立即 `return`（Go early-return 风格）？
- [ ] 是否有裸露的 `try-catch`？（只有 `lib/async.ts` 允许）

### 空值处理

- [ ] 业务代码中是否有 `?.` 或 `??`？应改用 `str()` / `arr()` / `unpackList()`
- [ ] 分页结果是否用 `unpackList()` 拆包？

### 条件逻辑

- [ ] 是否有三元运算符？改用 `pick()` 或 `if + return`
- [ ] 条件渲染超过 2 个分支是否已提取为独立子组件？

### 组件结构

- [ ] 每个 `.tsx` 文件是否只 `export` 1 个主组件？
- [ ] 子组件是否直接调 store action，而非通过回调 prop 透传？（最多 1 个 `onDone`）
- [ ] Props 超过 3 个是否提取为命名 `interface XxxProps`？
- [ ] 文件内区域是否按顺序：Types → Exported → Internal → Utils？

### 函数复杂度

- [ ] 函数是否超过 60 行？拆分
- [ ] 嵌套是否超过 4 层？提前 return
- [ ] JSX 属性中是否有多行箭头函数 `() => { ... }`？提取为命名函数

### useEffect

- [ ] 每个 `useEffect` 上方是否有 `// 触发：xxx` 注释说明执行时机？
- [ ] 依赖数组是否正确？

### 异步逻辑

- [ ] 闭包内超过 10 行的异步逻辑是否已提取为命名步骤函数？
- [ ] 防抖/竞态是否有 version 保护？

### 表单

- [ ] 表单字段是否统一使用 `<FormField>` 组件包裹？
- [ ] 表单初始值是否提取为 `xxxDefaults()` 函数，使用 `str()` / `arr()` 拆包？

---

## UI/UX 自查

### 颜色

- [ ] 颜色是否只有灰阶 + 1 个强调色（primary）+ 1 个危险色（destructive）？
- [ ] 是否有多彩标签？→ 统一 `Badge variant="secondary"`

### 阴影与边框

- [ ] 是否有非浮层元素使用阴影？→ 只有下拉、弹窗可用 `shadow`，其余用 `border`
- [ ] 列表项是否被卡片包裹？→ 去掉 border + shadow，改用 `hover:bg-muted`

### 字体

- [ ] 字号是否在 `text-xs` 到 `text-base` 范围内？
- [ ] 是否靠字号跨度区分层级？→ 应用字重和灰度区分

### 间距

- [ ] 间距是否使用三档：紧凑(gap-1/2)、常规(gap-3/4)、宽松(gap-6/8)？
- [ ] 是否有过多留白？桌面工具应紧凑

### 图标

- [ ] 图标是否为 lucide-react 线性风格？禁止填充图标
- [ ] 是否给每行文字都加了图标？→ 仅导航和操作按钮用图标

### 动效

- [ ] 动效是否超过 200ms？→ 缩短到 150ms
- [ ] 是否有元素进场动画或 hover 放大？→ 禁止

### 组件

- [ ] 空状态是否居中灰色图标 + 一行说明？不用插画或 emoji
- [ ] 操作按钮是否右对齐：主操作 `Button` + 取消 `Button variant="ghost"`？
- [ ] 标签是否统一 `Badge variant="secondary"`？
