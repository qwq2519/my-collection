# 技术栈与存储

## 技术选型

| 层 | 选择 | 理由 |
|----|------|------|
| 框架 | Wails v3 | Go + Web 前端，桌面应用，纯 Go 无 CGO |
| 后端 | Go 1.25 | 高性能、编译简单、单二进制部署 |
| 前端 | React + TypeScript + Vite | 生态成熟、类型安全、构建快 |
| 前端 UI 组件 | shadcn/ui (Radix UI) | 组件代码在项目内可控，视觉现代简洁，按需添加 |
| 前端 CSS | Tailwind CSS v4 | 原子化 CSS，shadcn/ui 标配，样式写在 className 中，不用单独 CSS 文件 |
| 前端状态管理 | Zustand | 极轻量（~1KB），API 简单，每模块独立 store |
| 前端表单 | react-hook-form + zod | shadcn/ui 推荐组合，zod 做 schema 校验，类型安全 |
| 前端图标 | lucide-react | shadcn/ui 默认图标库，600+ 图标，tree-shakable |
| Markdown 编辑器 | @uiw/react-md-editor | 编辑+预览双模式，轻量，支持自定义渲染 |
| 虚拟滚动 | @tanstack/react-virtual | 列表和网格虚拟化，万级数据流畅滚动 |
| 主存储 | BuntDB | 内存 KV，自动持久化，纯 Go |
| 搜索 | Bleve v2 | 全文搜索引擎，倒排索引，纯 Go |
| 中文分词 | gse | 纯 Go 中文分词，无 CGO，为 Bleve 提供中文 analyzer |
| 目标平台 | Windows 桌面 | 当前仅面向 Windows，`build/` 下其他平台目录为 Wails v3 框架默认生成。纯本地单机，不考虑多设备同步 |

## 前端技术栈

### 路由方案

桌面应用不使用 URL 路由。侧边栏 5 个一级页面（URL、笔记、媒体、标签管理、设置）通过 Zustand store 管理当前页面状态。不引入 React Router。

### UI 组件库选型决策

候选：Ant Design、shadcn/ui + Tailwind CSS、Arco Design。

**选定 shadcn/ui + Tailwind CSS**：

| 优势 | 说明 |
|------|------|
| 代码可控 | 组件代码复制到项目内，不受上游版本 breaking changes 影响 |
| 视觉现代 | 默认风格简洁克制，适合工具类桌面应用 |
| 按需使用 | CLI 逐个添加组件（`npx shadcn@latest add button`），无冗余 |
| Tailwind CSS | 样式写在 className 中，不用在 CSS 文件和组件间跳转 |
| 生态活跃 | 基于 Radix UI，无障碍友好，社区方案丰富 |

| 需额外处理 | 方案 |
|-----------|------|
| 标签输入（自动补全+多选+新建） | 基于 shadcn Command + Popover 组合实现 |
| 标签树形展示（筛选面板） | 自定义递归组件，按 `::` 分割渲染缩进层级+折叠 |
| 可折叠分组列表（站点+书签） | shadcn Collapsible 组件 |
| 可调整分栏布局 | shadcn Resizable 组件（基于 react-resizable-panels） |

**不选 Ant Design**：包体积大、视觉偏企业后台风格、样式定制需覆盖 token、版本升级有 breaking changes 风险。
**不选 Arco Design**：社区生态和文档质量逊于 antd，且同样存在依赖上游维护的问题。

### 前端依赖清单

```text
# shadcn/ui 基础（初始化时自动安装）
tailwindcss, @tailwindcss/vite, class-variance-authority, clsx, tailwind-merge, lucide-react

# 表单
react-hook-form, @hookform/resolvers, zod

# 状态管理
zustand

# Markdown 编辑器
@uiw/react-md-editor

# 虚拟滚动
@tanstack/react-virtual
```

## 后端目录结构

三包分层：`model`（数据定义）→ `store`（存储访问）→ `service`（业务逻辑 + Wails 绑定）。依赖方向单向：`service → store → model`。

