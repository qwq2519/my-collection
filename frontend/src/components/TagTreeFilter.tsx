import { useState, useMemo } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Filter, ChevronRight, ChevronDown, X } from "lucide-react"
import { cn } from "@/lib/utils"
import { pick } from "@/lib/safe"

interface TagTreeFilterProps {
  /** 所有可用标签（扁平列表，层级用 :: 分割） */
  allTags: string[]
  /** 当前选中的标签 */
  selectedTags: string[]
  /** 选中标签变化回调 */
  onChange: (tags: string[]) => void
}

/** 标签树节点 */
interface TagNode {
  name: string
  fullPath: string
  children: TagNode[]
}

/**
 * 标签树形筛选面板。
 * 标签按 :: 分割渲染为缩进层级，支持折叠。
 * 点击标签切换选中态，多选标签为 AND 语义。
 */
export function TagTreeFilter({ allTags, selectedTags, onChange }: TagTreeFilterProps) {
  const [open, setOpen] = useState(false)
  // eslint-disable-next-line no-restricted-syntax -- useMemo justified: expensive tree traversal on every tag change
  const tree = useMemo(() => buildTree(allTags), [allTags])

  const toggleTag = (tag: string) => {
    if (selectedTags.includes(tag)) {
      onChange(selectedTags.filter((t) => t !== tag))
    } else {
      onChange([...selectedTags, tag])
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs">
          <Filter size={14} />
          标签筛选
          {selectedTags.length > 0 && (
            <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-xs">
              {selectedTags.length}
            </Badge>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[260px] p-0" align="start">
        {selectedTags.length > 0 && (
          <div className="flex items-center justify-between px-3 py-2 border-b">
            <span className="text-xs text-muted-foreground">已选 {selectedTags.length} 个</span>
            <button
              onClick={() => onChange([])}
              className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-0.5"
            >
              <X size={12} />
              清除
            </button>
          </div>
        )}
        <ScrollArea className="max-h-[300px]">
          <div className="p-2">
            {tree.length === 0 && (
              <p className="text-xs text-muted-foreground text-center py-4">暂无标签</p>
            )}
            {tree.length > 0 &&
              tree.map((node) => (
                <TreeNode
                  key={node.fullPath}
                  node={node}
                  selectedTags={selectedTags}
                  onToggle={toggleTag}
                  depth={0}
                />
              ))}
          </div>
        </ScrollArea>
      </PopoverContent>
    </Popover>
  )
}

/** 递归渲染标签树节点 */
function TreeNode({
  node,
  selectedTags,
  onToggle,
  depth,
}: {
  node: TagNode
  selectedTags: string[]
  onToggle: (tag: string) => void
  depth: number
}) {
  const [expanded, setExpanded] = useState(true)
  const isSelected = selectedTags.includes(node.fullPath)
  const hasChildren = node.children.length > 0

  return (
    <div>
      <button
        className={cn(
          "flex items-center gap-1 w-full rounded-md px-2 py-1 text-left text-sm transition-colors duration-150",
          pick(isSelected, "bg-muted font-medium", "hover:bg-muted/50"),
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={() => onToggle(node.fullPath)}
      >
        {/* 折叠按钮：有子节点时显示 */}
        {hasChildren && (
          <span
            className="shrink-0 text-muted-foreground"
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
          >
            {pick(expanded, <ChevronDown size={12} />, <ChevronRight size={12} />)}
          </span>
        )}
        {!hasChildren && <span className="w-3 shrink-0" />}
        <span className="truncate">{node.name}</span>
      </button>

      {hasChildren && expanded && (
        <div>
          {node.children.map((child) => (
            <TreeNode
              key={child.fullPath}
              node={child}
              selectedTags={selectedTags}
              onToggle={onToggle}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * 将扁平标签列表构建为树形结构。
 * 如 ["开发", "开发::前端", "开发::后端", "工具"] →
 * [{ name: "开发", children: [{ name: "前端" }, { name: "后端" }] }, { name: "工具" }]
 */
function buildTree(tags: string[]): TagNode[] {
  const root: TagNode[] = []
  const sorted = [...tags].sort()

  for (const tag of sorted) {
    const parts = tag.split("::")
    let current = root

    for (let i = 0; i < parts.length; i++) {
      const fullPath = parts.slice(0, i + 1).join("::")
      let existing = current.find((n) => n.fullPath === fullPath)
      if (!existing) {
        existing = { name: parts[i], fullPath, children: [] }
        current.push(existing)
      }
      current = existing.children
    }
  }

  return root
}
