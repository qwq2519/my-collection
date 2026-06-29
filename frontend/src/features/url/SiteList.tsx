import { useEffect, useState } from "react"
import { cn, formatRelativeTime } from "@/lib/utils"
import { useURLStore, getActiveSiteId, useSiteListState, useSearchResultState } from "@/stores/url"
import { useInfiniteScroll } from "@/hooks/useInfiniteScroll"
import { EmptyState } from "@/components/EmptyState"
import { Badge } from "@/components/ui/badge"
import { Globe, Loader2, Search } from "lucide-react"
import { arr } from "@/lib/safe"
import { pick } from "@/lib/safe"

/**
 * 左侧站点列表：根据 searchMode 切换数据源。
 *
 * 组件结构：
 *   SiteList（路由器，根据 searchMode 选择子组件）
 *     → DefaultSiteList（默认模式，分页加载全部站点）
 *     → SearchResultList（搜索模式，展示搜索命中的站点）
 *     → SiteListItem（共用列表项 UI）
 *
 * 关于 useURLStore((s) => s.xxx) 的多行写法：
 *   React hooks 规则要求所有 hook 必须在函数顶部调用，不能放在 if/for 里。
 *   Zustand 的 selector 写法 (s) => s.xxx 是为了精确订阅单个字段，
 *   避免整个 store 任意字段变化时都触发重渲染。所以每个字段要写一行。
 */
export function SiteList() {
  const searchMode = useURLStore((s) => s.searchMode)

  if (searchMode) return <SearchResultList />
  return <DefaultSiteList />
}

/** 默认模式：站点列表 + 无限滚动 */
function DefaultSiteList() {
  const { sites, loading, hasMore, detailView, loadSites, loadMore, selectSite } =
    useSiteListState()

  const selectedSiteId = getActiveSiteId(detailView)
  const sentinelRef = useInfiniteScroll(loadMore, hasMore)

  useEffect(() => {
    loadSites()
  }, [loadSites])

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
  const { results, loading, hasMore, detailView, loadMore, selectResult } = useSearchResultState()

  const selectedSiteId = getActiveSiteId(detailView)
  const sentinelRef = useInfiniteScroll(loadMore, hasMore)

  if (!loading && results.length === 0) {
    return <EmptyState icon={Search} message="无匹配结果" className="h-full" />
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {results.map((item, index) => {
        const hitCount = item.bookmarks.length
        return (
          <SiteListItem
            key={item.site.id}
            index={index}
            title={item.site.title}
            icon={item.site.icon}
            domain={item.site.domain}
            count={hitCount}
            countLabel={pick(hitCount > 0, `${hitCount} 命中`, undefined)}
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

/**
 * 站点列表项：展示站点图标、标题、域名、书签数、时间、标签。
 * data-site-index 属性供键盘导航（useKeyboardNav）定位 DOM 元素。
 */
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
  index,
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
  index?: number
}) {
  const timeStr = formatRelativeTime(updatedAt)
  const visibleTags = arr(tags).slice(0, 3)

  const secondaryParts: string[] = []
  if (domain) secondaryParts.push(domain)
  secondaryParts.push(countLabel || `${count} 书签`)
  if (timeStr) secondaryParts.push(timeStr)

  return (
    <button
      onClick={onClick}
      data-site-index={index}
      className={cn(
        "flex gap-2 px-3 py-2 text-left rounded-md mx-1 transition-colors duration-150",
        pick(selected, "bg-muted font-medium", "hover:bg-muted/50"),
      )}
    >
      <div className="pt-0.5 shrink-0">
        <SiteListIcon icon={icon} />
      </div>
      <div className="flex-1 min-w-0 space-y-0.5">
        <div className="text-sm font-medium truncate">{title}</div>
        <div className="text-xs text-muted-foreground truncate">{secondaryParts.join(" · ")}</div>
        {visibleTags.length > 0 && (
          <div className="flex gap-1 flex-wrap">
            {visibleTags.map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs px-1.5 py-0 h-4 font-normal">
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

function SiteListIcon({ icon }: { icon?: string }) {
  const [failed, setFailed] = useState(false)
  if (!icon || failed) return <Globe size={16} className="text-muted-foreground" />
  return (
    <img
      src={`/persist/url-assets/icons/${icon}`}
      alt=""
      className="w-4 h-4 rounded-sm"
      onError={() => setFailed(true)}
    />
  )
}

function LoadingSpinner() {
  return (
    <div className="flex justify-center py-3">
      <Loader2 size={16} className="animate-spin text-muted-foreground" />
    </div>
  )
}
