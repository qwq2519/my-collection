/**
 * URL 收藏模块的全局状态管理（Zustand store）。
 *
 * ┌─────────────────────────────────────────────────────────┐
 * │  Zustand 简介（给 Go 工程师）                            │
 * │                                                         │
 * │  Zustand 类似 Go 中的全局 struct + mutex：               │
 * │  - 状态字段 = struct fields                              │
 * │  - action 方法 = struct methods                          │
 * │  - set() = 加锁后修改字段                                │
 * │  - get() = 读取当前快照                                  │
 * │                                                         │
 * │  组件通过 useURLStore((s) => s.xxx) 订阅特定字段，       │
 * │  字段变化时 React 自动重新渲染该组件（类似 pub/sub）。    │
 * │  每个 (s) => s.xxx 叫做 selector，只订阅需要的字段，     │
 * │  避免不相关字段变化导致不必要的重渲染。                    │
 * └─────────────────────────────────────────────────────────┘
 *
 * 本 store 管理 URL 收藏模块的三大数据域：
 *
 * 1. 站点列表（左栏） — sites*, 分页加载 + 无限滚动
 * 2. 详情面板（右栏） — detailView, currentSite, bookmarks*, currentBookmark
 * 3. 搜索/筛选       — search*, selectedTags
 *
 * 数据流：
 *
 *   用户操作          →  action 方法     →  调用后端 API   →  set() 更新状态  →  UI 自动刷新
 *   点击侧边栏"URL"   →  loadSites()     →  ListSites      →  sites=[...]     →  SiteList 重渲染
 *   点击某站点         →  selectSite(id)  →  GetSite +      →  currentSite,    →  SiteDetail 重渲染
 *                                           ListBookmarks     bookmarks=[...]
 *   输入搜索关键词     →  search(query)   →  SearchURL      →  searchResults   →  SearchResultList 重渲染
 */

import { create } from "zustand"
import { URLService } from "../../bindings/collections/internal/service"
import type {
  Site,
  Bookmark,
  SiteWithBookmarks,
} from "../../bindings/collections/internal/model"

// ─── 类型定义 ────────────────────────────────────────────────

/**
 * 右侧面板当前展示的视图类型（联合类型，类比 Go 的 interface + type assert）。
 *
 * - none:     空态，未选中任何站点
 * - site:     展示站点详情 + 书签列表
 * - bookmark: 展示单个书签详情（从站点视图点进去）
 */
export type DetailView =
  | { type: "none" }
  | { type: "site"; siteId: string }
  | { type: "bookmark"; bookmarkId: string; siteId: string }

/** 从 DetailView 中提取当前 siteId（站点或书签视图都有） */
export function getActiveSiteId(view: DetailView): string | null {
  return view.type === "none" ? null : view.siteId
}

/** 书签展示模式 */
export type BookmarkViewMode = "grid" | "list"

const PAGE_SIZE = 50

let searchVersion = 0

// ─── State 接口 ──────────────────────────────────────────────

interface URLState {
  // ═══ 数据域 1：站点列表（左栏，分页加载） ═══
  sites: Site[]
  sitesTotal: number
  sitesPage: number         // 当前已加载到第几页
  sitesHasMore: boolean     // 后端是否还有下一页
  sitesLoading: boolean

  // ═══ 数据域 2：详情面板（右栏） ═══
  detailView: DetailView    // 当前展示的视图类型
  currentSite: Site | null  // 选中的站点完整数据
  bookmarks: Bookmark[]     // 当前站点下的书签列表（也是分页的）
  bookmarksTotal: number
  bookmarksPage: number
  bookmarksHasMore: boolean
  bookmarksLoading: boolean
  bookmarkViewMode: BookmarkViewMode  // 网格 or 列表
  currentBookmark: Bookmark | null    // 选中的单个书签

  // ═══ 数据域 3：搜索与筛选 ═══
  searchMode: boolean       // 是否处于搜索模式（切换左栏数据源）
  searchQuery: string
  searchResults: SiteWithBookmarks[]  // 搜索结果：站点+命中的书签
  searchTotal: number
  searchHasMore: boolean
  searchPage: number
  searchLoading: boolean
  selectedTags: string[]    // 标签筛选（AND 语义，与关键词取交集）

  // ═══ Action 方法 ═══

  // --- 站点列表 ---
  loadSites: () => Promise<void>
  loadMoreSites: () => Promise<void>

