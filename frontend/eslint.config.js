import js from "@eslint/js"
import ts from "typescript-eslint"
import reactHooks from "eslint-plugin-react-hooks"
import prettier from "eslint-config-prettier"

export default ts.config(
  js.configs.recommended,
  ...ts.configs.recommended,
  prettier,
  {
    plugins: {
      "react-hooks": reactHooks,
    },
    rules: {
      "no-ternary": "error",

      "no-restricted-syntax": [
        "error",
        {
          selector: "TryStatement",
          message: "禁止 try-catch，用 callService() 返回 [result, err] 元组，见 lib/async.ts",
        },
        {
          selector: "CallExpression[callee.name='useCallback']",
          message: "禁止 useCallback，直接写普通函数",
        },
        {
          selector: "CallExpression[callee.name='useMemo']",
          message: "禁止 useMemo，除非有实测性能问题（加 eslint-disable 并注释原因）",
        },
        {
          selector: "JSXAttribute ArrowFunctionExpression[body.type='BlockStatement']",
          message: "JSX 属性中禁止多行箭头函数，请提取为命名函数",
        },
      ],

      "max-depth": ["error", 4],
      "max-lines-per-function": ["warn", { max: 60, skipBlankLines: true, skipComments: true }],
      complexity: ["warn", 12],

      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
        },
      ],

      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },
  {
    files: ["src/lib/async.ts"],
    rules: {
      "no-restricted-syntax": "off",
    },
  },
  {
    files: ["src/components/ui/**"],
    rules: {
      "max-lines-per-function": "off",
      "no-ternary": "off",
    },
  },
)
