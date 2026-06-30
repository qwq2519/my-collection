import { useState } from "react"
import { Globe } from "lucide-react"

interface SiteIconProps {
  icon?: string
  size?: number
  className?: string
}

/**
 * 站点图标：加载 /persist/url-assets/icons/ 下的图标，
 * 加载失败时 fallback 到 Globe 图标。
 */
export function SiteIcon({ icon, size = 16, className = "" }: SiteIconProps) {
  const [failed, setFailed] = useState(false)

  if (!icon || failed) {
    return <Globe size={size} className={`shrink-0 text-muted-foreground ${className}`} />
  }

  return (
    <img
      src={`/persist/url-assets/icons/${icon}`}
      alt=""
      className={`shrink-0 rounded-sm ${className}`}
      style={{ width: size, height: size }}
      onError={() => setFailed(true)}
    />
  )
}
