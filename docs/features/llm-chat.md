# LLM 对话（暂不实现）

内置大模型对话功能，支持多轮对话、重新生成、隐藏中间轮次。

## 需求

### 核心能力

| 能力 | 说明 |
|------|------|
| 多轮对话 | 支持连续多轮 user/assistant 交互 |
| 多会话 | 可创建多个独立对话，列表管理 |
| 重新生成 | 对 assistant 回复重新生成，保留历史版本，可切换 |
| 隐藏轮次 | 隐藏某轮 user+assistant，发送请求时跳过，不删除 |
| 模型切换 | 支持配置多个模型（API 地址 + Key），按对话选择 |
| 流式输出 | 打字机效果逐字展示 |

### 不做

- 不做 RAG / 知识库检索增强（后续可考虑结合本地收藏数据）
- 不做图片/文件上传（首版纯文本对话）
- 不做分叉编辑（编辑早期消息后产生新分支），首版用线性+版本模型

## 存储设计

### 架构

BuntDB 存对话注册表（轻量元数据），JSON 文件存完整对话内容。对话内容不常驻内存，打开对话时按需加载。

```text
BuntDB:
  conv:{id} → { id, title, model, created_at, updated_at }

persist/conversations/
  └── {conv_id}.json
```

### conv 注册表（BuntDB）

```json
{
  "id": "conv-uuid",
  "title": "Go 接口讨论",
  "model": "gpt-4o",
  "created_at": "2026-06-24T10:00:00Z",
  "updated_at": "2026-06-24T11:30:00Z"
}
```

### 对话内容（JSON 文件）

采用线性 + 版本数组模型：

```json
{
  "schema_version": 1,
  "id": "conv-uuid",
  "system_prompt": "你是一个编程助手",
  "messages": [
    {
      "id": "msg-1",
      "role": "user",
      "content": "解释下 Go 的接口",
      "hidden": false,
      "created_at": "2026-06-24T10:00:00Z"
    },
    {
      "id": "msg-2",
      "role": "assistant",
      "content": "当前展示的回复内容...",
      "hidden": false,
      "versions": [
        { "content": "第一次生成的回复...", "created_at": "2026-06-24T10:00:05Z" },
        { "content": "重新生成的回复...", "created_at": "2026-06-24T10:01:00Z" }
      ],
      "active_version": 1,
      "created_at": "2026-06-24T10:00:05Z"
    },
    {
      "id": "msg-3",
      "role": "user",
      "content": "这轮被隐藏了",
      "hidden": true,
      "created_at": "2026-06-24T10:02:00Z"
    },
    {
      "id": "msg-4",
      "role": "assistant",
      "content": "...",
      "hidden": true,
      "versions": [
        { "content": "...", "created_at": "2026-06-24T10:02:05Z" }
      ],
      "active_version": 0,
      "created_at": "2026-06-24T10:02:05Z"
    }
  ]
}
```

| 字段 | 说明 |
|------|------|
| `system_prompt` | 系统提示词，可为空 |
| `messages[*].id` | 消息唯一 ID |
| `messages[*].role` | `"user"` / `"assistant"` / `"system"` |
| `messages[*].content` | 当前展示的内容（= versions[active_version].content） |
| `messages[*].hidden` | 是否隐藏，隐藏的轮次不发送给模型 |
| `messages[*].versions` | 仅 assistant 消息有，所有生成版本的数组 |
| `messages[*].active_version` | 当前选中的版本索引 |

### 操作映射

| 操作 | 实现 |
|------|------|
| 发送消息 | push user message → 调用 API → push assistant message |
| 重新生成 | 往 versions 追加新版本，active_version 指向新版本，更新 content |
| 切换版本 | 修改 active_version 和 content |
| 隐藏轮次 | user + assistant 的 hidden 设为 true |
| 构造 API 请求 | 遍历 messages，跳过 hidden，assistant 取 active_version 内容 |
| 删除对话 | BuntDB 删注册表 + 删 JSON 文件 |

写入策略：与其他 JSON 文件一致，写临时文件 → rename 原子替换。

## UI

侧边栏新增"对话"入口，左侧对话列表 + 右侧聊天区：

```text
┌──────────────────┬──────────────────────────────────┐
│ + 新建对话        │  [模型: gpt-4o ▼]               │
│                  ├──────────────────────────────────┤
│ 💬 Go 接口讨论   │  🧑 解释下 Go 的接口              │
│ 💬 项目架构设计   │                                  │
│ 💬 ...           │  🤖 Go 的接口是一种类型...        │
│                  │     [◀ 1/2 ▶] [🔄 重新生成]      │
│                  │                                  │
│                  │  🧑 再详细说说        [👁 隐藏]   │
│                  │                                  │
│                  │  🤖 ...                          │
│                  ├──────────────────────────────────┤
│                  │  [输入消息...]          [发送]    │
└──────────────────┴──────────────────────────────────┘
```

- 版本切换：assistant 回复下方显示 `◀ 1/2 ▶` 切换器
- 重新生成：assistant 回复下方按钮
- 隐藏：每轮右上角隐藏按钮，隐藏后灰显 + 折叠
- 流式输出：通过 Wails Events 推送 token，前端逐字渲染

## 待细化

- 模型配置管理（API 地址、Key、参数模板）
- 流式输出实现（Go 侧 SSE 解析 + Events 推送）
- 对话标题自动生成策略
- token 用量统计与展示
- 对话导出（Markdown / JSON）
- Bleve 索引结构（对话内容全文搜索）
- 后续可考虑：树状分叉模型（schema_version 升级迁移）
- 后续可考虑：结合本地收藏数据做 RAG
