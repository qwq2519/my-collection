import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

interface TagListProps {
  tags?: string[] | null
  className?: string
}

/**
 * 标签展示列表：以 Badge 横排展示标签。
 * 用于详情面板中展示站点/书签的标签，tags 为空时不渲染任何内容。
 */
export function TagList({ tags, className }: TagListProps) {
  if (!tags || tags.length === 0) return null

  return (
    <div className={cn("flex flex-wrap gap-1", className)}>
      {tags.map((tag) => (
        <Badge key={tag} variant="secondary">
          {tag}
        </Badge>
      ))}
    </div>
  )
}
