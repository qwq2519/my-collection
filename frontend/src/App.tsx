import { TooltipProvider } from "@/components/ui/tooltip"
import { Sidebar } from "@/components/layout/Sidebar"
import { ContentArea } from "@/components/layout/ContentArea"
import { useTheme } from "@/hooks/useTheme"

function App() {
  // 应用启动时注入当前主题 CSS 变量
  useTheme()

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex h-screen w-screen overflow-hidden">
        <Sidebar />
        <ContentArea />
      </div>
    </TooltipProvider>
  )
}

export default App
