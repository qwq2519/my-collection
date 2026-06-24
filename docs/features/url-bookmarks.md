# URL 收藏

## 需求

### 数据结构

- **站点（Site）**：手动收藏，代表一个网站
- **网址（Bookmark）**：具体 URL，归属于已有站点；若域名无对应站点则进入临时队列

站点和书签功能类似，都有 title、icon、description、tags、cover、attachments。区别在于展示时书签按 domain 归组到站点下。

完整字段定义和存储格式详见 [数据结构 - site](../global/data-structures.md#site--站点) 和 [数据结构 - bm](../global/data-structures.md#bm--书签url)。

### 行为规则

- 用户先收藏一个站点
- 后续添加的 URL 由系统按域名自动归组到对应站点
- **添加 URL 时，若域名无对应站点则进入临时队列**
- **URL 去重**：添加 URL 时用归一化后的 URL（`normalized_url`）比对，已存在则拒绝并提示"该 URL 已收藏"
- 站点和书签各自独立打 tag，互不继承
- 检索支持：title、域名、tag

### 临时队列

- 添加 URL 时若域名无对应站点，URL 自动进入临时队列
- 队列作为**临时备忘**，只保留 URL 和基本元数据，不参与站点分组视图
- 用户可在队列中查看和删除条目
- 如需正式收藏，用户手动创建对应站点和书签
- 队列入口在 URL 模块顶部，有未处理条目时显示角标提示

### 编辑

- 右侧详情区提供**"编辑"按钮**，点击后整体切换为表单态
- 可编辑字段：title、description、icon、cover、tags、attachments（完整字段定义见 [数据结构](../global/data-structures.md#site--站点)）
- **封面（cover）**：编辑表单中支持上传图片作为封面，列表页主展示图；存储在 `persist/url-assets/covers/{entity_id}.{ext}`
- **附件（attachments）**：编辑表单中支持上传附件文件，每个附件可添加标注（label）；存储在 `persist/url-assets/attachments/{entity_id}/`
- 编辑完成后点击"保存"提交，或"取消"放弃修改
- 站点和书签的编辑交互一致

### 删除

- **非空站点不可删除**：站点下仍有书签时，删除操作返回错误并拒绝执行，需先删除所有书签后才能删除站点
- 站点下无书签时（`bookmark_count == 0`）可删除空站点
- 支持删除单条书签
- 支持**批量选择删除**：列表中多选书签后批量删除
- 删除操作需二次确认
- **bookmark_count 维护**：添加或删除书签时同步更新所属站点的 `bookmark_count`，详见 [数据结构 - site](../global/data-structures.md#site--站点)
- **标签 count 维护**：删除书签或站点时，遍历其 tags 数组，递减对应 url_tag 注册表的 count（批量删除时先汇总再一次性更新）

### URL 存活状态

- `status` 字段（`"alive"` / `"dead"`）仅用于书签详情页展示，**不参与搜索、筛选或排序**
- 新添加的 URL 默认状态为 `"alive"`——用户手动添加时已确认页面可访问，无需额外检测
- 暂不做批量检测，后续考虑引入任务队列异步检测
- 标记失效链接（404、超时、域名无法解析等）
- 不做自动定时检测

### 数据规模

- URL：万级上限
- Tag：千级上限

## 设计决策

### 域名匹配规则

按**完整域名（含子域名）**匹配，对 `www` 前缀做归一化（自动去除 www）。

- `github.com`、`gist.github.com`、`docs.github.com` 视为不同站点
- `www.example.com` 和 `example.com` 视为同一站点（归一化后都是 `example.com`）
- **不区分协议**：`http://` 和 `https://` 视为同一站点，统一以域名为准
- **路径不参与站点划分**：如 `notion.site/userA` 和 `notion.site/userB` 均归入 `notion.site` 站点
- **拒绝 localhost 和 IP 地址**：`localhost`、`127.0.0.1`、`192.168.x.x` 等地址不允许作为收藏，添加时直接拒绝并提示

### 存储

URL 和站点数据存储在 BuntDB 中，tag 搜索走 Bleve 索引。详见 [技术栈与存储](../global/tech-stack.md)。

### 元数据获取

添加/编辑站点或书签时，表单旁提供**"抓取"按钮**，输入 URL 后可抓取页面元数据作为填写参考。

**抓取流程：**

1. 用户在表单中输入 URL，点击"抓取"
2. 后端发 HTTP GET 请求，解析 HTML 返回结构化 JSON
3. 抓取结果展示在表单旁边，用户手动选择需要的字段复制到表单中
4. 抓取失败（超时/反爬/网络异常）时提示用户手动填写

**抓取字段（后端可扩展）：**

| 字段 | 来源 | 说明 |
|------|------|------|
| title | `<title>` 或 `og:title` | 页面标题 |
| description | `<meta name="description">` 或 `og:description` | 页面描述 |
| icon | `<link rel="icon">` 或 `{domain}/favicon.ico` | 页面图标 |
| og_image | `og:image` | Open Graph 封面图 |

后续可按需扩展更多 meta 标签（keywords、author、canonical 等）。

**设计原则：**

- 抓取是**辅助工具**，不自动填充表单，由用户决定使用哪些字段
- 超时上限 10 秒，不阻塞 UI（异步请求）
- 不缓存抓取结果，每次重新请求

## UI

双栏布局：左侧列表 + 右侧详情。

**默认状态：按站点分组折叠**

```text
🔍 [搜索...]  [站点筛选 ▼]  [临时队列 ③]
┌──────────────────────┬─────────────────────────┐
│ ▼ GitHub (3)         │  标题: golang/go         │
│   ├ 📌 golang/go     │  URL: github.com/go...  │
│   ├ 📌 react         │  站点: GitHub           │
│   └ 📌 vscode        │  Tags: #Go #开源         │
│ ▶ React 官网 (5)     │  描述: Go 语言主仓库      │
│ ▼ MDN (2)            │  添加时间: 2026-06-22    │
│   ├ 📌 CSS Grid 指南  │  [删除]                 │
│   └ 📌 Fetch API     │                         │
└──────────────────────┴─────────────────────────┘
```

- 站点作为可展开/折叠的分组头，旁边显示 URL 数量
- 点击具体 URL 时右侧展示详情
- 单条 URL 详情区域提供删除按钮

**搜索状态：平铺展示**

搜索时打破分组，结果平铺显示，每条标注所属站点：

```text
🔍 [go...]  [站点筛选 ▼]  [临时队列 ③]
┌──────────────────────┬─────────────────────────┐
│ 📌 golang/go         │  标题: golang/go         │
│    github.com  #Go   │  URL: github.com/go...  │
│ 📌 Go by Example    │  站点: GitHub           │
│    gobyexample.com   │  Tags: #Go #开源         │
│ ...                  │  ...                    │
└──────────────────────┴─────────────────────────┘
```

搜索清空后恢复分组视图。

子功能（站点筛选等）作为内容区顶部的筛选控件。存活状态仅在书签详情页展示，不作为列表筛选条件。
