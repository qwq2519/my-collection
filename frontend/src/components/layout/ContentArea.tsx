import { useAppStore } from "@/stores/app"
import { URLPage } from "@/features/url/URLPage"
import { SettingsPage } from "@/features/settings/SettingsPage"

/**
 * 内容区容器：根据 Zustand store 中的 currentPage 渲染对应功能模块。
 *
 * 本项目不使用 React Router，而是用 Zustand store 管理"当前页面"。
 * 每个 {currentPage === "xxx" && <XxxPage />} 相当于路由匹配：
 *   当 currentPage 等于 "xxx" 时渲染对应组件，否则不渲染（短路求值）。
 *
 * 各模块组件在 features/ 下实现后逐步替换 Placeholder 占位。
 */
export function ContentArea() {
  const currentPage = useAppStore((s) => s.currentPage)

  return (
    <main className="flex-1 h-full overflow-hidden">
      {currentPage === "url" && <URLPage />}
      {currentPage === "notes" && <Placeholder label="笔记" />}
      {currentPage === "media" && <Placeholder label="媒体" />}
      {currentPage === "tags" && <Placeholder label="标签管理" />}
      {currentPage === "settings" && <SettingsPage />}
    </main>
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
