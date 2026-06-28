import { useState } from "react"
import { useURLStore } from "@/stores/url"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { EmptyState } from "@/components/EmptyState"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { Globe, LayoutGrid, List, Loader2, ExternalLink, Bookmark, Pencil, Plus, Trash2 } from "lucide-react"
import { cn, extractError } from "@/lib/utils"
import { SiteForm } from "./SiteForm"
import { BookmarkForm } from "./BookmarkForm"
import { BatchToolbar } from "./BatchToolbar"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Bookmark as BookmarkType } from "../../../bindings/collections/internal/model"

/**
 * 站点详情面板：展示态 / 编辑态 / 新建书签态。
 */
export function SiteDetail() {
  const site = useURLStore((s) => s.currentSite)
  const bookmarks = useURLStore((s) => s.bookmarks)
  const bookmarksLoading = useURLStore((s) => s.bookmarksLoading)
  const viewMode = useURLStore((s) => s.bookmarkViewMode)
  const setViewMode = useURLStore((s) => s.setBookmarkViewMode)
  const selectBookmark = useURLStore((s) => s.selectBookmark)
  const refreshCurrentSite = useURLStore((s) => s.refreshCurrentSite)
  const loadSites = useURLStore((s) => s.loadSites)

  const [mode, setMode] = useState<"view" | "edit-site" | "add-bookmark">("view")
  const [showDeleteSite, setShowDeleteSite] = useState(false)
  const [deleteError, setDeleteError] = useState("")

  // 批量选择状态
  const [batchMode, setBatchMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const toggleSelect = (id: string) => {
    const next = new Set(selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelectedIds(next)
  }

  const exitBatchMode = () => {
    setBatchMode(false)
    setSelectedIds(new Set())
  }

  const handleDeleteSite = async () => {
    if (!site) return
    setDeleteError("")
    try {
      await URLService.DeleteSite(site.id)
      setShowDeleteSite(false)
      loadSites()
      useURLStore.setState({ detailView: { type: "none" }, currentSite: null })
    } catch (e: any) {
      setDeleteError(extractError(e))
    }
  }

  if (!site) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={20} className="animate-spin text-muted-foreground" />
      </div>
    )
  }

  // 编辑站点
  if (mode === "edit-site") {
    return (
      <SiteForm
        site={site}
        onSave={() => { setMode("view"); refreshCurrentSite() }}
        onCancel={() => setMode("view")}
      />
    )
  }

  // 新建书签
  if (mode === "add-bookmark") {
    return (
      <BookmarkForm
        onSave={() => { setMode("view"); refreshCurrentSite() }}
        onCancel={() => setMode("view")}
      />
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
          <div className="flex gap-1 shrink-0">
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setMode("edit-site")}>
              <Pencil size={14} />
            </Button>
            <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => setShowDeleteSite(true)}>
              <Trash2 size={14} />
            </Button>
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

      {/* 删除站点确认弹窗 */}
      <ConfirmDialog
        open={showDeleteSite}
        onOpenChange={setShowDeleteSite}
        title="删除站点"
        description={deleteError || `确定删除「${site.title}」？站点下仍有书签时无法删除。`}
        confirmLabel="删除"
        onConfirm={handleDeleteSite}
      />

      {/* 书签区域标题 + 添加/批量按钮 + 视图切换 */}
      <div className="flex items-center justify-between px-6 py-2">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">
            书签 ({site.bookmark_count})
          </span>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setMode("add-bookmark")}>
            <Plus size={14} />
          </Button>
          {bookmarks.length > 0 && !batchMode && (
            <Button variant="ghost" size="sm" className="h-6 text-xs px-2" onClick={() => setBatchMode(true)}>
              批量
            </Button>
          )}
        </div>
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

      {/* 批量操作栏 */}
      {batchMode && site && (
        <BatchToolbar
          siteId={site.id}
          selectedIds={selectedIds}
          totalCount={bookmarks.length}
          onSelectAll={() => setSelectedIds(new Set(bookmarks.map((b) => b.id)))}
          onDeselectAll={() => setSelectedIds(new Set())}
          onDone={() => { exitBatchMode(); refreshCurrentSite() }}
          onCancel={exitBatchMode}
        />
      )}

      {/* 书签内容 */}
      <div className="flex-1 px-6 pb-4 overflow-y-auto">
        {bookmarksLoading && bookmarks.length === 0 ? (
          <div className="flex justify-center py-8">
            <Loader2 size={20} className="animate-spin text-muted-foreground" />
          </div>
        ) : bookmarks.length === 0 ? (
          <EmptyState icon={Bookmark} message="暂无书签" />
        ) : viewMode === "grid" ? (
          <BookmarkGrid
            bookmarks={bookmarks}
            onSelect={batchMode ? toggleSelect : selectBookmark}
            batchMode={batchMode}
            selectedIds={selectedIds}
          />
        ) : (
          <BookmarkListView
            bookmarks={bookmarks}
            onSelect={batchMode ? toggleSelect : selectBookmark}
            batchMode={batchMode}
            selectedIds={selectedIds}
          />
        )}
      </div>
    </div>
  )
}

// ─── 书签网格视图 ─────────────────────────────────────────────

function BookmarkGrid({
  bookmarks,
  onSelect,
  batchMode = false,
  selectedIds = new Set(),
}: {
  bookmarks: BookmarkType[]
  onSelect: (id: string) => void
  batchMode?: boolean
  selectedIds?: Set<string>
}) {
  return (
    <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
      {bookmarks.map((bm) => {
        const cover = getCoverAttachment(bm)
        const isSelected = selectedIds.has(bm.id)
        return (
          <button
            key={bm.id}
            onClick={() => onSelect(bm.id)}
            className={cn(
              "flex flex-col rounded-md border overflow-hidden text-left transition-colors duration-150",
              isSelected ? "ring-2 ring-primary" : "hover:bg-muted/50"
            )}
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
            {/* 标题 + 批量选中指示 */}
            <div className="px-2 py-1.5 flex items-center gap-1.5">
              {batchMode && (
                <input type="checkbox" checked={isSelected} readOnly className="rounded shrink-0" />
              )}
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
  batchMode = false,
  selectedIds = new Set(),
}: {
  bookmarks: BookmarkType[]
  onSelect: (id: string) => void
  batchMode?: boolean
  selectedIds?: Set<string>
}) {
  return (
    <div className="flex flex-col">
      {bookmarks.map((bm) => {
        const isSelected = selectedIds.has(bm.id)
        return (
        <button
          key={bm.id}
          onClick={() => onSelect(bm.id)}
          className={cn(
            "flex items-center gap-3 px-3 py-2 rounded-md text-left",
            isSelected ? "bg-muted" : "hover:bg-muted/50",
            "transition-colors duration-150"
          )}
        >
          {batchMode && (
            <input type="checkbox" checked={isSelected} readOnly className="rounded shrink-0" />
          )}
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
        )
      })}
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
