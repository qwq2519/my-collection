# 文档中心

这里存放“资料收藏夹”的长期项目文档。根目录 `README.md` 只保留项目概览、快速开始和文档入口；详细说明统一放在 `docs/` 下维护。

## 文档职责

- `product/`：产品目标、功能设计、用户流程、需求拆分。
- `architecture/`：系统架构、模块边界、数据模型、技术决策。
- `development/`：开发环境、本地调试、编码约定、工具链说明。
- `operations/`：构建发布、安装包、故障排查、版本发布流程。
- `templates/`：新增文档时可复用的模板。

## 当前文档

- [开发文档](./development/README.md)
- [WSL + Windows 开发与构建](./development/windows-wsl.md)
- [产品文档](./product/README.md)
- [架构文档](./architecture/README.md)
- [运维文档](./operations/README.md)
- [文档模板](./templates/document-template.md)

## 新增文档规则

新增文档时优先放到对应分类目录，并在对应分类的 `README.md` 与本文件中补充入口。

建议每篇文档包含：

- 目的：这篇文档解决什么问题。
- 读者：谁应该阅读这篇文档。
- 范围：本文覆盖什么，不覆盖什么。
- 内容：具体方案、步骤或约定。
- 相关文档：可继续阅读的上下游文档。
