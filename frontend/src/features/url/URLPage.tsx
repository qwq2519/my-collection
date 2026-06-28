import { SiteList } from "./SiteList"
import { SiteDetail } from "./SiteDetail"
import { BookmarkDetail } from "./BookmarkDetail"
import { useURLStore } from "@/stores/url"
import { EmptyState } from "@/components/EmptyState"
import { Globe } from "lucide-react"

/**
 * URL 收藏模块入口：双栏布局。
 * 左侧站点列表 | 右侧详情面板（站点详情 / 书签详情 / 空态）。
 */
export function URLPage() {
  const detailView = useURLStore((s) => s.detailView)

  return (
    <div className="flex h-full">
      {/* 左栏：站点列表 */}
      <div className="w-[260px] shrink-0 border-r flex flex-col">
        <div className="px-3 py-2 border-b">
          <h2 className="text-sm font-semibold text-foreground">站点</h2>
        </div>
        <div className="flex-1 overflow-hidden py-1">
          <SiteList />
        </div>
      </div>

      {/* 右栏：详情面板 */}
      <div className="flex-1 overflow-hidden">
        {detailView.type === "none" && (
          <EmptyState icon={Globe} message="选择一个站点查看详情" className="h-full" />
        )}
        {detailView.type === "site" && <SiteDetail />}
        {detailView.type === "bookmark" && <BookmarkDetail />}
      </div>
    </div>
  )
}
