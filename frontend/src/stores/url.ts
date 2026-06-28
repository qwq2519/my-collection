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

  // ─── 搜索态（阶段三实现，先预留字段） ──────────────
  searchMode: boolean
  searchQuery: string
  searchResults: SiteWithBookmarks[]
  searchTotal: number
  searchHasMore: boolean

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
}))
