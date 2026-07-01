# 后端 AI 开发指引

编写后端代码时必须遵守本文件中的规则。具体功能的业务逻辑见 `docs/features/` 下对应文档，数据结构见 `docs/global/data-structures.md`。

## 技术栈

| 职责 | 库 | 说明 |
|------|-----|------|
| 主存储 | BuntDB | 内存 KV，自动持久化，单文件 |
| 全文搜索 | Bleve v2 | 倒排索引，可从主存储重建 |
| 中文分词 | gse | 为 Bleve 提供 CJK analyzer |
| 快速哈希 | xxHash (cespare/xxhash/v2) | Merkle Tree 变化检测 |

所有依赖必须为纯 Go 实现，无 CGO。

## 目录约定

```
internal/
├── model/       # 纯 struct：实体、请求、响应（无依赖）
├── store/       # 数据访问：封装 BuntDB + Bleve
├── service/     # 业务逻辑：公开方法即前端 API
└── util/        # 无状态工具函数
```

- 依赖方向单向：`service → store → model`，不可反向
- `util` 可被 `store` 和 `service` 共同引用
- 文件命名省略 `_service` / `_store` 后缀，包名已表达层级
- service 内的辅助文件（scanner、fetcher、asset）为同包私有，不对外暴露

## Go 开发实践

- **错误逐层上抛**：store/util 返回 error 并用 `fmt.Errorf("xxx: %w", err)` 附加上下文；service 层是错误边界，统一决定返回给前端的错误信息。不用 panic 处理业务错误
- **错误信息统一英文**：所有面向前端的 error message 使用简洁英文（如 `"site title required"`、`"bookmark not found"`），前端原样展示。日志用 `slog` 记录完整 error chain 供后端排错
- **依赖注入**：service 通过 struct 字段持有 `*store.Store`，初始化在 main.go 完成组装后传入，不用全局变量
- **指针与 omitempty 规范**：
  - **实体/响应 struct**：值类型字段（`string`、`int`、`[]string` 等）一律**不加** `omitempty`，确保 JSON 始终输出，前端拿到稳定零值（`""`/`0`/`[]`），无需 `??` 防御。仅在**指针类型且零值有歧义**时才加 `omitempty`（如 `*int` 区分"未知"与 `0`，`*time.Time` 区分"从未发生"与零时间）
  - **Create 请求**：值类型。必填字段不加 `omitempty`，可选字段加 `omitempty`，不传就用零值
  - **Update 请求**：所有可更新字段用指针 + `omitempty`（`nil` = 不更新，非 `nil` = 更新）。`ID` 等定位字段为值类型不加 `omitempty`
  - **`omitempty` 双向影响**：序列化时零值字段不输出；Wails 生成 TypeScript 时映射为可选属性（`field?: type`）
- **日志用 `log/slog`**：结构化输出，如 `slog.Info("scan complete", "folder_id", id, "count", n)`。三个级别：Info（正常流程关键节点）、Warn（可恢复异常，如 Bleve 写入失败）、Error（不可恢复错误）

## 项目规则

- Service 层公开方法即前端可调用接口，入参和返回值统一用 struct
- BuntDB 先写先提交，Bleve 后写；Bleve 失败标记 `index_dirty`，不回滚 BuntDB
- 文件写入使用"写临时文件 → rename"原子替换
- 写操作在 `backupMu.RLock()` 保护下执行，备份操作取 `Lock()`
- ID 统一使用 UUID v4
- **路径由后端构建**：返回给前端的所有资源路径（图标、封面、缩略图、预览等）必须是完整可访问路径（如 `/persist/media-folders/{id}/thumbnails/xxx.jpg`），前端直接用作 `src`，不做任何路径拼接
