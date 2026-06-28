# 前端 URL 模块开发计划

## 阶段一：基础设施搭建

- [ ] **1. 安装依赖**
  - Tailwind CSS v4 + `@tailwindcss/vite`
  - shadcn/ui 初始化（自动安装 class-variance-authority, clsx, tailwind-merge）
  - Zustand
  - react-hook-form + @hookform/resolvers + zod
  - lucide-react
  - 按需添加 shadcn 组件：button, input, badge, dialog, popover, command, collapsible, resizable, separator, dropdown-menu, tooltip, scroll-area

- [ ] **2. 主题系统**
  - `src/lib/themes.ts` — 四套主题预设注册（ink / stone / sage / indigo），HSL 色值定义
  - `src/index.css` — Tailwind 入口 + CSS 变量默认值（ink 主题）
  - `src/hooks/useTheme.ts` — 读取主题 key → 遍历写入 CSS 变量到 documentElement

- [ ] **3. 全局 store**
  - `src/stores/app.ts` — 当前页面（url / notes / media / tags / settings）、侧边栏折叠状态、当前主题 key

- [ ] **4. App 骨架布局**
  - `src/components/layout/Sidebar.tsx` — 5 个一级导航（URL、笔记、媒体、标签管理、设置），折叠/展开切换，选中态 `bg-muted`
  - `src/components/layout/ContentArea.tsx` — 内容区容器，根据当前页面渲染对应模块
  - `src/App.tsx` — 组装 Sidebar + ContentArea
  - `src/lib/utils.ts` — `cn()` 工具函数（shadcn 初始化生成）

## 阶段二：URL 模块 — 只读浏览

- [ ] **5. URL store**
  - `src/stores/url.ts` — sites 列表、selectedSiteId、selectedBookmarkId、bookmarks、loading 状态、分页信息
  - 方法：loadSites、loadMoreSites、selectSite、loadBookmarks、selectBookmark

- [ ] **6. 站点列表**
  - `src/features/url/URLPage.tsx` — 模块入口，双栏布局（左列表 + 右详情）
  - `src/features/url/SiteList.tsx` — 调用 `ListSites` 分页加载，每条显示站点名 + bookmark_count，无限滚动加载下一页
  - `src/components/EmptyState.tsx` — 通用空状态组件

- [ ] **7. 站点详情 + 书签视图**
  - `src/features/url/SiteDetail.tsx` — 上半部分：站点信息（icon、标题、URL、描述、标签）；下半部分：书签网格/列表视图切换
  - `src/features/url/BookmarkGrid.tsx` — 网格视图，用附件第一张图作封面，无附件显示标题文字卡片
  - `src/features/url/BookmarkListView.tsx` — 列表视图，每行显示标题 + URL 摘要

- [ ] **8. 书签详情**
  - `src/features/url/BookmarkDetail.tsx` — 完整书签信息（标题、URL、描述、标签、附件、状态、时间），返回按钮回到站点视图
  - `src/features/url/hooks.ts` — 模块 hooks，封装数据加载逻辑

## 阶段三：搜索与筛选

- [ ] **9. 搜索框**
  - `src/components/SearchBar.tsx` — 通用搜索框组件（防抖输入）
  - URL store 增加 searchMode、searchQuery、searchResults 状态
  - 接入 `SearchURL`，搜索结果按 `SiteWithBookmarks` 分组展示，左侧显示站点名 + 命中数
  - 搜索清空后恢复默认站点列表

- [ ] **10. 标签筛选**
  - `src/components/TagTreeFilter.tsx` — 标签树形筛选面板，按 `::` 分割渲染层级 + 折叠
  - 多选标签 AND 语义筛选，与搜索框取交集
  - URL store 增加 selectedTags 状态

## 阶段四：创建与编辑

- [ ] **11. 站点表单**
  - `src/features/url/SiteForm.tsx` — react-hook-form + zod schema 校验
  - 字段：title（必填）、url（必填）、description、tags、attachments
  - "抓取"按钮：调用 `FetchMetadata`，展示结果供用户确认回填
  - 创建模式调用 `CreateSite`，编辑模式调用 `UpdateSite`

- [ ] **12. 书签表单**
  - `src/features/url/BookmarkForm.tsx` — react-hook-form + zod schema 校验
  - 字段：url（必填，创建后不可改）、title（必填）、description、tags
  - 输入 URL 后调用 `LookupSiteByURL` 检查域名是否有对应站点
  - 无对应站点时提示用户先创建站点

- [ ] **13. 附件与文件上传**
  - `src/components/FileUpload.tsx` — 通用文件上传组件（拖拽 + 点击选择），读取文件为 base64
  - 调用 `UploadService.UploadFile`，按 scene 分发（site-icon / site-attachment / bm-attachment）
  - 附件列表展示：图片显示缩略图、视频显示抽帧缩略图、txt 显示文件名
  - 支持删除单个附件（`DeleteAttachment`）

- [ ] **14. 编辑模式**
  - `src/components/TagInput.tsx` — 标签输入组件（自动补全 + 多选 + 新建），基于 shadcn Command + Popover
  - 详情页增加"编辑"按钮，点击后整体切换为表单态（SiteForm / BookmarkForm），预填当前数据
  - 保存调用 Update 接口，取消恢复展示态

## 阶段五：删除与批量操作

- [ ] **15. 删除**
  - `src/components/ConfirmDialog.tsx` — 通用确认弹窗（shadcn AlertDialog）
  - 站点详情 / 书签详情增加删除按钮，二次确认后调用 `DeleteSite` / `DeleteBookmark`
  - 非空站点删除时展示后端返回的错误提示

- [ ] **16. 批量选择**
  - 书签列表/网格增加多选模式（checkbox），URL store 增加 selectedBookmarkIds 状态
  - 顶部操作栏：已选数量 + 全选/取消 + 批量操作按钮

- [ ] **17. 批量操作**
  - 批量删除：调用 `BatchDeleteBookmarks`，二次确认
  - 批量打标签：弹窗输入标签，调用 `BatchTagBookmarks`，追加到每条书签已有标签

## 阶段六：临时队列

- [ ] **18. 临时队列面板**
  - `src/features/url/TempQueue.tsx` — 队列列表，调用 `ListQueue` 加载
  - 每条显示 URL + 添加时间，支持单条删除（`DeleteQueueItem`）+ 清空（`ClearQueue`）
  - URL 模块顶部入口按钮，有未处理条目时显示角标数量
  - URL store 增加 queueItems、queueCount 状态

---

## 依赖关系

```
阶段一（1-4）→ 阶段二（5-8）→ 阶段三（9-10）
                                ↘ 阶段四（11-14）→ 阶段五（15-17）
                                ↘ 阶段六（18）
```

阶段三、四、六之间相互独立，可按需调整顺序。