  // --- 详情面板导航 ---
  selectSite: (siteId: string) => Promise<void>
  loadMoreBookmarks: () => Promise<void>
  selectBookmark: (bookmarkId: string) => Promise<void>
  /** 从书签详情返回站点视图 */
  backToSite: () => void
  setBookmarkViewMode: (mode: BookmarkViewMode) => void
  /** 数据变更后刷新右栏 + 左栏（创建/编辑/删除后调用） */
  refreshCurrentSite: () => Promise<void>

  // --- 搜索 ---
  search: (query: string) => Promise<void>
  loadMoreSearch: () => Promise<void>
  clearSearch: () => void
  selectSearchResult: (item: SiteWithBookmarks) => void
  setSelectedTags: (tags: string[]) => void
}

// ─── Store 实现 ──────────────────────────────────────────────

export const useURLStore = create<URLState>((set, get) => ({
  // --- 初始值 ---
  sites: [],
  sitesTotal: 0,
  sitesPage: 1,
  sitesHasMore: false,
  sitesLoading: false,

  detailView: { type: "none" },
  currentSite: null,
  bookmarks: [],
  bookmarksTotal: 0,
  bookmarksPage: 1,
  bookmarksHasMore: false,
  bookmarksLoading: false,
  bookmarkViewMode: "grid",
  currentBookmark: null,

  searchMode: false,
  searchQuery: "",
  searchResults: [],
  searchTotal: 0,
  searchHasMore: false,
  searchPage: 1,
  searchLoading: false,
  selectedTags: [],

  // ═══ 站点列表操作 ═══

  /** 加载第一页站点（页面初始化 或 数据变更后刷新） */
  loadSites: async () => {
    set({ sitesLoading: true })
    try {
      const result = await URLService.ListSites({ page: 1, page_size: PAGE_SIZE })
      set({
        sites: result?.items ?? [],
        sitesTotal: result?.total ?? 0,
        sitesPage: 1,
        sitesHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ sitesLoading: false })
    }
  },

  /** 加载下一页站点（无限滚动触发，由 useInfiniteScroll hook 调用） */
  loadMoreSites: async () => {
    const { sitesHasMore, sitesLoading, sitesPage, sites } = get()
    if (!sitesHasMore || sitesLoading) return  // 防止重复请求
    set({ sitesLoading: true })
    try {
      const nextPage = sitesPage + 1
      const result = await URLService.ListSites({ page: nextPage, page_size: PAGE_SIZE })
      set({
        sites: [...sites, ...(result?.items ?? [])],  // 追加到已有列表
        sitesTotal: result?.total ?? 0,
        sitesPage: nextPage,
        sitesHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ sitesLoading: false })
    }
  },

  // ═══ 详情面板导航 ═══

  /**
   * 选中站点：并行加载站点详情和第一页书签。
   * 先 set 空态让 UI 立即切换到 loading 状态，再异步填充数据。
   */
  selectSite: async (siteId) => {
    set({
      detailView: { type: "site", siteId },
      currentSite: null,
      bookmarks: [],
      bookmarksPage: 1,
      bookmarksHasMore: false,
      currentBookmark: null,
      bookmarksLoading: true,
    })
    try {
      const [site, bmResult] = await Promise.all([
        URLService.GetSite(siteId),
        URLService.ListBookmarks({ site_id: siteId, page: 1, page_size: PAGE_SIZE }),
      ])
      if (getActiveSiteId(get().detailView) !== siteId) return
      set({
        currentSite: site ?? null,
        bookmarks: bmResult?.items ?? [],
        bookmarksTotal: bmResult?.total ?? 0,
        bookmarksHasMore: bmResult?.has_more ?? false,
      })
    } finally {
      if (getActiveSiteId(get().detailView) === siteId) {
        set({ bookmarksLoading: false })
      }
    }
  },

  loadMoreBookmarks: async () => {
    const { detailView, bookmarksHasMore, bookmarksLoading, bookmarksPage, bookmarks } = get()
    if (detailView.type !== "site" || !bookmarksHasMore || bookmarksLoading) return
    set({ bookmarksLoading: true })
    try {
      const nextPage = bookmarksPage + 1
      const result = await URLService.ListBookmarks({
        site_id: detailView.siteId,
        page: nextPage,
        page_size: PAGE_SIZE,
      })
      set({
        bookmarks: [...bookmarks, ...(result?.items ?? [])],
        bookmarksTotal: result?.total ?? 0,
        bookmarksPage: nextPage,
        bookmarksHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ bookmarksLoading: false })
    }
  },

  selectBookmark: async (bookmarkId) => {
    const { detailView } = get()
    const siteId = getActiveSiteId(detailView) ?? ""
    set({ detailView: { type: "bookmark", bookmarkId, siteId }, currentBookmark: null })
    try {
      const bm = await URLService.GetBookmark(bookmarkId)
      const current = get().detailView
      if (current.type === "bookmark" && current.bookmarkId === bookmarkId) {
        set({ currentBookmark: bm ?? null })
      }
    } catch {
      // 加载失败保持空态，UI 会展示 loading 或错误提示
    }
  },

  backToSite: () => {
    const { detailView } = get()
    if (detailView.type === "bookmark") {
      set({
        detailView: { type: "site", siteId: detailView.siteId },
        currentBookmark: null,
      })
    }
  },

  setBookmarkViewMode: (mode) => set({ bookmarkViewMode: mode }),

  /**
   * 刷新当前站点的所有数据（创建/编辑/删除书签后调用）。
   * 副作用链：刷新右栏详情 → 同时刷新左栏站点列表（因为书签数等可能变化）。
   */
  refreshCurrentSite: async () => {
    const siteId = getActiveSiteId(get().detailView)
    if (!siteId) return

    const [site, bmResult] = await Promise.all([
      URLService.GetSite(siteId),
      URLService.ListBookmarks({ site_id: siteId, page: 1, page_size: PAGE_SIZE }),
    ])
    set({
      currentSite: site ?? null,
      bookmarks: bmResult?.items ?? [],
      bookmarksTotal: bmResult?.total ?? 0,
      bookmarksPage: 1,
      bookmarksHasMore: bmResult?.has_more ?? false,
    })
    get().loadSites()
  },

  // ═══ 搜索与筛选 ═══

  /** 执行搜索：关键词 + 标签筛选取交集（AND），均为空时退出搜索模式 */
  search: async (query) => {
    const { selectedTags } = get()
    if (!query.trim() && selectedTags.length === 0) {
      get().clearSearch()
      return
    }
    const version = ++searchVersion
    set({ searchMode: true, searchQuery: query, searchLoading: true, detailView: { type: "none" } })
    try {
      const result = await URLService.SearchURL({
        search: query.trim() || undefined,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        page: 1,
        page_size: PAGE_SIZE,
      })
      if (searchVersion !== version) return
      set({
        searchResults: result?.items ?? [],
        searchTotal: result?.total ?? 0,
        searchPage: 1,
        searchHasMore: result?.has_more ?? false,
      })
    } finally {
      if (searchVersion === version) set({ searchLoading: false })
    }
  },

  loadMoreSearch: async () => {
    const { searchHasMore, searchLoading, searchPage, searchQuery, selectedTags } = get()
    if (!searchHasMore || searchLoading) return
    const version = searchVersion
    set({ searchLoading: true })
    try {
      const nextPage = searchPage + 1
      const result = await URLService.SearchURL({
        search: searchQuery.trim() || undefined,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        page: nextPage,
        page_size: PAGE_SIZE,
      })
      if (searchVersion !== version) return
      const current = get().searchResults
      set({
        searchResults: [...current, ...(result?.items ?? [])],
        searchTotal: result?.total ?? 0,
        searchPage: nextPage,
        searchHasMore: result?.has_more ?? false,
      })
    } finally {
      if (searchVersion === version) set({ searchLoading: false })
    }
  },

  /** 退出搜索模式，重置所有搜索状态 */
  clearSearch: () => {
    set({
      searchMode: false,
      searchQuery: "",
      searchResults: [],
      searchTotal: 0,
      searchHasMore: false,
      searchPage: 1,
      searchLoading: false,
      selectedTags: [],
      detailView: { type: "none" },
    })
  },

  /**
   * 选中搜索结果中的一条：直接用搜索返回的数据填充右栏，
   * 不再额外请求后端（因为搜索结果中已包含站点+命中书签）。
   */
  selectSearchResult: (item) => {
    set({
      detailView: { type: "site", siteId: item.site.id },
      currentSite: item.site,
      bookmarks: item.bookmarks,
      bookmarksTotal: item.bookmarks.length,
      bookmarksPage: 1,
      bookmarksHasMore: false,
      bookmarksLoading: false,
      currentBookmark: null,
    })
  },

  /** 标签筛选变化时自动触发搜索（或清除搜索） */
  setSelectedTags: (tags) => {
    set({ selectedTags: tags })
    const { searchQuery } = get()
    if (tags.length > 0 || searchQuery.trim()) {
      get().search(searchQuery)
    } else {
      get().clearSearch()
    }
  },

}))
