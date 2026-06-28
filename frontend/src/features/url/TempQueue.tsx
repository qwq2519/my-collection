import { useEffect, useState } from "react"
import { useURLStore } from "@/stores/url"
import { Button } from "@/components/ui/button"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { EmptyState } from "@/components/EmptyState"
import { Loader2, Trash2, ExternalLink, Inbox } from "lucide-react"

/**
 * 临时队列面板：展示所有未归属站点的暂存 URL。
 */
export function TempQueue() {
  const items = useURLStore((s) => s.queueItems)
  const loading = useURLStore((s) => s.queueLoading)
  const loadQueue = useURLStore((s) => s.loadQueue)
  const deleteItem = useURLStore((s) => s.deleteQueueItem)
  const clearAll = useURLStore((s) => s.clearQueue)

  const [showClearConfirm, setShowClearConfirm] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  useEffect(() => {
    loadQueue()
  }, [loadQueue])

  const handleDelete = async (id: string) => {
    setDeletingId(id)
    try {
      await deleteItem(id)
    } finally {
      setDeletingId(null)
    }
  }

  const handleClear = async () => {
    await clearAll()
    setShowClearConfirm(false)
  }

  if (loading && items.length === 0) {
    return (
      <div className="flex items-center justify-center h-32">
        <Loader2 size={20} className="animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* 标题栏 */}
      <div className="flex items-center justify-between px-6 pt-5 pb-3">
        <h2 className="text-base font-semibold">临时队列</h2>
        {items.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 text-xs px-2 gap-1 text-destructive hover:text-destructive"
            onClick={() => setShowClearConfirm(true)}
          >
            <Trash2 size={12} /> 清空
          </Button>
        )}
      </div>

      <p className="px-6 pb-3 text-xs text-muted-foreground">
        当新建书签找不到对应站点时，URL 会暂存在此。请先创建站点后再将其归档。
      </p>

      {/* 队列列表 */}
      <div className="flex-1 overflow-y-auto px-6 pb-4">
        {items.length === 0 ? (
          <EmptyState icon={Inbox} message="队列为空" />
        ) : (
          <div className="flex flex-col gap-1">
            {items.map((item) => (
              <div
                key={item.id}
                className="flex items-center gap-3 px-3 py-2 rounded-md border hover:bg-muted/50 transition-colors group"
              >
                <ExternalLink size={14} className="text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="text-sm truncate">{item.url}</div>
                  <div className="text-xs text-muted-foreground">
                    {formatTime(item.added_at)}
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive transition-opacity"
                  disabled={deletingId === item.id}
                  onClick={() => handleDelete(item.id)}
                >
                  {deletingId === item.id ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <Trash2 size={12} />
                  )}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 清空确认弹窗 */}
      <ConfirmDialog
        open={showClearConfirm}
        onOpenChange={setShowClearConfirm}
        title="清空临时队列"
        description={`确定清空全部 ${items.length} 条暂存 URL？此操作不可撤销。`}
        confirmLabel="清空"
        onConfirm={handleClear}
      />
    </div>
  )
}

function formatTime(t: any): string {
  try {
    const d = new Date(typeof t === "string" ? t : t?.toString?.())
    if (isNaN(d.getTime())) return ""
    return d.toLocaleString("zh-CN", {
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    })
  } catch {
    return ""
  }
}
