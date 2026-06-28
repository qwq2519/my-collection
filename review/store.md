# Store 层代码审查总结

## 问题：为什么 store 层逻辑复杂、职责不明确

当前 store 层承担了两类本质不同的工作：

### 1. 原子 CRUD（数据访问）

单实体、单事务的基础操作，如 `GetTag`、`SetTag`、`GetSite`、`CreateBookmark`。
这是 store 层的本职工作。

### 2. 跨实体编排（业务逻辑）

多实体联动的复合事务，如：

- `RenameURLTag` / `MergeURLTag` / `DeleteURLTagFromEntities`：遍历所有 site + bookmark，逐个检查并修改 tags 数组，更新注册表，再重建 Bleve 索引
- `CreateBookmark` / `DeleteBookmark`：写书签的同时更新 site 的 bookmark_count 和 updated_at
- `ListSites` / `ListMediaFiles`：根据请求参数决定走 BuntDB 还是 Bleve，并构建复杂的 conjunction/disjunction 查询

这些按理属于 service 层的"业务编排"职责。

### 根源：BuntDB 事务模型的约束

BuntDB 使用闭包事务 `db.Update(func(tx *buntdb.Tx) error {...})`，事务作用域被限制在闭包内。如果把跨实体操作拆到 service 层，各步骤分散在独立事务中，将丧失原子性——中途失败会导致数据不一致。

因此，为了保证原子性，开发时自然把所有相关操作塞进同一个 `db.Update` 闭包，导致 store 层函数越写越大、包含了业务决策逻辑。

---

## 三种改进方案

### 方案 1：暴露事务句柄给 service

store 层提供 `RunInUpdate(fn func(StoreTx) error)` 和一组接受 tx 参数的细粒度操作。service 在事务闭包内编排业务逻辑。

- 优点：职责最清晰、灵活性高、可测试性好
- 缺点：API 翻倍（有/无 tx 两套）、事务概念泄漏到 service、改造成本大
- 适合：多人协作的大项目

### 方案 2：保持现状，规范文件组织 ✅ 已选择

接受 store 层存在两种函数，通过文件拆分和命名约定明确区分：
- 原子 CRUD 函数：单实体单事务
- 编排事务函数：多实体单事务，含业务逻辑

- 优点：零/低改造成本、原子性天然保证、对桌面应用规模够用
- 缺点：职责边界依赖约定而非强制、复用性一般
- 适合：个人项目 / 小团队

### 方案 3：读写分离——查询上移，写事务保留

读路径（查询构建、路由决策）搬到 service 层；写路径（跨实体变更事务）留在 store 层。

- 优点：store 层显著变薄、改造成本适中、原子性仍有保证
- 缺点：读写割裂有 TOCTOU 风险（桌面单用户场景几乎无影响）、"变更集"抽象需要设计能力
- 适合：中等规模项目

---

## 方案 2 执行计划：按职责拆分 store 文件

拆分前：

```
internal/store/
├── bookmark.go    # CRUD + 跨实体联动（更新 site count）
├── buntdb.go      # DB 初始化 + 索引注册
├── index.go       # IndexManager 生命周期
├── media.go       # CRUD + 列表查询（含 Bleve 查询构建）
├── note.go        # CRUD + 列表查询
├── search.go      # Bleve 索引操作 + 脏队列 + 查询映射定义
├── site.go        # CRUD + 列表查询
├── store.go       # Store 主结构体
└── tag.go         # CRUD + 跨实体编排（rename/merge/delete/recount）
```

拆分后：

```
internal/store/
├── store.go       # Store 主结构体、New、Close
├── buntdb.go      # BuntDB 初始化 + 索引注册
├── index.go       # IndexManager 生命周期管理
├── search.go      # Bleve 文档映射、IndexDoc/DeleteDoc/Search、脏队列
├── note.go        # 笔记 CRUD（纯原子操作，无需拆分）
├── site.go        # 站点原子 CRUD：Create/Get/Update/Delete
├── site_list.go   # 站点列表查询：listFromDB / listFromBleve 路由与查询构建
├── bookmark.go    # 书签原子 CRUD：Create/Get/Update/Delete（含 site count 联动）
├── bookmark_list.go # 书签列表查询
├── media.go       # 媒体文件夹 CRUD + meta/tree_hash 读写
├── media_list.go  # 媒体文件列表查询：listFromDisk / listFromBleve
├── tag.go         # 标签注册表原子操作：Get/List/Set/Delete/AdjustCount
└── tag_ops.go     # 标签编排事务：Rename/Merge/DeleteFromEntities/Recount
```

约定规则：
- `xxx.go` — 原子 CRUD，单实体单事务
- `xxx_list.go` — 列表/搜索查询逻辑
- `xxx_ops.go` — 跨实体编排事务（多实体联动 + 业务规则）
