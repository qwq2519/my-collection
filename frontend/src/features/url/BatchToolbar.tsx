import { useState } from "react"
import { Button } from "@/components/ui/button"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { TagInput } from "@/components/TagInput"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Trash2, Tags, X, Loader2 } from "lucide-react"
import { URLService } from "../../../bindings/collections/internal/service"
import { extractError } from "@/lib/utils"
import { toast } from "sonner"

interface BatchToolbarProps {
  siteId: string
  selectedIds: Set<string>
  totalCount: number
  onSelectAll: () => void
  onDeselectAll: () => void
  /** 操作完成后回调（刷新数据） */
  onDone: () => void
  onCancel: () => void
}

/**
 * 批量操作栏：显示已选数量，提供全选/取消、批量删除、批量打标签。
 */
export function BatchToolbar({
  siteId,
  selectedIds,
  totalCount,
  onSelectAll,
  onDeselectAll,
  onDone,
  onCancel,
}: BatchToolbarProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [showTagDialog, setShowTagDialog] = useState(false)
  const [tagsToAdd, setTagsToAdd] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [tagError, setTagError] = useState("")

  const count = selectedIds.size

  const handleBatchDelete = async () => {
    if (count === 0) return
    await URLService.BatchDeleteBookmarks(siteId, Array.from(selectedIds))
    toast.success(`已删除 ${count} 条书签`)
    onDone()
  }

  const handleBatchTag = async () => {
    if (count === 0 || tagsToAdd.length === 0) return
    setLoading(true)
    setTagError("")
    try {
      await URLService.BatchTagBookmarks(Array.from(selectedIds), tagsToAdd)
      toast.success(`已为 ${count} 条书签添加标签`)
      setShowTagDialog(false)
      setTagsToAdd([])
      onDone()
    } catch (e: unknown) {
      const msg = extractError(e)
      setTagError(msg)
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div className="flex items-center gap-2 px-6 py-1.5 bg-muted/50 border-b">
        <span className="text-xs text-muted-foreground">
          已选 {count} / {totalCount}
        </span>
        <Button variant="ghost" size="sm" className="h-6 text-xs px-2" onClick={count === totalCount ? onDeselectAll : onSelectAll}>
          {count === totalCount ? "取消全选" : "全选"}
        </Button>
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          className="h-6 text-xs px-2 gap-1"
          disabled={count === 0}
          onClick={() => setShowTagDialog(true)}
        >
          <Tags size={12} /> 打标签
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="h-6 text-xs px-2 gap-1 text-destructive hover:text-destructive"
          disabled={count === 0}
          onClick={() => setShowDeleteConfirm(true)}
        >
          <Trash2 size={12} /> 删除
        </Button>
        <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onCancel}>
          <X size={14} />
        </Button>
      </div>

      {/* 批量删除确认 */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="批量删除书签"
        description={`确定删除选中的 ${count} 条书签？此操作不可撤销。`}
        confirmLabel="删除"
        onConfirm={handleBatchDelete}
      />

      {/* 批量打标签弹窗 */}
      <Dialog open={showTagDialog} onOpenChange={setShowTagDialog}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle>批量添加标签</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            为选中的 {count} 条书签追加标签（不覆盖已有标签）
          </p>
          <TagInput value={tagsToAdd} onChange={setTagsToAdd} placeholder="输入标签后按回车" />
          {tagError && <p className="text-sm text-destructive">{tagError}</p>}
          <DialogFooter>
            <Button variant="ghost" size="sm" onClick={() => setShowTagDialog(false)}>
              取消
            </Button>
            <Button size="sm" disabled={tagsToAdd.length === 0 || loading} onClick={handleBatchTag}>
              {loading && <Loader2 size={14} className="animate-spin mr-1" />}
              添加
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