```text
internal/
├── model/                     # 领域模型 + 请求/响应类型（纯 struct，无依赖）
│   ├── site.go                # Site, CreateSiteReq, UpdateSiteReq, SiteListResult
│   ├── bookmark.go            # Bookmark, CreateBookmarkReq, BookmarkListResult
│   ├── queue.go               # QueueItem
│   ├── note.go                # Note, CreateNoteReq, NoteListResult
│   ├── media.go               # MediaFolder, MediaFile, MediaMeta
│   ├── tag.go                 # URLTag, MediaTag
│   └── common.go              # 通用类型：UploadFileReq, UploadFileResult, 事件载荷等
│
├── store/                     # 数据访问层（BuntDB + Bleve 协同写入）
│   ├── store.go               # Store 主结构体：初始化/关闭、backupMu、index_dirty 管理
│   ├── buntdb.go              # BuntDB 连接、自定义索引注册（idx:site_domain 等）
│   ├── search.go              # Bleve 初始化、gse analyzer、index mapping、RebuildIndex
│   ├── site.go                # 站点 CRUD + 域名唯一性校验
│   ├── bookmark.go            # 书签 CRUD + URL 去重 + bookmark_count 维护
│   ├── queue.go               # 临时队列 CRUD + 入队去重
│   ├── note.go                # 笔记 CRUD
│   ├── media.go               # media_meta.json / tree_hash.json 读写
│   └── tag.go                 # 标签注册表 CRUD + count 维护
│
├── service/                   # 业务逻辑层（Wails 绑定的 Service 结构体）
│   ├── url.go                 # URLService：站点 + 书签 + 临时队列
│   ├── note.go                # NoteService：笔记增删改查 + 图片管理
│   ├── media.go               # MediaService：文件夹管理、扫描触发、缩略图、标签
│   ├── tag.go                 # TagService：重命名、合并、删除、重算 count
│   ├── upload.go              # UploadService：统一文件上传、scene 路由、缩略图生成
│   ├── setting.go             # SettingService：配置读写、索引重建、备份导出
│   ├── scanner.go             # 媒体文件夹扫描：Merkle Tree 构建、比对、变化检测
│   ├── fetcher.go             # URL 元数据抓取：HTTP GET + HTML 解析 + icon 下载
│   └── asset.go               # Wails AssetHandler：/persist/ 路径映射 + 白名单 + 缓存头
│
└── util/                      # 通用工具函数（跨包复用）
    ├── urlutil.go             # URL 归一化、域名提取、输入校验
    ├── fileutil.go            # 原子写入（tmp→rename）、路径安全校验
    └── mdutil.go              # Markdown strip（Bleve 索引前预处理）
```

### 分层职责

| 层 | 包 | 职责 | 持有 |
|----|---|------|------|
| 数据定义 | `model` | 纯 struct：实体、请求、响应、事件载荷 | 无状态 |
| 数据访问 | `store` | 封装 BuntDB + Bleve 写入事务策略，屏蔽存储细节 | BuntDB 实例、Bleve 实例、backupMu |
| 业务逻辑 | `service` | 编排校验、存储调用、文件操作、事件推送；公开方法即前端 API | `*store.Store` 引用 |
| 工具函数 | `util` | 跨包复用的无状态工具：URL 处理、文件操作、文本处理 | 无状态 |

### 设计约束

- **service 内的辅助文件**（scanner、fetcher、asset）为同包内部实现，不对外暴露，仅被同包 Service 方法调用
- **util 包**为纯函数集合，可被 store 和 service 共同引用，无状态、无副作用
- **文件命名省略 `_service` / `_store` 后缀**，包名已表达层级含义
- **后续拆包原则**：当单个辅助文件膨胀超过 500 行时，再提取为独立 `internal/` 子包

### 初始化流程

`main.go` 负责组装：

```text
main.go
  1. store.New("persist")          → 打开 BuntDB + Bleve，注册索引
  2. 创建各 Service，注入 Store    → &service.URLService{Store: s}
  3. application.New(Options{      → 注册 Services + AssetHandler
       Services: [...],
       Assets: asset.NewHandler(),
     })
  4. app.Run()
```

## 前端目录结构

按功能模块组织（feature-based），与后端 Service 分层对应：

```text
frontend/src/
├── main.tsx                        # 入口
├── App.tsx                         # 根组件：侧边栏 + 内容区
├── index.css                       # Tailwind 入口 + CSS 变量（主题色等）
│
├── components/                     # 通用组件
│   ├── ui/                         # shadcn/ui 组件（CLI 自动生成到此目录）
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   └── ...
│   ├── layout/
│   │   ├── Sidebar.tsx             # 侧边栏导航
│   │   └── ContentArea.tsx         # 内容区容器
│   ├── TagInput.tsx                # 标签输入（自动补全+多选+新建）
│   ├── TagTreeFilter.tsx           # 标签树形筛选面板
│   ├── SearchBar.tsx               # 通用搜索框
│   ├── FileUpload.tsx              # 通用文件上传
│   ├── ConfirmDialog.tsx           # 通用确认弹窗
│   └── EmptyState.tsx              # 空状态占位
│
├── features/                       # 功能模块
│   ├── url/                        # URL 收藏模块
│   │   ├── URLPage.tsx             # 模块入口：搜索栏 + 列表 + 详情
│   │   ├── SiteList.tsx            # 站点分组折叠列表
│   │   ├── BookmarkItem.tsx        # 单条书签列表项
│   │   ├── SiteDetail.tsx          # 站点详情/编辑
│   │   ├── BookmarkDetail.tsx      # 书签详情/编辑
│   │   ├── SiteForm.tsx            # 站点表单
│   │   ├── BookmarkForm.tsx        # 书签表单
│   │   ├── TempQueue.tsx           # 临时队列面板
│   │   └── hooks.ts               # 模块 hooks（数据请求、操作）
│   ├── notes/
│   │   ├── NotesPage.tsx
│   │   ├── NoteList.tsx
│   │   ├── NoteEditor.tsx          # Markdown 编辑器封装
│   │   └── hooks.ts
│   ├── media/
│   │   ├── MediaPage.tsx
│   │   ├── MediaGrid.tsx           # 缩略图虚拟网格
│   │   ├── MediaCard.tsx           # 单个缩略图卡片
│   │   ├── MediaDetail.tsx         # 详情面板
│   │   ├── MediaFilter.tsx         # 筛选栏
│   │   └── hooks.ts
│   ├── tags/
│   │   ├── TagsPage.tsx
│   │   ├── TagList.tsx
│   │   ├── TagMergeDialog.tsx      # 合并弹窗
│   │   └── hooks.ts
│   └── settings/
│       ├── SettingsPage.tsx
│       ├── FolderManager.tsx       # 媒体文件夹管理
│       ├── IndexStatus.tsx         # 索引状态与重建
│       └── hooks.ts
│
├── stores/                         # Zustand stores（每模块独立）
│   ├── app.ts                      # 全局：当前页面、侧边栏状态
│   ├── url.ts                      # URL 模块：站点列表、书签、搜索状态
│   ├── note.ts                     # 笔记模块
│   ├── media.ts                    # 媒体模块
│   ├── tag.ts                      # 标签模块
│   └── setting.ts                  # 设置模块
│
├── hooks/                          # 全局通用 hooks
│   ├── useTheme.ts                 # 主题注入：读取主题 key → 写入 CSS 变量
│   ├── useWailsEvent.ts            # Wails Events 订阅封装
│   └── usePagination.ts            # 分页 + 无限滚动逻辑
│
├── lib/                            # 工具函数
│   ├── utils.ts                    # cn() 等工具（shadcn 初始化生成）
│   └── themes.ts                   # 主题预设注册中心（所有主题 HSL 色值）
│
└── types/                          # 前端专用类型
    └── index.ts                    # 页面枚举、筛选条件等（后端实体类型由 bindings/ 自动生成）
```

