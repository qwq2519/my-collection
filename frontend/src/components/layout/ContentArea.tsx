import { useAppStore } from "@/stores/app"

/**
 * 内容区容器：根据当前页面渲染对应功能模块。
 * 各模块组件在 features/ 下实现后逐步替换占位。
 */
export function ContentArea() {
  const currentPage = useAppStore((s) => s.currentPage)

  return (
    <main className="flex-1 h-full overflow-hidden">
      {currentPage === "url" && <Placeholder label="URL 收藏" />}
      {currentPage === "notes" && <Placeholder label="笔记" />}
      {currentPage === "media" && <Placeholder label="媒体" />}
      {currentPage === "tags" && <Placeholder label="标签管理" />}
      {currentPage === "settings" && <Placeholder label="设置" />}
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
