# 开发指南

WSL 中编辑代码，Windows 工具链构建 Wails 桌面应用。

## 核心原则

本项目目标是 Windows 桌面应用，在 WSL 中通过 `cmd.exe` 调用 Windows 侧工具链：

```bash
cmd.exe /C "wails3 dev"       # 开发模式（支持前端热更新）
cmd.exe /C "wails3 build"     # 构建 → bin\collections.exe
```

不要在 WSL/Linux 侧直接运行 `wails3`，否则会走 Linux 工具链，需要 GTK/WebKitGTK 等依赖，且不符合 Windows 构建目标。

## 路径关系

项目代码位于 Windows 盘：

```text
WSL 路径:     /mnt/f/collections
Windows 路径: F:\collections
```

`cmd.exe /C "..."` 在 Windows 环境中执行，使用 Windows 侧的 Go、Node、npm、Wails、WebView2。

## Windows 侧环境要求

| 工具 | 说明 |
|------|------|
| Go | Go 编译器 |
| Node.js / npm | 前端构建 |
| Wails v3 CLI (`wails3`) | 应用构建框架 |
| WebView2 Runtime | Windows 10/11 通常已内置 |

检查 Windows 侧工具是否就绪：

```bash
cmd.exe /C "where go && go version"
cmd.exe /C "where node && node --version"
cmd.exe /C "where npm && npm --version"
cmd.exe /C "where wails3 && wails3 version"
```

安装 Wails v3 CLI：

```bash
cmd.exe /C "go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
```
