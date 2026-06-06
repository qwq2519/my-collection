# 资料收藏夹

一个使用 Wails v3、Go、React、TypeScript、Vite 和 SWC 初始化的桌面小工具项目。

## 技术栈

- 后端：Go + Wails v3
- 前端：React + TypeScript + Vite + SWC
- 目标平台：Windows 桌面应用

## WSL + Windows 开发方式

当前代码位于 Windows 盘，例如在 WSL 中路径类似：

```bash
/mnt/f/collections
```

对应 Windows 路径：

```text
F:\collections
```

因为目标是 Windows 桌面应用，建议从 WSL 调用 Windows 侧工具链执行 Wails 命令：

```bash
cmd.exe /C "wails3 dev"
cmd.exe /C "wails3 build"
```

这样会使用 Windows 环境里的 Go、Node、npm、WebView2 和 Windows 构建相关工具，而不是 WSL/Linux 侧依赖。

## 环境要求

Windows 侧需要安装：

- Go
- Node.js / npm
- Wails v3 CLI：`wails3`
- Microsoft Edge WebView2 Runtime，通常 Windows 10/11 已内置

在 WSL 中可以这样检查 Windows 侧工具：

```bash
cmd.exe /C "where go && go version"
cmd.exe /C "where node && node --version"
cmd.exe /C "where npm && npm --version"
cmd.exe /C "where wails3 && wails3 version"
```

如果 Windows 侧没有 `wails3`，在 WSL 中执行：

```bash
cmd.exe /C "go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
```

## 开发

从项目根目录运行：

```bash
cmd.exe /C "wails3 dev"
```

这会启动 Wails 开发模式，并支持前端热更新。

也可以单独运行前端：

```bash
cd frontend
npm install
npm run dev
```

## 构建

构建 Windows 可执行文件：

```bash
cmd.exe /C "wails3 build"
```

生成结果通常位于：

```text
bin\collections.exe
```

## 不建议的方式

不建议在 WSL 中直接执行：

```bash
wails3 dev
wails3 build
```

直接使用 WSL/Linux 侧 `wails3` 会走 Linux 工具链，可能需要额外安装 `pkg-config`、GTK、WebKitGTK 等 Linux 依赖，并且不符合当前项目的 Windows 构建目标。

## 项目结构

- `main.go`：Wails 应用入口和窗口配置
- `greetservice.go`：模板生成的 Go 服务示例
- `frontend/`：React + TypeScript 前端代码
- `build/config.yml`：Wails 构建和产品信息配置
- `build/windows/`：Windows 图标、manifest、打包任务配置
- `Taskfile.yml`：Wails 任务入口

