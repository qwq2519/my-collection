import { useState } from "react"
import { useURLStore } from "@/stores/url"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { EmptyState } from "@/components/EmptyState"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { Globe, LayoutGrid, List, Loader2, ExternalLink, Bookmark, Pencil, Plus, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { SiteForm } from "./SiteForm"
import { BookmarkForm } from "./BookmarkForm"
import { BatchToolbar } from "./BatchToolbar"
import { BookmarkGrid, BookmarkListView } from "./BookmarkViews"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Site } from "../../../bindings/collections/internal/model"

/**
 * 站点详情面板：协调器，根据当前模式渲染对应子视图。
 */
export function SiteDetail() {
  const site = useURLStore((s) => s.currentSite)
  const refreshCurrentSite = useURLStore((s) => s.refreshCurrentSite)
  const [mode, setMode] = useState<"view" | "edit-site" | "add-bookmark">("view")

  if (!site) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={20} className="animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (mode === "edit-site") {
    return (
      <SiteForm
        site={site}
        onSave={() => { setMode("view"); refreshCurrentSite() }}
        onCancel={() => setMode("view")}
      />
    )
  }

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
      <SiteHeader site={site} onEdit={() => setMode("edit-site")} />
      <Separator />
      <BookmarkSection site={site} onAddBookmark={() => setMode("add-bookmark")} />
    </div>
  )
}

// ─── 站点头部：信息展示 + 编辑/删除 ──────────────────────────

function SiteHeader({ site, onEdit }: { site: Site; onEdit: () => void }) {
  const loadSites = useURLStore((s) => s.loadSites)
  const [showDelete, setShowDelete] = useState(false)

  const handleDelete = async () => {
    await URLService.DeleteSite(site.id)
    toast.success("站点已删除")
    loadSites()
    useURLStore.setState({ detailView: { type: "none" }, currentSite: null })
  }

  return (
    <div className="px-6 pt-5 pb-4">
      <div className="flex items-start gap-3">
        <SiteIcon icon={site.icon} size={32} />
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
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onEdit}>
            <Pencil size={14} />
          </Button>
          <Button
            variant="ghost" size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-destructive"
            onClick={() => setShowDelete(true)}
          >
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

      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="删除站点"
        description={`确定删除「${site.title}」？站点下仍有书签时无法删除。`}
        confirmLabel="删除"
        onConfirm={handleDelete}
      />
    </div>
  )
}

// ─── 书签区域：工具栏 + 批量操作 + 内容列表 ──────────────────

function BookmarkSection({ site, onAddBookmark }: { site: Site; onAddBookmark: () => void }) {
  const bookmarks = useURLStore((s) => s.bookmarks)
  const bookmarksLoading = useURLStore((s) => s.bookmarksLoading)
  const viewMode = useURLStore((s) => s.bookmarkViewMode)
  const setViewMode = useURLStore((s) => s.setBookmarkViewMode)
  const selectBookmark = useURLStore((s) => s.selectBookmark)
  const refreshCurrentSite = useURLStore((s) => s.refreshCurrentSite)

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

  const onSelect = batchMode ? toggleSelect : selectBookmark

  return (
    <>
      {/* 工具栏：书签数 + 添加/批量按钮 + 视图切换 */}
      <div className="flex items-center justify-between px-6 py-2">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">书签 ({site.bookmark_count})</span>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onAddBookmark}>
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
            size="icon" className="h-7 w-7"
            onClick={() => setViewMode("grid")}
          >
            <LayoutGrid size={14} />
          </Button>
          <Button
            variant={viewMode === "list" ? "secondary" : "ghost"}
            size="icon" className="h-7 w-7"
            onClick={() => setViewMode("list")}
          >
            <List size={14} />
          </Button>
        </div>
      </div>

      {batchMode && (
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
          <BookmarkGrid bookmarks={bookmarks} onSelect={onSelect} batchMode={batchMode} selectedIds={selectedIds} />
        ) : (
          <BookmarkListView bookmarks={bookmarks} onSelect={onSelect} batchMode={batchMode} selectedIds={selectedIds} />
        )}
      </div>
    </>
  )
}

// ─── 站点图标（带 fallback） ──────────────────────────────────

function SiteIcon({ icon, size }: { icon?: string; size: number }) {
  if (!icon) return <Globe size={size} className="shrink-0 text-muted-foreground mt-0.5" />
  return (
    <img
      src={`/persist/url-assets/icons/${icon}`}
      alt=""
      className="rounded-md shrink-0 mt-0.5"
      style={{ width: size, height: size }}
      onError={(e) => { e.currentTarget.style.display = "none" }}
    />
  )
}

