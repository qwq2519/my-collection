# 文档中心

"资料收藏夹"项目文档。根目录 `README.md` 保留项目概览；详细说明统一放在 `docs/` 下。

> 快速查找文档？看 [文档索引](./INDEX.md)。

## 文档分类

| 目录 | 职责 |
|------|------|
| `development/` | 开发环境、本地调试、工具链、编码约定 |
| `product/` | 产品目标、功能设计、用户流程（待补充） |
| `architecture/` | 系统架构、数据模型、技术决策（待补充） |
| `operations/` | 构建发布、安装包、版本管理（待补充） |

## 现有文档

- [功能需求](./product/requirements.md)
- [UI 界面设计](./product/ui-design.md)
- [存储技术栈决策](./architecture/storage-decision.md)
- [WSL + Windows 开发与构建](./development/windows-wsl.md)

## 新增文档

新文档放到对应分类目录，并更新本文件和 [INDEX.md](./INDEX.md)。

每篇文档建议包含以下结构：

```markdown
# 标题

## 目的
说明这篇文档要解决的问题。

## 读者
谁应该阅读这篇文档。

## 内容
具体方案、步骤或约定。

## 相关文档
- [文档索引](../INDEX.md)
```
