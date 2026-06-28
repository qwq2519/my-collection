import { useURLStore } from "@/stores/url"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { EmptyState } from "@/components/EmptyState"
import { Globe, LayoutGrid, List, Loader2, ExternalLink, Bookmark } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Bookmark as BookmarkType } from "../../../bindings/collections/internal/model"

/**
 * 站点详情面板：上半部分站点信息，下半部分书签网格/列表。
 */
export function SiteDetail() {
  const site = useURLStore((s) => s.currentSite)
  const bookmarks = useURLStore((s) => s.bookmarks)
  const bookmarksLoading = useURLStore((s) => s.bookmarksLoading)
  const viewMode = useURLStore((s) => s.bookmarkViewMode)
  const setViewMode = useURLStore((s) => s.setBookmarkViewMode)
  const selectBookmark = useURLStore((s) => s.selectBookmark)

  if (!site) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={20} className="animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {/* 站点信息 */}
      <div className="px-6 pt-5 pb-4">
        <div className="flex items-start gap-3">
          {site.icon ? (
            <img
              src={`/persist/url-assets/icons/${site.icon}`}
              alt=""
              className="w-8 h-8 rounded-md shrink-0 mt-0.5"
              onError={(e) => { e.currentTarget.style.display = "none" }}
            />
          ) : (
            <Globe size={32} className="shrink-0 text-muted-foreground mt-0.5" />
          )}
          <div className="flex-1 min-w-0">
            <h1 className="text-base font-semibold truncate">{site.title}</h1>
            <a
              href={site.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-muted-foreground hover:text-primary inline-flex items-center gap-1 mt-0.5"
            >
              {site.domain}
              <ExternalLink size={10} />
            </a>
          </div>
        </div>

        {site.description && (
          <p className="text-sm text-muted-foreground mt-3">{site.description}</p>
        )}

        {site.tags && site.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-3">
            {site.tags.map((tag) => (
              <Badge key={tag} variant="secondary">{tag}</Badge>
            ))}
          </div>
        )}
      </div>

      <Separator />

      {/* 书签区域标题 + 视图切换 */}
      <div className="flex items-center justify-between px-6 py-2">
        <span className="text-sm text-muted-foreground">
          书签 ({site.bookmark_count})
        </span>
        <div className="flex gap-1">
          <Button
            variant={viewMode === "grid" ? "secondary" : "ghost"}
            size="icon"
            className="h-7 w-7"
            onClick={() => setViewMode("grid")}
          >
            <LayoutGrid size={14} />
          </Button>
          <Button
            variant={viewMode === "list" ? "secondary" : "ghost"}
            size="icon"
            className="h-7 w-7"
            onClick={() => setViewMode("list")}
          >
            <List size={14} />
          </Button>
        </div>
      </div>

      {/* 书签内容 */}
      <div className="flex-1 px-6 pb-4 overflow-y-auto">
        {bookmarksLoading && bookmarks.length === 0 ? (
          <div className="flex justify-center py-8">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : bookmarks.length === 0 ? (
          <EmptyState icon={Bookmark} message="暂无书签" />
        ) : viewMode === "grid" ? (
          <BookmarkGrid bookmarks={bookmarks} onSelect={selectBookmark} />
        ) : (
          <BookmarkListView bookmarks={bookmarks} onSelect={selectBookmark} />
        )}
      </div>
    </div>
  )
}

// ─── 书签网格视图 ─────────────────────────────────────────────

function BookmarkGrid({
  bookmarks,
  onSelect,
}: {
  bookmarks: BookmarkType[]
  onSelect: (id: string) => void
}) {
  return (
    <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
      {bookmarks.map((bm) => {
        const cover = getCoverAttachment(bm)
        return (
          <button
            key={bm.id}
            onClick={() => onSelect(bm.id)}
            className="flex flex-col rounded-md border hover:bg-muted/50 overflow-hidden text-left transition-colors duration-150"
          >
            {/* 封面区域 */}
            <div className="aspect-[16/10] bg-muted flex items-center justify-center overflow-hidden">
              {cover ? (
                <img
                  src={`/persist/url-assets/attachments/${bm.id}/${cover.filename}.thumb.jpg`}
                  alt=""
                  className="w-full h-full object-cover"
                  onError={(e) => {
                    // 缩略图加载失败时尝试原图
                    const img = e.currentTarget
                    if (!img.dataset.fallback) {
                      img.dataset.fallback = "1"
                      img.src = `/persist/url-assets/attachments/${bm.id}/${cover.filename}`
                    }
                  }}
                />
              ) : (
                <span className="text-xs text-muted-foreground px-2 text-center truncate">
                  {bm.title}
                </span>
              )}
            </div>
            {/* 标题 */}
            <div className="px-2 py-1.5">
              <div className="text-sm font-medium truncate">{bm.title}</div>
            </div>
          </button>
        )
      })}
    </div>
  )
}

// ─── 书签列表视图 ─────────────────────────────────────────────

function BookmarkListView({
  bookmarks,
  onSelect,
}: {
  bookmarks: BookmarkType[]
  onSelect: (id: string) => void
}) {
  return (
    <div className="flex flex-col">
      {bookmarks.map((bm) => (
        <button
          key={bm.id}
          onClick={() => onSelect(bm.id)}
          className={cn(
            "flex items-center gap-3 px-3 py-2 rounded-md text-left",
            "hover:bg-muted/50 transition-colors duration-150"
          )}
        >
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">{bm.title}</div>
            <div className="text-xs text-muted-foreground truncate">{bm.url}</div>
          </div>
          {bm.tags && bm.tags.length > 0 && (
            <span className="text-xs text-muted-foreground shrink-0">
              {bm.tags.length} 标签
            </span>
          )}
        </button>
      ))}
    </div>
  )
}

// ─── 工具函数 ─────────────────────────────────────────────────

/** 从附件中取第一个可预览的图片/视频作为封面 */
const imageExts = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "svg"])
const videoExts = new Set(["mp4", "mkv", "avi", "mov", "webm", "wmv", "flv"])

function getCoverAttachment(bm: BookmarkType) {
  if (!bm.attachments || bm.attachments.length === 0) return null
  return bm.attachments.find((a) => {
    const ext = a.filename.split(".").pop()?.toLowerCase() ?? ""
    return imageExts.has(ext) || videoExts.has(ext)
  }) ?? null
}
