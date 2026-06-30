import { Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"

interface LoadingStateProps {
  className?: string
}

/**
 * 通用加载状态占位组件。
 * 居中旋转图标，与 EmptyState 对称使用。
 */
export function LoadingState({ className }: LoadingStateProps) {
  return (
    <div className={cn("flex items-center justify-center h-full", className)}>
      <Loader2 size={20} className="animate-spin text-muted-foreground" />
    </div>
  )
}
