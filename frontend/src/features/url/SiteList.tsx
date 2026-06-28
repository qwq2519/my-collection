import { useEffect, useRef } from "react"
import { cn, formatRelativeTime } from "@/lib/utils"
import { useURLStore, getActiveSiteId } from "@/stores/url"
import { EmptyState } from "@/components/EmptyState"
import { Badge } from "@/components/ui/badge"
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

  const selectedSiteId = getActiveSiteId(detailView)

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
  }, [hasMore])

  if (!loading && sites.length === 0) {
    return <EmptyState icon={Globe} message="暂无站点" className="h-full" />
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {sites.map((site, index) => (
        <SiteListItem
          key={site.id}
          index={index}
          title={site.title}
          icon={site.icon}
          domain={site.domain}
          count={site.bookmark_count}
          updatedAt={site.updated_at}
          tags={site.tags}
          selected={selectedSiteId === site.id}
          onClick={() => selectSite(site.id)}
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

  const selectedSiteId = getActiveSiteId(detailView)

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
  }, [hasMore])

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
            domain={item.site.domain}
            count={hitCount}
            countLabel={hitCount > 0 ? `${hitCount} 命中` : undefined}
            updatedAt={item.site.updated_at}
            tags={item.site.tags}
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
  domain,
  count,
  countLabel,
  updatedAt,
  tags,
  selected,
  onClick,
}: {
  title: string
  icon?: string
  domain?: string
  count: number
  countLabel?: string
  updatedAt?: string | Date | null
  tags?: string[]
  selected: boolean
  onClick: () => void
}) {
  const timeStr = formatRelativeTime(updatedAt)
  const visibleTags = tags?.slice(0, 3) ?? []

  const secondaryParts: string[] = []
  if (domain) secondaryParts.push(domain)
  secondaryParts.push(countLabel ?? `${count} 书签`)
  if (timeStr) secondaryParts.push(timeStr)

  return (
    <button
      onClick={onClick}
      className={cn(
        "flex gap-2 px-3 py-2 text-left rounded-md mx-1 transition-colors duration-150",
        selected ? "bg-muted font-medium" : "hover:bg-muted/50"
      )}
    >
      <div className="pt-0.5 shrink-0">
        {icon ? (
          <img
            src={`/persist/url-assets/icons/${icon}`}
            alt=""
            className="w-4 h-4 rounded-sm"
            onError={(e) => { e.currentTarget.style.display = "none" }}
          />
        ) : (
          <Globe size={16} className="text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0 space-y-0.5">
        <div className="text-sm font-medium truncate">{title}</div>
        <div className="text-xs text-muted-foreground truncate">
          {secondaryParts.join(" · ")}
        </div>
        {visibleTags.length > 0 && (
          <div className="flex gap-1 flex-wrap">
            {visibleTags.map((tag) => (
              <Badge
                key={tag}
                variant="secondary"
                className="text-xs px-1.5 py-0 h-4 font-normal"
              >
                {tag}
              </Badge>
            ))}
            {tags && tags.length > 3 && (
              <span className="text-xs text-muted-foreground">+{tags.length - 3}</span>
            )}
          </div>
        )}
      </div>
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