### 目录约定

| 目录 | 职责 | 谁维护 |
|------|------|--------|
| `components/ui/` | shadcn/ui 组件 | CLI 自动生成，可手动修改 |
| `components/` 其他 | 跨模块复用的业务组件 | 手动 |
| `features/{module}/` | 功能模块，组件 + hooks 同目录 | 手动 |
| `stores/` | 每模块独立 store，与后端 Service 对应 | 手动 |
| `bindings/` | Wails 自动生成的 TS 类型和服务调用 | `wails3 generate bindings` 自动生成，**不手动编辑** |
| `types/` | 前端独有类型（页面枚举、筛选条件等） | 手动 |

## 主题系统

### 架构

shadcn/ui 的所有组件通过 CSS 变量引用颜色（`bg-primary`、`text-muted-foreground` 等）。替换 CSS 变量的值即可切换整个 UI 配色，不需要改任何组件代码。

```text
src/lib/themes.ts          主题预设注册中心（所有主题的 HSL 色值）
src/stores/app.ts          Zustand 持有当前主题 key，持久化到 BuntDB
src/hooks/useTheme.ts      应用启动时读取主题并注入 CSS 变量
src/index.css              定义 CSS 变量默认值（ink 主题），运行时被 JS 覆盖
```

切换主题的数据流：

```text
用户在设置页选择主题
  → Zustand store 更新 themeKey
  → useTheme 从 themes 注册表取出对应色值
  → 遍历写入 document.documentElement.style.setProperty(...)
  → 所有 shadcn 组件通过 CSS 变量自动跟随变化
  → 持久化到 BuntDB（SettingService）
```

新增主题只需在 `themes.ts` 的注册表中添加一组 HSL 值，不需要改任何组件。

### 主题预设

四套内置主题，风格统一为低饱和度淡雅路线。默认主题：`ink`。

**CSS 变量语义说明：**

| 变量 | 用途 |
|------|------|
| `--background` / `--foreground` | 页面背景 / 正文文字 |
| `--primary` / `--primary-foreground` | 强调色（按钮、链接）/ 强调色上的文字 |
| `--secondary` / `--secondary-foreground` | 次要按钮背景 / 次要按钮文字 |
| `--muted` / `--muted-foreground` | 弱化区域背景 / 弱化文字（占位符、次要信息） |
| `--accent` / `--accent-foreground` | 悬停/选中态背景 / 对应文字 |
| `--destructive` / `--destructive-foreground` | 危险操作（删除） |
| `--border` | 边框 |
| `--input` | 输入框边框 |
| `--ring` | 焦点环（键盘导航） |
| `--card` / `--card-foreground` | 卡片背景 / 卡片文字（与 background 通常一致） |
| `--popover` / `--popover-foreground` | 浮层背景 / 浮层文字 |
| `--sidebar-*` | 侧边栏专用变量（与主背景微妙区分） |

#### A. 水墨（ink）— 无彩色

纯灰阶，最克制。强调色为深灰，整体像 iA Writer、Typora 的气质。

