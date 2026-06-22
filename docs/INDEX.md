# 文档索引

快速定位本项目所有文档。

## 文档一览

| 文档 | 路径 | 说明 |
|------|------|------|
| 项目概览 | [`../README.md`](../README.md) | 项目定位、技术栈、结构概览 |
| 文档中心 | [`README.md`](./README.md) | 文档分类规则、写作规范、未来规划 |
| 功能需求 | [`product/requirements.md`](./product/requirements.md) | 功能定义、数据规模、行为规则 |
| UI 界面设计 | [`product/ui-design.md`](./product/ui-design.md) | 布局、导航、交互模式 |
| 存储技术栈决策 | [`architecture/storage-decision.md`](./architecture/storage-decision.md) | BuntDB + Bleve 选型理由与开发注意事项 |
| WSL 开发指南 | [`development/windows-wsl.md`](./development/windows-wsl.md) | WSL + Windows 工具链开发与构建流程 |

## 按主题查找

### 产品与需求

| 问题 | 去哪里看 |
|------|----------|
| 三大功能模块是什么？ | [requirements.md](./product/requirements.md#三大功能模块) |
| 图片管理怎么做？ | [requirements.md § 图片管理](./product/requirements.md#3-图片管理) |
| 搜索能力？ | [requirements.md § 搜索能力](./product/requirements.md#搜索能力) |
| 标签系统规则？ | [requirements.md § 标签系统](./product/requirements.md#标签系统) |
| 备份怎么做？ | [requirements.md § 数据持久化与备份](./product/requirements.md#数据持久化与备份) |
| 界面布局？ | [ui-design.md](./product/ui-design.md#整体布局) |
| 侧边栏设计？ | [ui-design.md § 侧边栏](./product/ui-design.md#侧边栏) |

### 架构与存储

| 问题 | 去哪里看 |
|------|----------|
| 用什么存储方案？为什么？ | [storage-decision.md](./architecture/storage-decision.md#结论) |
| 为什么不用 SQLite？ | [storage-decision.md § 为什么不选 SQLite](./architecture/storage-decision.md#为什么不选-sqlite) |
| KV 和关系型数据库有什么区别？ | [storage-decision.md § 开发差异](./architecture/storage-decision.md#kvjson-vs-关系型开发差异) |
| 查询走哪个组件？ | [storage-decision.md § 职责分工](./architecture/storage-decision.md#职责分工) |

### 开发环境与工具链

| 问题 | 去哪里看 |
|------|----------|
| 怎么启动开发模式？ | [windows-wsl.md § 开发命令](./development/windows-wsl.md#开发命令) |
| 怎么构建 exe？ | [windows-wsl.md § 构建命令](./development/windows-wsl.md#构建命令) |
| Windows 侧需要装什么？ | [windows-wsl.md § Windows 侧环境要求](./development/windows-wsl.md#windows-侧环境要求) |
| WSL 和 Windows 路径怎么对应？ | [windows-wsl.md § 路径关系](./development/windows-wsl.md#路径关系) |
| 为什么不能在 WSL 里直接 wails3？ | [windows-wsl.md § 不推荐方式](./development/windows-wsl.md#不推荐方式) |

### 项目配置

| 问题 | 去哪里看 |
|------|----------|
| 项目技术栈是什么？ | [README.md](../README.md) |
| Wails 构建配置在哪？ | [`build/config.yml`](../build/config.yml) |
| Task 命令有哪些？ | [`Taskfile.yml`](../Taskfile.yml) |
| 前端依赖和脚本？ | [`frontend/package.json`](../frontend/package.json) |

### 文档贡献

| 问题 | 去哪里看 |
|------|----------|
| 怎么写新文档？ | [文档中心 § 新增文档](./README.md#新增文档) |
| 文档放在哪个目录？ | [文档中心 § 文档分类](./README.md#文档分类) |

## 规划中的文档

以下文档尚未编写，待项目推进时逐步补充：

- `docs/product/vision.md` — 产品目标与边界
- `docs/product/features.md` — 功能清单与优先级
- `docs/architecture/overview.md` — 整体架构设计
- `docs/architecture/data-model.md` — Key 结构与数据模型设计
- `docs/operations/release.md` — 发布流程
- `docs/operations/packaging.md` — Windows 安装包说明
