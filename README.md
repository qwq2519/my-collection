# 资料收藏夹

收集、整理和检索个人资料链接与本地记录的 Windows 桌面小工具。

技术栈：Wails v3 + Go + React + TypeScript + Vite

[Wails v3 官方文档](https://v3.wails.io/quick-start/installation/)

## 快速开始

本项目可在 WSL 中编辑代码，通过 Windows 工具链构建。详见 [开发指南](./docs/global/dev-guide.md)。

```bash
cmd.exe /C "wails3 dev"       # 开发模式
cmd.exe /C "wails3 build"     # 构建 → bin\collections.exe
```

## 项目结构

```text
main.go              Wails 应用入口
greetservice.go      Go 服务示例
frontend/            React + TypeScript 前端
build/config.yml     Wails 构建配置
build/windows/       Windows 打包配置（当前目标平台）
build/darwin|linux|…  Wails 框架默认生成，暂不使用
Taskfile.yml         任务入口
docs/                项目文档
```

## 文档

- [文档导航](./docs/doc-index.md) — 所有文档入口
- [开发指南](./docs/global/dev-guide.md) — WSL + Windows 环境与构建

## 前端开发命令

在 `frontend/` 目录下执行：

```bash
npm run dev           # Vite 开发服务器
npm run check         # 类型检查 + ESLint + Prettier（提交前必跑）
npm run lint          # ESLint 静态分析
npm run fmt           # Prettier 格式化

# 依赖关系可视化（需安装 graphviz: brew install graphviz）
npm run deps          # 生成 deps.svg 组件依赖图（浏览器打开查看）
npm run deps:circular # 检测循环依赖
```

查看单个模块的依赖关系：

```bash
npx madge --extensions ts,tsx --image url-module.svg src/features/url/
```

查看某个组件被谁引用（反向依赖）：

```bash
npx madge --extensions ts,tsx --depends src/components/TagList.tsx src/
```
