import { create } from "zustand"
import { URLService } from "../../bindings/collections/internal/service"
import type {
  Site,
  Bookmark,
  SiteWithBookmarks,
} from "../../bindings/collections/internal/model"

/** 右侧面板当前展示的视图类型 */
export type DetailView =
  | { type: "none" }
  | { type: "site"; siteId: string }
  | { type: "bookmark"; bookmarkId: string; siteId: string }

/** 书签展示模式 */
export type BookmarkViewMode = "grid" | "list"

const PAGE_SIZE = 50

interface URLState {
  // ─── 站点列表 ───────────────────────────────────
  sites: Site[]
  sitesTotal: number
  sitesPage: number
  sitesHasMore: boolean
  sitesLoading: boolean

  // ─── 右侧面板 ───────────────────────────────────
  detailView: DetailView
  currentSite: Site | null
  bookmarks: Bookmark[]
  bookmarksTotal: number
  bookmarksPage: number
  bookmarksHasMore: boolean
  bookmarksLoading: boolean
  bookmarkViewMode: BookmarkViewMode
  currentBookmark: Bookmark | null

  // ─── 搜索与筛选 ──────────────────────────────────
  searchMode: boolean
  searchQuery: string
  searchResults: SiteWithBookmarks[]
  searchTotal: number
  searchHasMore: boolean
  searchPage: number
  searchLoading: boolean
  selectedTags: string[]

  // ─── 操作方法 ───────────────────────────────────
  /** 加载第一页站点 */
  loadSites: () => Promise<void>
  /** 加载下一页站点（无限滚动） */
  loadMoreSites: () => Promise<void>

  /** 选中站点，加载站点详情和第一页书签 */
  selectSite: (siteId: string) => Promise<void>
  /** 加载下一页书签 */
  loadMoreBookmarks: () => Promise<void>

  /** 选中书签，加载书签详情 */
  selectBookmark: (bookmarkId: string) => Promise<void>
  /** 返回到站点视图 */
  backToSite: () => void

  /** 切换书签网格/列表视图 */
  setBookmarkViewMode: (mode: BookmarkViewMode) => void

  /** 刷新当前站点数据（创建/编辑/删除后调用） */
  refreshCurrentSite: () => Promise<void>

  /** 执行搜索（关键词 + 标签筛选取交集） */
  search: (query: string) => Promise<void>
  /** 加载更多搜索结果 */
  loadMoreSearch: () => Promise<void>
  /** 清除搜索，恢复默认站点列表 */
  clearSearch: () => void
  /** 选中搜索结果中的站点（展示命中的书签） */
  selectSearchResult: (item: SiteWithBookmarks) => void
  /** 设置标签筛选 */
  setSelectedTags: (tags: string[]) => void
}

export const useURLStore = create<URLState>((set, get) => ({
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

  loadMoreSites: async () => {
    const { sitesHasMore, sitesLoading, sitesPage, sites } = get()
    if (!sitesHasMore || sitesLoading) return
    set({ sitesLoading: true })
    try {
      const nextPage = sitesPage + 1
      const result = await URLService.ListSites({ page: nextPage, page_size: PAGE_SIZE })
      set({
        sites: [...sites, ...(result?.items ?? [])],
        sitesTotal: result?.total ?? 0,
        sitesPage: nextPage,
        sitesHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ sitesLoading: false })
    }
  },

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
      set({
        currentSite: site ?? null,
        bookmarks: bmResult?.items ?? [],
        bookmarksTotal: bmResult?.total ?? 0,
        bookmarksHasMore: bmResult?.has_more ?? false,
      })
    } finally {
      set({ bookmarksLoading: false })
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
    const siteId = detailView.type === "site" ? detailView.siteId
      : detailView.type === "bookmark" ? detailView.siteId
      : ""
    set({ detailView: { type: "bookmark", bookmarkId, siteId }, currentBookmark: null })
    try {
      const bm = await URLService.GetBookmark(bookmarkId)
      set({ currentBookmark: bm ?? null })
    } catch {
      // 加载失败保持空态，UI 会展示错误提示
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

  refreshCurrentSite: async () => {
    const { detailView } = get()
    const siteId = detailView.type === "site" ? detailView.siteId
      : detailView.type === "bookmark" ? detailView.siteId
      : null
    if (!siteId) return

    // 同时刷新站点列表和当前站点详情
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
    // 刷新左侧站点列表
    get().loadSites()
  },

  search: async (query) => {
    const { selectedTags } = get()
    // 无关键词且无标签筛选时退出搜索模式
    if (!query.trim() && selectedTags.length === 0) {
      get().clearSearch()
      return
    }
    set({ searchMode: true, searchQuery: query, searchLoading: true, detailView: { type: "none" } })
    try {
      const result = await URLService.SearchURL({
        search: query.trim() || undefined,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        page: 1,
        page_size: PAGE_SIZE,
      })
      set({
        searchResults: result?.items ?? [],
        searchTotal: result?.total ?? 0,
        searchPage: 1,
        searchHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ searchLoading: false })
    }
  },

  loadMoreSearch: async () => {
    const { searchHasMore, searchLoading, searchPage, searchResults, searchQuery, selectedTags } = get()
    if (!searchHasMore || searchLoading) return
    set({ searchLoading: true })
    try {
      const nextPage = searchPage + 1
      const result = await URLService.SearchURL({
        search: searchQuery.trim() || undefined,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        page: nextPage,
        page_size: PAGE_SIZE,
      })
      set({
        searchResults: [...searchResults, ...(result?.items ?? [])],
        searchTotal: result?.total ?? 0,
        searchPage: nextPage,
        searchHasMore: result?.has_more ?? false,
      })
    } finally {
      set({ searchLoading: false })
    }
  },

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

  setSelectedTags: (tags) => {
    set({ selectedTags: tags })
    const { searchQuery } = get()
    // 标签变更时重新搜索
    if (tags.length > 0 || searchQuery.trim()) {
      get().search(searchQuery)
    } else {
      get().clearSearch()
    }
  },
}))
