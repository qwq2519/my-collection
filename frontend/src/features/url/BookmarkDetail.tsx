import { useState } from "react"
import { useURLStore } from "@/stores/url"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { ArrowLeft, ExternalLink, Loader2, FileText, ImageIcon, Video, Pencil, Trash2 } from "lucide-react"
import { BookmarkForm } from "./BookmarkForm"
import { URLService } from "../../../bindings/collections/internal/service"

/**
 * 书签详情面板：展示态 / 编辑态。
 */
export function BookmarkDetail() {
  const bm = useURLStore((s) => s.currentBookmark)
  const backToSite = useURLStore((s) => s.backToSite)
  const refreshCurrentSite = useURLStore((s) => s.refreshCurrentSite)
  const selectBookmark = useURLStore((s) => s.selectBookmark)
  const [editing, setEditing] = useState(false)
  const [showDelete, setShowDelete] = useState(false)

  const handleDelete = async () => {
    if (!bm) return
    try {
      await URLService.DeleteBookmark(bm.id)
      setShowDelete(false)
      backToSite()
      refreshCurrentSite()
    } catch {
      // 错误由 ConfirmDialog 内部处理
    }
  }

  if (!bm) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 size={20} className="animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (editing) {
    return (
      <BookmarkForm
        bookmark={bm}
        onSave={() => {
          setEditing(false)
          selectBookmark(bm.id)
          refreshCurrentSite()
        }}
        onCancel={() => setEditing(false)}
      />
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {/* 顶部导航 */}
      <div className="flex items-center gap-2 px-4 py-2 border-b">
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={backToSite}>
          <ArrowLeft size={16} />
        </Button>
        <span className="text-xs text-muted-foreground flex-1">返回站点</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setEditing(true)}>
          <Pencil size={14} />
        </Button>
        <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={() => setShowDelete(true)}>
          <Trash2 size={14} />
        </Button>
      </div>

      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="删除书签"
        description={`确定删除「${bm.title}」？此操作不可撤销。`}
        confirmLabel="删除"
        onConfirm={handleDelete}
      />

      {/* 书签信息 */}
      <div className="px-6 pt-5 pb-4">
        <h1 className="text-base font-semibold">{bm.title}</h1>
        <a
          href={bm.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-muted-foreground hover:text-primary inline-flex items-center gap-1 mt-1"
        >
          {bm.url}
          <ExternalLink size={10} />
        </a>

        {bm.description && (
          <p className="text-sm text-muted-foreground mt-3">{bm.description}</p>
        )}

        {bm.tags && bm.tags.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-3">
            {bm.tags.map((tag) => (
              <Badge key={tag} variant="secondary">{tag}</Badge>
            ))}
          </div>
        )}

        {/* 元信息 */}
        <div className="flex items-center gap-4 mt-4 text-xs text-muted-foreground">
          <span>状态: {bm.status === "alive" ? "正常" : "失效"}</span>
          {bm.created_at && (
            <span>创建: {formatTime(bm.created_at)}</span>
          )}
          {bm.updated_at && (
            <span>更新: {formatTime(bm.updated_at)}</span>
          )}
        </div>
      </div>

      {/* 附件 */}
      {bm.attachments && bm.attachments.length > 0 && (
        <>
          <Separator />
          <div className="px-6 py-4">
            <h3 className="text-sm font-medium mb-3">
              附件 ({bm.attachments.length})
            </h3>
            <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
              {bm.attachments.map((att) => {
                const ext = att.filename.split(".").pop()?.toLowerCase() ?? ""
                const isImage = imageExts.has(ext)
                const isVideo = videoExts.has(ext)

                return (
                  <div key={att.filename} className="rounded-md border overflow-hidden">
                    {isImage || isVideo ? (
                      <div className="aspect-[16/10] bg-muted overflow-hidden">
                        <img
                          src={`/persist/url-assets/attachments/${bm.id}/${att.filename}.thumb.jpg`}
                          alt={att.label || att.filename}
                          className="w-full h-full object-cover"
                          onError={(e) => {
                            const img = e.currentTarget
                            if (!img.dataset.fallback) {
                              img.dataset.fallback = "1"
                              img.src = `/persist/url-assets/attachments/${bm.id}/${att.filename}`
                            }
                          }}
                        />
                      </div>
                    ) : null}
                    <div className="px-2 py-1.5 flex items-center gap-1.5">
                      <AttachmentIcon ext={ext} />
                      <span className="text-xs truncate flex-1">
                        {att.label || att.filename}
                      </span>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// ─── 辅助 ──────────────────────────────────────────────────────

const imageExts = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "svg"])
const videoExts = new Set(["mp4", "mkv", "avi", "mov", "webm", "wmv", "flv"])

function AttachmentIcon({ ext }: { ext: string }) {
  if (imageExts.has(ext)) return <ImageIcon size={12} className="shrink-0 text-muted-foreground" />
  if (videoExts.has(ext)) return <Video size={12} className="shrink-0 text-muted-foreground" />
  return <FileText size={12} className="shrink-0 text-muted-foreground" />
}

function formatTime(t: any): string {
  if (!t) return ""
  try {
    const d = typeof t === "string" ? new Date(t) : new Date(t.toString())
    return d.toLocaleDateString("zh-CN")
  } catch {
    return ""
  }
}
