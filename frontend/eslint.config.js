/**
 * ESLint 静态分析配置。
 *
 * ESLint 类比 Go 的 golangci-lint：不管格式（那是 Prettier 的事），
 * 只管逻辑层面的代码质量和风格约束。
 *
 * 规则从上到下叠加，后面的覆盖前面的（类似 golangci-lint 的配置继承）。
 *
 * 规则级别：
 *   "error" = 红色报错，lint 不通过（类比 Go 编译错误）
 *   "warn"  = 黄色警告，lint 仍通过（类比 go vet 建议）
 *   "off"   = 关闭
 */
import js from "@eslint/js" // JavaScript 基础规则集
import ts from "typescript-eslint" // TypeScript 专用规则集
import reactHooks from "eslint-plugin-react-hooks" // React Hooks 使用规则
import prettier from "eslint-config-prettier" // 关掉和 Prettier 冲突的格式规则

export default ts.config(
  js.configs.recommended, // 第 1 层：JavaScript 社区推荐规则
  ...ts.configs.recommended, // 第 2 层：TypeScript 社区推荐规则
  prettier, // 第 3 层：关掉格式类规则（格式交给 Prettier）
  {
    plugins: {
      "react-hooks": reactHooks,
    },
    rules: {
      // ── Go 风格约束 ──────────────────────────────────────────

      // 禁止嵌套三元运算符 a ? b : c ? d : e（可读性差，用 if-else 或提取函数）
      // 单层三元 a ? b : c 允许使用
      "no-nested-ternary": "error",

      // no-restricted-syntax: 万能禁令，用 AST 选择器匹配任意语法结构
      // 类比 golangci-lint 的 forbidigo（禁止调用特定函数）
      "no-restricted-syntax": [
        "error",
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

      // ── 复杂度控制 ──────────────────────────────────────────

      // 代码缩进嵌套不超过 4 层（类比 Go 的 nestif linter）
      "max-depth": ["error", 4],

      // 单个函数不超过 60 行（不算空行和注释），超了警告，鼓励拆分
      "max-lines-per-function": ["warn", { max: 60, skipBlankLines: true, skipComments: true }],

      // 圈复杂度不超过 12（= 函数里 if/for/switch 分支数，类比 Go 的 gocyclo）
      complexity: ["warn", 12],

      // ── 类型安全 ────────────────────────────────────────────

      // 禁止 any 类型（Go 里没有 any，TypeScript 也应该写明确类型）
      "@typescript-eslint/no-explicit-any": "error",

      // 未使用的变量报错，_ 开头的豁免（和 Go 的 _ 忽略变量一致）
      // 例：const [_result, err] = ... → _result 不报错
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
        },
      ],

      // ── React ───────────────────────────────────────────────

      // React Hooks 两条铁律：(1) 只能在函数顶层调用 (2) 只能在组件/hook 中调用
      "react-hooks/rules-of-hooks": "error",

      // useEffect 依赖数组是否写全（只警告，有时故意不写全）
      "react-hooks/exhaustive-deps": "warn",

      // 禁止 console.log（生产代码不应有），允许 console.warn 和 console.error
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },

  // ── 文件级豁免 ────────────────────────────────────────────
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
      "max-lines-per-function": "off", // 生成的组件可能超 60 行
    },
  },
)
