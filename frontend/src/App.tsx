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
