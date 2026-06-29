/**
 * Prettier 格式化配置。
 *
 * Prettier 类比 Go 的 gofmt：强制统一格式，没有争论空间。
 * 保存时自动运行（通过 .vscode/settings.json 的 formatOnSave），
 * 也可手动 npm run fmt。
 */
export default {
  // 语句末尾不加分号：const x = 1（不写 const x = 1;）
  semi: false,

  // 字符串用双引号："hello"（不写 'hello'）
  singleQuote: false,

  // 最后一项后面也加逗号（和 Go struct 初始化一致，减少 git diff 噪音）
  trailingComma: "all",

  // 一行最多 100 字符，超了自动换行
  printWidth: 100,

  // 缩进用 2 个空格（前端惯例，Go 用 tab）
  tabWidth: 2,

  // 箭头函数参数总加括号：(x) => x + 1（不写 x => x + 1，更一致）
  arrowParens: "always",

  // 换行符用 \n（Linux/Mac 风格，不用 Windows 的 \r\n）
  endOfLine: "lf",
}