| 变量 | HSL 值 |
|------|--------|
| `--background` | `0 0% 100%` |
| `--foreground` | `220 10% 12%` |
| `--primary` | `220 10% 28%` |
| `--primary-foreground` | `0 0% 98%` |
| `--secondary` | `220 5% 96%` |
| `--secondary-foreground` | `220 10% 25%` |
| `--muted` | `220 5% 96%` |
| `--muted-foreground` | `220 5% 46%` |
| `--accent` | `220 5% 94%` |
| `--accent-foreground` | `220 10% 12%` |
| `--destructive` | `0 60% 50%` |
| `--destructive-foreground` | `0 0% 98%` |
| `--border` | `220 5% 90%` |
| `--input` | `220 5% 88%` |
| `--ring` | `220 10% 28%` |
| `--card` | `0 0% 100%` |
| `--card-foreground` | `220 10% 12%` |
| `--popover` | `0 0% 100%` |
| `--popover-foreground` | `220 10% 12%` |
| `--sidebar-background` | `220 5% 97%` |
| `--sidebar-foreground` | `220 10% 12%` |
| `--sidebar-accent` | `220 5% 93%` |
| `--sidebar-accent-foreground` | `220 10% 12%` |
| `--sidebar-border` | `220 5% 91%` |

#### B. 暖石（stone）— 暖灰棕

带棕色调的暖灰，像牛皮纸或旧书页，和"收藏"的语义契合。

| 变量 | HSL 值 |
|------|--------|
| `--background` | `40 20% 99%` |
| `--foreground` | `30 10% 12%` |
| `--primary` | `30 30% 33%` |
| `--primary-foreground` | `40 20% 98%` |
| `--secondary` | `35 15% 95%` |
| `--secondary-foreground` | `30 12% 25%` |
| `--muted` | `35 12% 95%` |
| `--muted-foreground` | `30 8% 46%` |
| `--accent` | `35 15% 93%` |
| `--accent-foreground` | `30 10% 12%` |
| `--destructive` | `0 55% 48%` |
| `--destructive-foreground` | `40 20% 98%` |
| `--border` | `35 10% 89%` |
| `--input` | `35 10% 87%` |
| `--ring` | `30 30% 33%` |
| `--card` | `40 20% 99%` |
| `--card-foreground` | `30 10% 12%` |
| `--popover` | `40 18% 100%` |
| `--popover-foreground` | `30 10% 12%` |
| `--sidebar-background` | `38 18% 96%` |
| `--sidebar-foreground` | `30 10% 12%` |
| `--sidebar-accent` | `35 14% 92%` |
| `--sidebar-accent-foreground` | `30 10% 12%` |
| `--sidebar-border` | `35 10% 90%` |

#### C. 青竹（sage）— 灰绿

低饱和灰绿色，安静自然，长时间看屏幕舒适。

| 变量 | HSL 值 |
|------|--------|
| `--background` | `150 10% 99%` |
| `--foreground` | `150 8% 12%` |
| `--primary` | `155 25% 30%` |
| `--primary-foreground` | `150 10% 98%` |
| `--secondary` | `150 8% 95%` |
| `--secondary-foreground` | `150 10% 25%` |
| `--muted` | `150 6% 95%` |
| `--muted-foreground` | `150 5% 46%` |
| `--accent` | `155 10% 93%` |
| `--accent-foreground` | `150 8% 12%` |
| `--destructive` | `0 55% 48%` |
| `--destructive-foreground` | `150 10% 98%` |
| `--border` | `150 5% 89%` |
| `--input` | `150 5% 87%` |
| `--ring` | `155 25% 30%` |
| `--card` | `150 10% 99%` |
| `--card-foreground` | `150 8% 12%` |
| `--popover` | `150 8% 100%` |
| `--popover-foreground` | `150 8% 12%` |
| `--sidebar-background` | `150 8% 97%` |
| `--sidebar-foreground` | `150 8% 12%` |
| `--sidebar-accent` | `152 8% 92%` |
| `--sidebar-accent-foreground` | `150 8% 12%` |
| `--sidebar-border` | `150 5% 90%` |

#### D. 靛青（indigo）— 灰蓝

低饱和蓝灰色，经典工具类应用配色方向，稳妥有辨识度。

| 变量 | HSL 值 |
|------|--------|
| `--background` | `220 14% 99%` |
| `--foreground` | `224 12% 12%` |
| `--primary` | `224 30% 38%` |
| `--primary-foreground` | `220 14% 98%` |
| `--secondary` | `220 10% 96%` |
| `--secondary-foreground` | `224 14% 25%` |
| `--muted` | `220 8% 96%` |
| `--muted-foreground` | `220 8% 46%` |
| `--accent` | `220 10% 94%` |
| `--accent-foreground` | `224 12% 12%` |
| `--destructive` | `0 60% 50%` |
| `--destructive-foreground` | `220 14% 98%` |
| `--border` | `220 8% 90%` |
| `--input` | `220 8% 88%` |
| `--ring` | `224 30% 38%` |
| `--card` | `220 14% 99%` |
| `--card-foreground` | `224 12% 12%` |
| `--popover` | `220 12% 100%` |
| `--popover-foreground` | `224 12% 12%` |
| `--sidebar-background` | `220 10% 97%` |
| `--sidebar-foreground` | `224 12% 12%` |
| `--sidebar-accent` | `220 9% 93%` |
| `--sidebar-accent-foreground` | `224 12% 12%` |
| `--sidebar-border` | `220 8% 91%` |

