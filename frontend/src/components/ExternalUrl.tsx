import { ExternalLink } from "lucide-react"
import { cn } from "@/lib/utils"

interface ExternalUrlProps {
  href: string
  className?: string
  children?: React.ReactNode
}

/**
 * 外链展示：带 ExternalLink 图标的可点击链接。
 * 新窗口打开，hover 变色，用于详情页中展示 URL。
 */
export function ExternalUrl({ href, className, children }: ExternalUrlProps) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "text-xs text-muted-foreground hover:text-primary inline-flex items-center gap-1",
        className,
      )}
    >
      {children || href}
      <ExternalLink size={10} />
    </a>
  )
}
