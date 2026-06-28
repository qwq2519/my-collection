/**
 * 应用根组件。
 *
 * 整体布局：左侧 Sidebar（导航） + 右侧 ContentArea（按当前页面渲染内容）。
 * TooltipProvider 和 Toaster 是全局 UI 基础设施：
 *   - TooltipProvider: 让所有 Tooltip 组件共享延迟等配置
 *   - Toaster: 全局 toast 通知容器（操作成功/失败提示）
 *
 * useTheme() 负责将当前主题的 CSS 变量注入到 <html> 元素上。
 */
import { TooltipProvider } from "@/components/ui/tooltip"
import { Toaster } from "@/components/ui/sonner"
import { Sidebar } from "@/components/layout/Sidebar"
import { ContentArea } from "@/components/layout/ContentArea"
import { useTheme } from "@/hooks/useTheme"

function App() {
  useTheme()

  return (
    <TooltipProvider delayDuration={300}>
      <div className="flex h-screen w-screen overflow-hidden">
        <Sidebar />
        <ContentArea />
      </div>
      <Toaster position="bottom-right" duration={3000} />
    </TooltipProvider>
  )
}

export default App