### 设计约束

- 四套主题色相不同但风格统一：低饱和度、高明度背景、灰阶主导
- 所有主题的 `--destructive` 均为红色系，保持危险操作的直觉辨识
- `--sidebar-background` 比 `--background` 略深 2-3%，产生微妙层次但不割裂
- 后续如需暗色模式，在 `ThemePreset.colors` 中增加 `dark` 字段，架构无需改动

## 存储架构

```text
┌─────────────────────────────────────────────────────────┐
│                   Go 应用 (Wails v3)                     │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  BuntDB (main.db)          每文件夹 JSON         Bleve  │
│  - URL/笔记/站点/队列     - 媒体标签与元数据    - 搜索索引│
│  - 文件夹注册表           - Merkle Tree 快照    - 可重建  │
│  - 内存常驻               - 启动时全量加载               │
│  - 自动持久化到单文件      - 原子写入 JSON 文件           │
│                                                         │
│  persist/main.db     persist/media-folders/{id}/  persist/search.bleve/│
└─────────────────────────────────────────────────────────┘
```

| 组件 | 包 | 定位 |
|------|----|------|
| BuntDB | `github.com/tidwall/buntdb` | 主存储，内存 KV，自动持久化 |
| Bleve v2 | `github.com/blevesearch/bleve/v2` | 全文搜索，倒排索引 |
| gse | `github.com/go-ego/gse` | 中文分词，为 Bleve 提供 CJK analyzer |
| xxHash | `github.com/cespare/xxhash/v2` | Merkle Tree 变化检测用的高速 hash |

四者均为纯 Go 实现，无 CGO 依赖。

## 为什么选 BuntDB + Bleve

### 需求约束

| 需求 | 说明 |
|------|------|
| 高性能读 | 收藏查询、标签检索需要毫秒级响应 |
| 减少磁盘 IO | 避免频繁读写磁盘，尽量内存操作 |
| 标签精确匹配 | 多标签组合精确查询，不做前缀模糊（`前端::*`） |
| 灵活搜索 | 媒体标签细粒度，接近全文检索 |
| 笔记内容搜索 | 支持对 note 正文全文搜索 |
| 部署简单 | 桌面应用，不依赖外部服务 |

### 候选方案对比

| 方案 | 类型 | 特点 |
|------|------|------|
| SQLite + FTS5 | 嵌入式关系数据库 | 生态成熟、SQL 查询、FTS5 全文搜索 |
| **BuntDB + Bleve** | 内存 KV + 搜索引擎 | 纯 Go、内存快查、倒排索引 |
| Badger + Bleve | 磁盘 KV + 搜索引擎 | 纯 Go、大数据量、LSM-tree |
| go-memdb + JSON 刷盘 | 内存数据库 + 手动持久化 | 极快、多索引、需自己处理刷盘 |

**不选 SQLite**：Go 调用需 CGO（或纯 Go 翻译版性能降 20-30%），增加编译复杂度；SQL 表达力对本项目超配。

**不选 Badger**：数据量万级以下，全内存方案更优；LSM-tree 有写放大。

### KV/JSON vs 关系型

**获得的优势：**

| 方面 | 说明 |
|------|------|
| 无需 migration | 加字段直接写入 |
| 半结构化友好 | 不同实体可有不同字段 |
| 性能可预测 | 无 SQL 解析开销，内存直读 |
| 部署简单 | 一个文件就是整个数据库 |
| 和 Go struct 对应 | JSON ↔ struct 直接序列化 |

**承担的代价：**

| 方面 | 应对 |
|------|------|
| 多条件查询要自己写 | 提前设计索引，复杂查询走 Bleve |
| 关联查询是手动的 | 用 key 前缀模拟层级关系 |
| 数据完整性靠代码 | Store 层封装校验逻辑 |
| 无标准查询语言 | 抽象 Repository 接口隔离存储细节 |

**为什么可接受**：实体类型少（3 种），查询模式有限且可预知，标签用 JSON 数组比关联表更自然，数据量不大。

## persist 目录

