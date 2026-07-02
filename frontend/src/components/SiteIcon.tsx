import { useState, useEffect } from "react"
import { Globe } from "lucide-react"
import { cn } from "@/lib/utils"

interface SiteIconProps {
  icon?: string
  size?: number
  className?: string
}

/**
 * 站点图标：加载 /persist/url-assets/icons/ 下的图标，
 * 加载失败时 fallback 到 Globe 图标。
 */
export function SiteIcon({ icon, size = 16, className }: SiteIconProps) {
  const [failed, setFailed] = useState(false)

  // 触发：icon prop 变化时重置 failed 状态，允许新图标尝试加载
  useEffect(() => {
    setFailed(false)
  }, [icon])

  if (!icon || failed) {
    return <Globe size={size} className={cn("shrink-0 text-muted-foreground", className)} />
  }

  return (
    <img
      src={`/persist/url-assets/icons/${icon}`}
      alt=""
      className={cn("shrink-0 rounded-sm", className)}
      style={{ width: size, height: size }}
      onError={() => setFailed(true)}
    />
  )
}
