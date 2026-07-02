# 后端分层职责与防御性编程规范

基于代码审查总结的分层准则，用于指导后续开发和 Code Review。

## 分层职责边界

### Service 层（API 边界）

| 职责 | 说明 |
|------|------|
| 入参校验 | 必填字段非空、格式合法、值域合理 |
| 业务编排 | 组合多个 store 调用、文件操作、事件推送 |
| 标签 count 维护 | 调用 `util.ComputeTagDeltas` + `store.BatchAdjustTagCounts` |
| 资源清理 | 删除关联文件（icon、附件、缩略图） |
| 错误边界 | 决定返回给前端的错误信息（简洁英文） |

### Store 层（数据访问）

| 职责 | 说明 |
|------|------|
| CRUD 事务 | BuntDB 读写 + Bleve 索引同步 |
| 数据完整性约束 | 唯一性校验、引用存在性（在事务内，防 TOCTOU） |
| 数据格式保证 | `EnsureSlices`、JSON 序列化/反序列化 |
| not-found 错误 | 统一转译为实体级错误信息（如 `"bookmark not found"`） |

### 校验归属判定规则

```
用户输入合法性   → Service（如 title 非空、URL 格式）
数据状态一致性   → Store 事务内（如域名唯一、bookmark_count > 0 时拒绝删除站点）
两者都需要时    → Service 做前置快速失败，Store 在事务内做最终保证
```

**不重复校验**：如果 store 事务内已做了某项校验（如 `getSiteTx` 返回 not-found），service 层不需要提前 `GetSite` 做同样的存在性检查——除非 service 需要旧数据用于后续逻辑（如 tag diff、资源清理）。

## 消除冗余的核心原则

### 原则 1：Store 写操作返回足够信息，避免 service 预查询

**反模式**：service 先 `Get` 拿旧实体 → 再调 store `Delete`/`Update`（store 内部事务又读一次）。

**正确做法**：store 的 Delete 返回被删实体，Update 返回旧值（或 diff 所需的旧字段），service 用返回值做后续清理。

```go
// ✗ 反模式：读两次
func (s *Service) DeleteSite(id string) error {
    site, _ := s.Store.GetSite(id)     // 第 1 次读
    s.Store.DeleteSite(id)             // 内部事务第 2 次读
    s.adjustTagCounts(nil, site.Tags)
}

// ✓ 正确：读一次
func (s *Service) DeleteSite(id string) error {
    site, _ := s.Store.DeleteSite(id)  // 事务内读 + 删，返回被删实体
    s.adjustTagCounts(nil, site.Tags)
}
```

### 原则 2：同一函数内不重复调用做相同事情的 util 函数

**反模式**：先 `ValidateURL` 再 `ExtractDomain`，两者内部都调 `parseAndValidate`。

**正确做法**：`ExtractDomain` 已包含完整校验，直接调用即可。若需要校验但不需要域名，才单独调 `ValidateURL`。

### 原则 3：批量操作在 store 层用单事务完成

**反模式**：service 层 for 循环逐条 `GetXxx` + `UpdateXxx`（每次一个事务）。

**正确做法**：store 提供批量方法，在单个 BuntDB 事务内完成所有读写。

### 原则 4：相同模式的代码提取为共享 helper

**适用场景**：
- `adjustURLTagCounts` 和 `adjustMediaTagCounts` 逻辑完全一致，仅 prefix 不同
- `tag_ops.go` 中三个函数（rename/merge/delete）的 site+bm 遍历循环结构相同

### 原则 5：防御性检查保留在 API 边界，内部调用信任上游

**保留**：service 公开方法的空 ID 检查（Wails API 边界，用户可能传空值）

**不需要**：store 内部的存在性检查如果 service 刚查过且无并发修改路径

**特殊保留**：负数钳位（`clampBookmarkCount`、tag count ≥ 0）作为数据完整性兜底，保留但标注为 integrity guard

## 错误处理规范

### not-found 统一策略

Store 层所有 `getXxxTx` / `GetXxx` 方法统一处理 `buntdb.ErrNotFound`，转译为实体级错误：

```go
// Store 层统一处理
if err == buntdb.ErrNotFound {
    return nil, fmt.Errorf("site not found")
}
```

Service 层不再包装 `"xxx not found: %w"`——store 返回的错误即为最终面向前端的错误信息。

### 错误包装规则

- **Store → Service**：store 返回的错误已含上下文（如 `"site not found"`、`"site already exists for domain "xxx""`），service 直接返回，不再 `fmt.Errorf("xxx: %w", err)`
- **Util → Service**：util 返回的错误已是用户可读信息（如 `"only http/https supported"`），service 直接返回
- **仅在转换语义时包装**：如 store 返回内部错误但 service 要给前端不同的表述

## 索引同步模式

store 层书签 CRUD 操作同时更新书签和所属站点的 Bleve 索引（因为书签增删改会联动站点 `updated_at`/`bookmark_count`）。使用 `reindexURLEntities` 统一处理，避免各处手写 `IndexDoc` 调用。

## Code Review 检查清单

- [ ] Service 方法是否对同一实体做了"先 Get 再 Delete/Update"？→ 考虑让 store 返回旧数据
- [ ] 同一函数内是否对同一输入调了两个内部逻辑重叠的 util 函数？
- [ ] 批量操作是否在 service 层 for 循环中逐条调 store？→ 考虑 store 批量方法
- [ ] Store 方法的 not-found 是否返回了实体级错误？→ 统一转译
- [ ] Service 是否重新包装了 store/util 已经用户可读的错误？→ 直接返回
- [ ] 新增的 helper 是否与已有 helper 逻辑重复？→ 检查 `adjustXxxTagCounts`、`mediaBleveFields` 等
