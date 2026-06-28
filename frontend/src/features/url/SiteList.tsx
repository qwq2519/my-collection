import { useEffect, useRef, useCallback } from "react"
import { cn } from "@/lib/utils"
import { useURLStore } from "@/stores/url"
import { EmptyState } from "@/components/EmptyState"
import { Globe, Loader2 } from "lucide-react"

/**
 * 左侧站点列表：分页加载 + 无限滚动。
 * 每条显示站点名称 + 书签数量，选中态高亮。
 */
export function SiteList() {
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

  // 首次挂载加载站点
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
        <button
          key={site.id}
          onClick={() => handleSelect(site.id)}
          className={cn(
            "flex items-center gap-2 px-3 py-2 text-left rounded-md mx-1 transition-colors duration-150",
            selectedSiteId === site.id
              ? "bg-muted font-medium"
              : "hover:bg-muted/50"
          )}
        >
          {/* 站点图标：有 icon 时显示，否则用默认 Globe */}
          {site.icon ? (
            <img
              src={`/persist/url-assets/icons/${site.icon}`}
              alt=""
              className="w-4 h-4 rounded-sm shrink-0"
              onError={(e) => { e.currentTarget.style.display = "none" }}
            />
          ) : (
            <Globe size={16} className="shrink-0 text-muted-foreground" />
          )}
          <div className="flex-1 min-w-0">
            <div className="text-sm truncate">{site.title}</div>
          </div>
          <span className="text-xs text-muted-foreground shrink-0">
            {site.bookmark_count}
          </span>
        </button>
      ))}

      {/* 加载状态 */}
      {loading && (
        <div className="flex justify-center py-3">
          <Loader2 size={16} className="animate-spin text-muted-foreground" />
        </div>
      )}

      {/* 无限滚动哨兵 */}
      {hasMore && <div ref={sentinelRef} className="h-4 shrink-0" />}
    </div>
  )
}
