import { type LucideIcon, Inbox } from "lucide-react"
import { cn } from "@/lib/utils"

interface EmptyStateProps {
  icon?: LucideIcon
  message?: string
  className?: string
  children?: React.ReactNode
}

/**
 * 通用空状态占位组件。
 * 居中灰色图标 + 说明文字，可选操作引导按钮通过 children 传入。
 */
export function EmptyState({
  icon: Icon = Inbox,
  message = "暂无数据",
  className,
  children,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-12 text-muted-foreground",
        className,
      )}
    >
      <Icon size={32} className="mb-2" />
      <p className="text-sm">{message}</p>
      {children}
    </div>
  )
}
