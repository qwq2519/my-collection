import { useEffect, useRef, useCallback } from "react"
import { cn } from "@/lib/utils"
import { useURLStore } from "@/stores/url"
import { EmptyState } from "@/components/EmptyState"
import { Globe, Loader2, Search } from "lucide-react"

/**
 * 左侧站点列表：默认模式分页加载，搜索模式展示搜索结果。
 * 两种模式共用同一个列表 UI，数据源不同。
 */
export function SiteList() {
  const searchMode = useURLStore((s) => s.searchMode)

  return searchMode ? <SearchResultList /> : <DefaultSiteList />
}

/** 默认模式：站点列表 + 无限滚动 */
function DefaultSiteList() {
  const sites = useURLStore((s) => s.sites)
  const loading = useURLStore((s) => s.sitesLoading)
  const hasMore = useURLStore((s) => s.sitesHasMore)
  const detailView = useURLStore((s) => s.detailView)
  const loadSites = useURLStore((s) => s.loadSites)
  const loadMore = useURLStore((s) => s.loadMoreSites)
  const selectSite = useURLStore((s) => s.selectSite)

  const selectedSiteId =
    detailView.type === "site" ? detailView.siteId
    : detailView.type === "bookmark" ? detailView.siteId
    : null

  useEffect(() => {
    loadSites()
  }, [loadSites])

  // 无限滚动：IntersectionObserver 监听哨兵元素
  const sentinelRef = useRef<HTMLDivElement>(null)
  const loadMoreRef = useRef(loadMore)
  loadMoreRef.current = loadMore

  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMoreRef.current()
        }
      },
      { threshold: 0.1 }
    )
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [])

  const handleSelect = useCallback(
    (id: string) => { selectSite(id) },
    [selectSite]
  )

  if (!loading && sites.length === 0) {
    return <EmptyState icon={Globe} message="暂无站点" className="h-full" />
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {sites.map((site) => (
        <SiteListItem
          key={site.id}
          title={site.title}
          icon={site.icon}
          count={site.bookmark_count}
          selected={selectedSiteId === site.id}
          onClick={() => handleSelect(site.id)}
        />
      ))}

      {loading && <LoadingSpinner />}
      {hasMore && <div ref={sentinelRef} className="h-4 shrink-0" />}
    </div>
  )
}

/** 搜索模式：展示搜索结果，每条显示站点名 + 命中书签数 */
function SearchResultList() {
  const results = useURLStore((s) => s.searchResults)
  const loading = useURLStore((s) => s.searchLoading)
  const hasMore = useURLStore((s) => s.searchHasMore)
  const detailView = useURLStore((s) => s.detailView)
  const loadMore = useURLStore((s) => s.loadMoreSearch)
  const selectResult = useURLStore((s) => s.selectSearchResult)

  const selectedSiteId =
    detailView.type === "site" ? detailView.siteId
    : detailView.type === "bookmark" ? detailView.siteId
    : null

  const sentinelRef = useRef<HTMLDivElement>(null)
  const loadMoreRef = useRef(loadMore)
  loadMoreRef.current = loadMore

  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMoreRef.current()
        }
      },
      { threshold: 0.1 }
    )
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [])

  if (!loading && results.length === 0) {
    return <EmptyState icon={Search} message="无匹配结果" className="h-full" />
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {results.map((item) => {
        const hitCount = item.bookmarks.length
        return (
          <SiteListItem
            key={item.site.id}
            title={item.site.title}
            icon={item.site.icon}
            count={hitCount}
            countLabel={hitCount > 0 ? `${hitCount} 命中` : undefined}
            selected={selectedSiteId === item.site.id}
            onClick={() => selectResult(item)}
          />
        )
      })}

      {loading && <LoadingSpinner />}
      {hasMore && <div ref={sentinelRef} className="h-4 shrink-0" />}
    </div>
  )
}

// ─── 共用子组件 ────────────────────────────────────────────────

function SiteListItem({
  title,
  icon,
  count,
  countLabel,
  selected,
  onClick,
}: {
  title: string
  icon?: string
  count: number
  countLabel?: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 px-3 py-2 text-left rounded-md mx-1 transition-colors duration-150",
        selected ? "bg-muted font-medium" : "hover:bg-muted/50"
      )}
    >
      {icon ? (
        <img
          src={`/persist/url-assets/icons/${icon}`}
          alt=""
          className="w-4 h-4 rounded-sm shrink-0"
          onError={(e) => { e.currentTarget.style.display = "none" }}
        />
      ) : (
        <Globe size={16} className="shrink-0 text-muted-foreground" />
      )}
      <div className="flex-1 min-w-0">
        <div className="text-sm truncate">{title}</div>
      </div>
      <span className="text-xs text-muted-foreground shrink-0">
        {countLabel ?? count}
      </span>
    </button>
  )
}

function LoadingSpinner() {
  return (
    <div className="flex justify-center py-3">
      <Loader2 size={16} className="animate-spin text-muted-foreground" />
    </div>
  )
}
