import { useState } from "react"
import { useURLStore } from "@/stores/url"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { LoadingState } from "@/components/LoadingState"
import { TagList } from "@/components/TagList"
import { ExternalUrl } from "@/components/ExternalUrl"
import {
  ArrowLeft,
  FileText,
  ImageIcon,
  Video,
  Pencil,
  Trash2,
} from "lucide-react"
import { IMAGE_EXTS, VIDEO_EXTS, isPreviewableExt, formatDate } from "@/lib/utils"
import { getFileExt } from "@/lib/safe"
import { toast } from "sonner"
import { BookmarkForm } from "./BookmarkForm"
import { ThumbnailImage } from "./BookmarkViews"
import { URLService } from "../../../bindings/collections/internal/service"
import type { Bookmark } from "../../../bindings/collections/internal/model"
import { callService } from "@/lib/async"


/**
 * 书签详情面板：协调器，管理展示态/编辑态切换。
 *
 * 组件结构：
 *   BookmarkDetail（协调器：view/edit 模式切换）
 *     → BookmarkNav（顶部导航栏：面包屑 + 编辑/删除按钮）
 *     → BookmarkInfo（正文信息：标题、URL、描述、标签、状态）
 *     → AttachmentGallery（附件画廊：缩略图预览 + 文件列表）
 *
 * useURLStore.getState() 直接调用说明：
 *   在 onSave 回调中使用 getState() 而非 selector，因为这是事件处理器
 *   （非渲染逻辑），不需要订阅变化，只需在触发时读取最新状态。
 */
export function BookmarkDetail() {
  const bm = useURLStore((s) => s.currentBookmark)
  const [editing, setEditing] = useState(false)

  if (!bm) {
    return <LoadingState />
  }

  if (editing) {
    return (
      <div className="h-full overflow-y-auto">
        <BookmarkForm
          bookmark={bm}
          onSave={() => {
            setEditing(false)
            useURLStore.getState().selectBookmark(bm.id)
            useURLStore.getState().refreshCurrentSite()
          }}
          onCancel={() => setEditing(false)}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <BookmarkNav bm={bm} onEdit={() => setEditing(true)} />
      <BookmarkInfo bm={bm} />
      {bm.attachments && bm.attachments.length > 0 && (
        <>
          <Separator />
          <AttachmentGallery bookmarkId={bm.id} attachments={bm.attachments} />
        </>
      )}
    </div>
  )
}

// ─── 顶部导航栏：返回 + 编辑 + 删除 ─────────────────────────

function BookmarkNav({ bm, onEdit }: { bm: Bookmark; onEdit: () => void }) {
  const backToSite = useURLStore((s) => s.backToSite)
  const currentSite = useURLStore((s) => s.currentSite)
  const refreshCurrentSite = useURLStore((s) => s.refreshCurrentSite)
  const [showDelete, setShowDelete] = useState(false)

  const handleDelete = async () => {
    const [, err] = await callService(() => URLService.DeleteBookmark(bm.id))
    if (err) {
      toast.error(err)
      return
    }
    toast.success("书签已删除")
    backToSite()
    refreshCurrentSite()
  }

  return (
    <div className="flex items-center gap-2 px-4 py-2 border-b">
      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={backToSite}>
        <ArrowLeft size={16} />
      </Button>
      <div className="flex items-center gap-1 flex-1 min-w-0 text-xs text-muted-foreground">
        <span className="hover:text-foreground cursor-pointer shrink-0" onClick={backToSite}>
          {currentSite?.title || "站点"}
        </span>
        <span className="shrink-0">&gt;</span>
        <span className="truncate">{bm.title}</span>
      </div>
      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onEdit}>
        <Pencil size={14} />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground hover:text-destructive"
        onClick={() => setShowDelete(true)}
      >
        <Trash2 size={14} />
      </Button>

      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="删除书签"
        description={`确定删除「${bm.title}」？此操作不可撤销。`}
        confirmLabel="删除"
        onConfirm={handleDelete}
      />
    </div>
  )
}

// ─── 书签正文信息 ─────────────────────────────────────────────

function BookmarkInfo({ bm }: { bm: Bookmark }) {
  return (
    <div className="px-6 pt-5 pb-4">
      <h1 className="text-base font-semibold">{bm.title}</h1>
      <ExternalUrl href={bm.url} className="mt-1" />

      {bm.description && <p className="text-sm text-muted-foreground mt-3">{bm.description}</p>}

      <TagList tags={bm.tags} className="mt-3" />

      <div className="flex items-center gap-4 mt-4 text-xs text-muted-foreground">
        <span>状态: {bm.status === "alive" ? "正常" : "失效"}</span>
        {bm.created_at && <span>创建: {formatDate(bm.created_at)}</span>}
        {bm.updated_at && <span>更新: {formatDate(bm.updated_at)}</span>}
      </div>
    </div>
  )
}

// ─── 附件画廊 ─────────────────────────────────────────────────

interface Attachment {
  filename: string
  label: string
}

function AttachmentGallery({
  bookmarkId,
  attachments,
}: {
  bookmarkId: string
  attachments: Attachment[]
}) {
  return (
    <div className="px-6 py-4">
      <h3 className="text-sm font-medium mb-3">附件 ({attachments.length})</h3>
      <div className="@container">
        <div className="grid grid-cols-2 @[480px]:grid-cols-3 gap-2">
          {attachments.map((att) => {
            const ext = getFileExt(att.filename)
            return (
              <div key={att.filename} className="rounded-md border overflow-hidden">
                {isPreviewableExt(ext) && (
                  <div className="aspect-[16/10] bg-muted overflow-hidden">
                    <ThumbnailImage bookmarkId={bookmarkId} filename={att.filename} />
                  </div>
                )}
                <div className="px-2 py-1.5 flex items-center gap-1.5">
                  <AttachmentIcon ext={ext} />
                  <span className="text-xs truncate flex-1">{att.label || att.filename}</span>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// ─── 辅助 ──────────────────────────────────────────────────────

function AttachmentIcon({ ext }: { ext: string }) {
  if (IMAGE_EXTS.has(ext)) return <ImageIcon size={12} className="shrink-0 text-muted-foreground" />
  if (VIDEO_EXTS.has(ext)) return <Video size={12} className="shrink-0 text-muted-foreground" />
  return <FileText size={12} className="shrink-0 text-muted-foreground" />
}
