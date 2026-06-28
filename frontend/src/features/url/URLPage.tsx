import { useMemo, useState } from "react"
import { SiteList } from "./SiteList"
import { SiteDetail } from "./SiteDetail"
import { BookmarkDetail } from "./BookmarkDetail"
import { SiteForm } from "./SiteForm"
import { BookmarkForm } from "./BookmarkForm"
import { useURLStore } from "@/stores/url"
import { EmptyState } from "@/components/EmptyState"
import { SearchBar } from "@/components/SearchBar"
import { TagTreeFilter } from "@/components/TagTreeFilter"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog"
import { Globe, Plus, Bookmark } from "lucide-react"

type CreateMode = null | "site" | "bookmark"

/**
 * URL 收藏模块入口：搜索栏 + 左列表右详情双栏布局。
 * 新建站点/书签用弹窗，不替换右栏内容。
 */
export function URLPage() {
  const detailView = useURLStore((s) => s.detailView)
  const searchMode = useURLStore((s) => s.searchMode)
  const loadSites = useURLStore((s) => s.loadSites)
  const [createMode, setCreateMode] = useState<CreateMode>(null)

  const handleCreated = () => {
    setCreateMode(null)
    loadSites()
  }

  return (
    <div className="flex flex-col h-full">
      <SearchToolbar
        onCreateSite={() => setCreateMode("site")}
        onCreateBookmark={() => setCreateMode("bookmark")}
      />

      <div className="flex flex-1 overflow-hidden">
        {/* 左栏：站点列表 */}
        <div className="w-[260px] shrink-0 border-r flex flex-col">
          <div className="px-3 py-2 border-b">
            <h2 className="text-sm font-semibold text-foreground">
              {searchMode ? "搜索结果" : "站点"}
            </h2>
          </div>
          <div className="flex-1 overflow-hidden py-1">
            <SiteList />
          </div>
        </div>

        {/* 右栏：始终展示详情，不被创建表单替换 */}
        <div className="flex-1 overflow-hidden">
          {detailView.type === "none" ? (
            <EmptyState icon={Globe} message="选择一个站点查看详情" className="h-full" />
          ) : detailView.type === "site" ? (
            <SiteDetail />
          ) : (
            <BookmarkDetail />
          )}
        </div>
      </div>

      {/* 创建弹窗：覆盖在当前内容之上，关闭后右栏不变 */}
      <CreateFormDialog
        mode={createMode}
        onClose={() => setCreateMode(null)}
        onCreated={handleCreated}
        onSwitchToSite={() => setCreateMode("site")}
      />
    </div>
  )
}

// ─── 创建表单弹窗 ─────────────────────────────────────────────

function CreateFormDialog({
  mode,
  onClose,
  onCreated,
  onSwitchToSite,
}: {
  mode: CreateMode
  onClose: () => void
  onCreated: () => void
  onSwitchToSite: () => void
}) {
  return (
    <Dialog open={mode !== null} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="sm:max-w-[500px] p-0 max-h-[85vh] overflow-y-auto">
        <DialogTitle className="sr-only">
          {mode === "site" ? "新建站点" : "新建书签"}
        </DialogTitle>
        {mode === "site" && (
          <SiteForm onSave={onCreated} onCancel={onClose} />
        )}
        {mode === "bookmark" && (
          <BookmarkForm
            onSave={onCreated}
            onCancel={onClose}
            onCreateSite={onSwitchToSite}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

// ─── 搜索工具栏：搜索 + 标签筛选 + 新建按钮 ─────────────────

function SearchToolbar({
  onCreateSite,
  onCreateBookmark,
}: {
  onCreateSite: () => void
  onCreateBookmark: () => void
}) {
  const searchQuery = useURLStore((s) => s.searchQuery)
  const selectedTags = useURLStore((s) => s.selectedTags)
  const search = useURLStore((s) => s.search)
  const setSelectedTags = useURLStore((s) => s.setSelectedTags)
  const allTags = useCollectedTags()

  return (
    <div className="flex items-center gap-2 px-3 py-2 border-b shrink-0">
      <SearchBar
        value={searchQuery}
        onChange={search}
        placeholder="搜索站点和书签..."
        className="flex-1"
      />
      <TagTreeFilter
        allTags={allTags}
        selectedTags={selectedTags}
        onChange={setSelectedTags}
      />
      <Button
        variant="outline" size="sm"
        className="h-8 gap-1 shrink-0 text-xs"
        onClick={onCreateSite}
      >
        <Plus size={14} />
        站点
      </Button>
      <Button
        variant="outline" size="sm"
        className="h-8 gap-1 shrink-0 text-xs"
        onClick={onCreateBookmark}
      >
        <Bookmark size={14} />
        书签
      </Button>
    </div>
  )
}

// ─── 从当前数据中收集标签（后续替换为 TagService） ───────────

function useCollectedTags(): string[] {
  const sites = useURLStore((s) => s.sites)
  const searchResults = useURLStore((s) => s.searchResults)
  const searchMode = useURLStore((s) => s.searchMode)

  return useMemo(() => {
    const source = searchMode ? searchResults.map((r) => r.site) : sites
    const tagSet = new Set<string>()
    for (const site of source) {
      if (site.tags) site.tags.forEach((t) => tagSet.add(t))
    }
    return Array.from(tagSet).sort()
  }, [sites, searchResults, searchMode])
}
