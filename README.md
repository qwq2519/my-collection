# 资料收藏夹

资料收藏夹是一个使用 Wails v3、Go、React、TypeScript、Vite 和 SWC 开发的 Windows 桌面小工具。

[wails 框架官方文档](https://v3.wails.io/quick-start/installation/)

## 项目定位

- 收集、整理和检索个人资料链接与本地资料记录。
- 使用 Go 负责桌面端能力、数据读写和系统集成。
- 使用 React + TypeScript 负责交互界面和状态管理。

## 快速开始

从项目根目录启动开发模式：

```bash
cmd.exe /C "wails3 dev"
```

构建 Windows 可执行文件：

```bash
cmd.exe /C "wails3 build"
```

生成结果通常位于：

```text
bin\collections.exe
```

## 文档入口

- [文档中心](./docs/README.md)
- [WSL + Windows 开发与构建](./docs/development/windows-wsl.md)

## 项目结构

- `main.go`：Wails 应用入口和窗口配置
- `greetservice.go`：模板生成的 Go 服务示例
- `frontend/`：React + TypeScript 前端代码
- `build/config.yml`：Wails 构建和产品信息配置
- `build/windows/`：Windows 图标、manifest、打包任务配置
- `Taskfile.yml`：Wails 任务入口
- `docs/`：项目文档中心
