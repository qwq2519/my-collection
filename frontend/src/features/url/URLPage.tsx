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
import { Globe, Plus, Bookmark } from "lucide-react"

type CreateMode = null | "site" | "bookmark"

/**
 * URL 收藏模块入口：搜索栏 + 左列表右详情双栏布局。
 */
export function URLPage() {
  const detailView = useURLStore((s) => s.detailView)
  const searchMode = useURLStore((s) => s.searchMode)
  const loadSites = useURLStore((s) => s.loadSites)
  const [createMode, setCreateMode] = useState<CreateMode>(null)

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

        {/* 右栏：详情面板 */}
        <div className="flex-1 overflow-hidden">
          <RightPanel createMode={createMode} setCreateMode={setCreateMode} />
        </div>
      </div>
    </div>
  )
}

// ─── 右栏面板：创建态 / 详情态 / 空态 ────────────────────────

function RightPanel({
  createMode,
  setCreateMode,
}: {
  createMode: CreateMode
  setCreateMode: (mode: CreateMode) => void
}) {
  const detailView = useURLStore((s) => s.detailView)
  const loadSites = useURLStore((s) => s.loadSites)

  const handleSaved = () => {
    setCreateMode(null)
    loadSites()
  }
  const handleCancel = () => setCreateMode(null)

  if (createMode === "site") {
    return <SiteForm onSave={handleSaved} onCancel={handleCancel} />
  }

  if (createMode === "bookmark") {
    return (
      <BookmarkForm
        onSave={handleSaved}
        onCancel={handleCancel}
        onCreateSite={() => setCreateMode("site")}
      />
    )
  }

  if (detailView.type === "none") {
    return <EmptyState icon={Globe} message="选择一个站点查看详情" className="h-full" />
  }

  return detailView.type === "site" ? <SiteDetail /> : <BookmarkDetail />
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