所有持久化数据集中存放，是唯一需要备份的目录。完整目录结构和各文件格式详见 [数据结构](./data-structures.md#persist-目录结构)。

**目录位置**：`persist/` 位于可执行文件所在目录下（即 `{exe_dir}/persist/`）。开发阶段 `wails3 dev` 在项目根目录运行，因此 `persist/` 生成在项目根目录下。

**备份方式**：通过应用内"导出备份"功能，打包 `persist/` 目录为 zip 文件。

- `main.db` + `media-folders/*/media_meta.json` 是核心数据
- `search.bleve/` 可从上述数据重建，**备份时跳过**以减小体积
- `url-assets/icons/` 可通过重新抓取恢复
- `url-assets/covers/` + `url-assets/attachments/` 是用户上传数据
- `thumbnails/` 可从源文件重新生成，**备份时跳过**以减小体积

**不建议直接拷贝 `persist/`**：BuntDB 正在写入时拷贝可能得到不一致的文件。应使用应用内备份功能确保一致性。

### 备份一致性机制

后端采用双锁分离策略，分别管理备份协调和索引状态保护。

**Store.backupMu**（备份协调锁 `sync.RWMutex`）：借用 RWMutex 的"多读单写"机制实现"多写单备份"——写入操作取 RLock（允许并发），备份和全量索引重建取 Lock（独占）。**读操作不取 backupMu**，备份期间搜索不受影响。

**IndexManager.mu**（索引状态锁 `sync.RWMutex`）：保护索引实例和状态的并发安全。索引读写取 RLock，重建和关闭取 Lock。独立于 backupMu。

```text
Store.backupMu (sync.RWMutex)         IndexManager.mu (sync.RWMutex)
         /                \                    /               \
  写入操作: RLock()    备份/重建: Lock()   索引读写: RLock()  重建/关闭: Lock()
  (允许并发写入)      (独占，阻塞写入)    (允许并发搜索)    (阻塞索引操作)
  读操作: 不加锁                        
```

**普通写入操作**（BuntDB 写入、Bleve 索引更新等）：

```go
backupMu.RLock()
defer backupMu.RUnlock()
// 执行写入（索引操作由 IndexManager.mu 内部保护）
```

**读操作**（搜索查询）：

```go
// 不取 backupMu，直接调用 IndexManager
// IndexManager 内部取 mu.RLock 保证索引状态安全
return s.idx.Search(req)
```

**备份/全量重建**：

```go
backupMu.Lock()
defer backupMu.Unlock()
// 阻塞所有写入，但搜索不受影响（走 IndexManager.mu）
// 重建期间搜索会因 IndexManager 状态为 Rebuilding 返回错误
```

**备份内容**（zip 中包含）：

| 包含 | 说明 |
|------|------|
| `main.db` | 核心数据 |
| `media-folders/*/media_meta.json` | 媒体元数据 |
| `media-folders/*/tree_hash.json` | Merkle Tree 快照 |
| `url-assets/` | 图标、封面、附件 |
| `note-images/` | 笔记图片 |

| 跳过 | 理由 |
|------|------|
| `search.bleve/` | 可从 main.db + media_meta.json 重建 |
| `media-folders/*/thumbnails/` | 可从源文件重新生成 |

**恢复**：解压 zip 覆盖 `persist/` 目录，启动时自动检测 `search.bleve/` 缺失或 `index_dirty`，触发重建。

## 静态资源访问

前端需要访问 `persist/` 目录下的本地文件（站点图标、笔记图片、媒体缩略图）。采用 **Wails AssetHandler** 方案。

### 方案：Wails AssetHandler

Wails v3 的 `Application` 支持配置自定义 `AssetHandler`，当请求路径在前端打包产物（`embed.FS`）中未命中时，转发给该 handler。将 `/persist/...` 路径映射到本地文件系统。

**路径职责划分**：

后端返回给前端的所有资源路径（图标、封面、缩略图、预览等）必须是**完整可访问路径**，前端直接用作 `<img src>` 或 `<video src>`，**不做任何路径拼接**。路径构建逻辑全部由后端负责。

```html
<!-- 前端直接使用后端返回的路径，无需拼接 -->
<img src="/persist/url-assets/icons/github.com.png" />
<img src="/persist/url-assets/covers/a1b2c3d4.gif" />
<img src="/persist/note-images/c9d0e1f2/a3f2b8c1e5d7f9ab.png" />
<img src="/persist/media-folders/550e8400/thumbnails/a3f2b8c1e5d7f9ab.jpg" />
```

笔记 body 中存储的图片路径（`note-images/xxx/img.png`）渲染时加 `/persist/` 前缀即可（这是唯一的前端路径处理）。

**安全约束**：

- 路径白名单：只允许访问 `persist/` 下的 `url-assets/`、`note-images/`、`media-folders/` 子目录
- 目录遍历防护：校验解析后的绝对路径仍在 `persist/` 目录内（防止 `../` 攻击）

**缓存策略**：

- 图标和缩略图变更频率低，设置 `Cache-Control` 响应头，减少重复读盘
- 笔记图片按内容 hash 命名，天然适合长期缓存

### 候选方案对比（决策记录）

| | AssetHandler（已选） | Go 方法返回 Base64 | 本地 HTTP 服务器 | 自定义 URL Scheme |
|---|---|---|---|---|
| 实现复杂度 | 低 | 中 | 中 | 高 |
| Markdown 兼容 | 好（标准 `src`） | 差（需拦截替换所有图片引用） | 好 | 需适配非标准协议 |
| 列表性能 | 好（浏览器并行加载） | 差（大量串行 Go 调用） | 好 | 好 |
| 缓存 | 可配置 HTTP 缓存头 | 无浏览器缓存 | 天然支持 | 取决于 WebView 实现 |
| 安全性 | 需路径校验 | Go 层校验，较安全 | 需路径校验 + CORS | 天然隔离 |
| 架构契合度 | 高（Wails 原生机制） | 中 | 中（已规划 HTTP 扩展可复用） | 待验证 WebView2 支持 |

**不选 Base64**：列表页（媒体网格、书签列表）图片多，逐张异步调用 Go 方法性能差；Base64 编码膨胀约 33%；Markdown 预览中 `![](path)` 无法直接渲染。

**不选 HTTP 服务器**：引入端口管理、CORS 配置、独立生命周期管理，对纯本地桌面工具过重。

**不选自定义 Scheme**：Wails v3 对自定义协议的支持依赖底层 WebView，跨平台行为可能不一致，且 Markdown 编辑器需适配非标准协议。

## 职责分工

| 操作 | 走哪个组件 |
|------|-----------|
| 增删改收藏/笔记 | BuntDB → 同步更新 Bleve |
| 临时队列读写 | BuntDB（前缀扫描 `queue:*`，仅存 URL） |
| 批量删除 URL | BuntDB 事务 + 清理 Bleve |
| 按 ID 精确查询 | BuntDB |
| 标签筛选（多选精确匹配） | Bleve keyword 精确匹配 |
| 标签注册表（列表、count） | BuntDB 前缀扫描 `url_tag:*` / `media_tag:*` |
| 笔记全文搜索 | Bleve |
| 按站点列出所有收藏 | BuntDB 自定义索引 `idx:bm_site` |
| 文件夹注册表 | BuntDB |
| 媒体元数据增删改 | 每文件夹 media_meta.json → 同步更新 Bleve |
| 媒体标签搜索 | Bleve |
| 媒体文件变化检测 | 每文件夹 tree_hash.json |

## 写入事务策略

BuntDB 和 Bleve 不具备跨组件原子性。策略：**BuntDB 先写先提交，Bleve 后写；Bleve 失败时不回滚 BuntDB，而是标记索引脏并通知前端。**

```text
Store.Create(entity):
  1. BuntDB.Update(tx) → 写入 KV（提交）
  2. Bleve.Index(doc)  → 更新索引
  3. 若 Bleve 失败 → BuntDB 数据已持久化，标记 index_dirty，返回结果 + 索引异常警告
```

| 场景 | 处理 |
|------|------|
| BuntDB 写入失败 | 事务回滚，Bleve 不执行，返回错误 |
| BuntDB 成功、Bleve 失败 | 数据已写入，标记 `index_dirty`，返回成功 + 索引异常警告；前端展示"索引异常"状态 |
| 应用崩溃 | BuntDB AOF 保证持久化，启动时检测 `index_dirty` 标志自动重建 Bleve |

**索引脏标志（`index_dirty`）**：

- 存储在 BuntDB 中：`meta:index_dirty → true`
- Bleve 写入失败时置为 `true`
- 前端可查询该标志，在 UI 顶部展示"搜索索引异常，建议重建"提示
- 用户可在设置页手动触发重建；应用启动时若检测到该标志也自动重建
- 重建完成后清除标志

**媒体元数据写入**（独立于 BuntDB 事务）：

```text
MediaStore.Update(folder_id, changes):
  1. 更新内存中的 files 数据
  2. 原子写入 media_meta.json (write tmp → rename)
  3. Bleve.Index(docs) → 更新索引
  4. 若 Bleve 失败 → media_meta.json 已持久化，标记 index_dirty
```

## 架构约束

1. **Store 层封装写入**：每次写入先完成 BuntDB 持久化，再同步更新 Bleve；Bleve 失败不阻塞写入
2. **Repository 接口隔离**：上层不直接依赖 BuntDB/Bleve API
3. **Bleve 可重建**：从 main.db + media_meta.json 全量灌入，启动时自动检测 `index_dirty` 触发重建
4. **索引状态可观测**：`index_dirty` 标志暴露给前端，用户可感知并手动触发重建
5. **数据格式 JSON**：统一使用 JSON，便于调试和导出
6. **媒体数据隔离**：按文件夹独立存储，不与 URL/笔记耦合

## 统一文件上传

前端提供通用的图片/视频上传组件（原子化、可复用），所有模块的文件上传统一调用后端的 `UploadFile` 接口。后端根据前端传递的 `scene` 参数将文件分发到 `persist/` 下不同子目录。

**接口签名：**

```go
// scene: 上传场景，决定文件存储路径
// entity_id: 关联实体 ID（站点/书签/笔记的 UUID）
// file: 文件内容
UploadFile(req UploadFileReq) (*UploadFileResult, error)
```

**scene 路由表：**

| scene | 存储路径 | 使用场景 |
|-------|---------|---------|
| `site-icon` | `persist/url-assets/icons/{domain}.{ext}` | 站点图标 |
| `site-cover` | `persist/url-assets/covers/{entity_id}.{ext}` | 站点封面 |
| `bm-cover` | `persist/url-assets/covers/{entity_id}.{ext}` | 书签封面 |
| `site-attachment` | `persist/url-assets/attachments/{entity_id}/{filename}` | 站点附件 |
| `bm-attachment` | `persist/url-assets/attachments/{entity_id}/{filename}` | 书签附件 |
| `note-image` | `persist/note-images/{entity_id}/{hash}.{ext}` | 笔记图片 |

**支持格式：**

| 类型 | 格式 |
|------|------|
| 图片 | JPG/JPEG、PNG、GIF、WebP、BMP、AVIF、SVG |
| 视频 | MP4、MKV、AVI、MOV、WebM、WMV、FLV |
| 文本 | TXT（仅附件场景） |

后端校验文件扩展名是否在 scene 允许范围内，不合法则拒绝。返回值包含存储后的文件路径，供前端写入实体数据。

## 接口约定

Wails 将 Go 结构体的公开方法直接暴露给前端调用，无需手写 REST API。接口文档以 **Go 函数注释**为主，不单独维护接口文档文件（Go 注释会自动生成 TypeScript JSDoc）。

### Service 层

每个 Service 结构体对应一个功能模块，公开方法即前端可调用的接口：

| Service | 职责 |
|---------|------|
| `URLService` | 站点/书签/临时队列的增删改查 |
| `NoteService` | 笔记的增删改查、图片管理 |
| `MediaService` | 媒体文件夹管理、扫描、缩略图、标签 |
| `TagService` | 标签管理（重命名/合并/删除/重算 count） |
| `UploadService` | 统一文件上传，按 scene 分发到 persist 子目录 |
| `SettingService` | 应用设置、索引重建 |

### 方法签名约定

- 方法上方的 Go 注释说明：用途、参数含义、返回值、可能的错误
- 入参和返回值统一使用 struct，便于 Wails 生成 TypeScript 类型绑定
- 错误通过 `error` 返回，前端侧为 rejected Promise
- 指针字段表示可选（`*T` → TypeScript `T | null`）

**返回模式：**

| 操作类型 | 返回签名 | 示例 |
|---------|---------|------|
| 单条详情 | `(*Entity, error)` | `GetBookmark(id string) (*Bookmark, error)` |
| 分页列表 | `(*XxxListResult, error)` | `ListBookmarks(req BookmarkListReq) (*BookmarkListResult, error)` |
| 创建/更新 | `(*Entity, error)` | `CreateBookmark(req CreateBookmarkReq) (*Bookmark, error)` |
| 删除 | `error` | `DeleteBookmark(id string) error` |

列表接口返回摘要字段（title、domain、tags 等），单条详情接口返回完整信息（含 description、cover、attachments 等）。

**分页请求/响应结构：**

```go
// 请求 — 每种实体各定义一个，包含该实体特有的筛选字段
// 排序固定为 updated_at 降序，不暴露排序参数
type BookmarkListReq struct {
    Page     int      `json:"page"`
    PageSize int      `json:"page_size"`
    Search   string   `json:"search,omitempty"`
    Tags     []string `json:"tags,omitempty"`
    SiteID   string   `json:"site_id,omitempty"`
}

// 响应 — 每种实体各定义一个 ListResult
type BookmarkListResult struct {
    Items   []Bookmark `json:"items"`
    Total   int        `json:"total"`
    HasMore bool       `json:"has_more"`
}
```

### 异步通知（Events）

Service 方法用于请求-响应式调用。异步通知走 Wails Events 推送：

| 事件名 | 时机 | 数据 |
|--------|------|------|
| `index:warning` | Bleve 写入失败 | 错误信息字符串 |
| `media:scan-progress` | 媒体扫描进行中 | `{ folder_id, scanned, total }` |
| `media:scan-complete` | 媒体扫描完成 | `{ folder_id, added, removed, modified }` |

前端通过 `Events.On("event-name", callback)` 订阅。

### 后续 HTTP 接口

后续如需开放少量 HTTP 接口（如手机 share URL 到本机），不改动 Service 层，加一个 HTTP 薄壳：

```text
┌──────────────────────────────────┐
│        前端 (Wails Bridge)        │
└──────────────┬───────────────────┘
               │
┌──────────────▼───────────────────┐
│          Service 层               │  ← 核心逻辑，唯一实现
└──────────────▲───────────────────┘
               │
┌──────────────┴───────────────────┐
│    HTTP Handler（标准库 net/http） │  ← 薄壳，仅解析请求 + 序列化响应
│    监听 127.0.0.1:端口            │
└──────────────────────────────────┘
```

- HTTP handler 直接调用 Service 方法，无需重复业务逻辑
- 响应格式：直接 JSON 序列化 Service 返回的 struct，error 映射为 HTTP status code
- 接口数量少，用标准库即可，不引入框架
- 默认只监听本机（`127.0.0.1`），后续需局域网访问再开放

## 部署与扩展

- **纯本地单机**桌面应用
- 后续可选：开放 HTTP 接口，支持从手机 share URL 到本机
- 始终单机部署，不考虑多设备同步

## 暂不考虑

- 浏览器书签导入/导出
- 全局快捷键唤起
- 多设备同步
