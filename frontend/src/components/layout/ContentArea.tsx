import { useRef } from "react"
import { useAppStore, type Page } from "@/stores/app"
import { URLPage } from "@/features/url/URLPage"
import { SettingsPage } from "@/features/settings/SettingsPage"
import { cn } from "@/lib/utils"

/**
 * 内容区容器：根据 currentPage 显示/隐藏对应功能模块。
 *
 * 使用 CSS display:none 隐藏非活跃页面而非卸载组件，
 * 保留页面内部状态（滚动位置、选中项、表单输入等）。
 * 页面首次访问时才渲染（懒加载），避免启动时加载全部模块。
 */
export function ContentArea() {
  const currentPage = useAppStore((s) => s.currentPage)
  const mountedRef = useRef<Set<Page>>(new Set())

  // 记录已访问过的页面，只有访问过才渲染（懒加载）
  mountedRef.current.add(currentPage)

  return (
    <main className="flex-1 h-full overflow-hidden relative">
      <PageSlot page="url" currentPage={currentPage} mounted={mountedRef.current}>
        <URLPage />
      </PageSlot>
      <PageSlot page="notes" currentPage={currentPage} mounted={mountedRef.current}>
        <Placeholder label="笔记" />
      </PageSlot>
      <PageSlot page="media" currentPage={currentPage} mounted={mountedRef.current}>
        <Placeholder label="媒体" />
      </PageSlot>
      <PageSlot page="tags" currentPage={currentPage} mounted={mountedRef.current}>
        <Placeholder label="标签管理" />
      </PageSlot>
      <PageSlot page="settings" currentPage={currentPage} mounted={mountedRef.current}>
        <SettingsPage />
      </PageSlot>
    </main>
  )
}

// ─── Internal ───────────────────────────────────────────────

function PageSlot({
  page,
  currentPage,
  mounted,
  children,
}: {
  page: Page
  currentPage: Page
  mounted: Set<Page>
  children: React.ReactNode
}) {
  if (!mounted.has(page)) return null
  return (
    <div className={cn("absolute inset-0", currentPage !== page && "hidden")}>{children}</div>
  )
}

/** 临时占位组件，后续被各模块 Page 组件替换 */
function Placeholder({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center h-full text-muted-foreground">
      <p className="text-sm">{label}</p>
    </div>
  )
}
