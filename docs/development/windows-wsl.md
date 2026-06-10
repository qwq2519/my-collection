# WSL + Windows 开发与构建

## 目的

说明在 WSL 中编辑代码、但使用 Windows 工具链开发和构建 Wails 桌面应用的推荐方式。

## 读者

- 在 WSL 终端里开发本项目的人。
- 需要运行 `wails3 dev` 或 `wails3 build` 的人。
- 需要判断应该安装 Linux 依赖还是 Windows 依赖的人。

## 结论

本项目目标是 Windows 桌面应用，推荐在 WSL 中调用 Windows 侧工具链：

```bash
cmd.exe /C "wails3 dev"
cmd.exe /C "wails3 build"
```

不要优先在 WSL/Linux 侧直接运行 `wails3 dev` 或 `wails3 build`。

## 路径关系

项目代码位于 Windows 盘，在 WSL 中路径类似：

```bash
/mnt/f/collections
```

对应 Windows 路径：

```text
F:\collections
```

从 WSL 调用 `cmd.exe /C "..."` 时，命令会在 Windows 环境中执行，因此会使用 Windows 侧的 Go、Node、npm、Wails 和 WebView2。

## Windows 侧环境要求

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

## 开发命令

从项目根目录启动开发模式：

```bash
cmd.exe /C "wails3 dev"
```

这会启动 Wails 开发模式，并支持前端热更新。

## 构建命令

构建 Windows 可执行文件：

```bash
cmd.exe /C "wails3 build"
```

生成结果通常位于：

```text
bin\collections.exe
```

## 不推荐方式

不建议在 WSL 中直接执行：

```bash
wails3 dev
wails3 build
```

原因：直接使用 WSL/Linux 侧 `wails3` 会走 Linux 工具链，可能需要额外安装 `pkg-config`、GTK、WebKitGTK 等 Linux 依赖，并且不符合当前项目的 Windows 构建目标。

## 相关文档

- [开发文档](./README.md)
- [文档中心](../README.md)
