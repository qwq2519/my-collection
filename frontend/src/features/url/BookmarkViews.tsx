/**
 * 书签视图组件：网格视图和列表视图。
 *
 * 从 SiteDetail.tsx 中拆出，职责单一——只负责书签列表的渲染，
 * 不涉及站点详情、批量操作等逻辑。
 *
 * 组件关系：
 *   SiteDetail（协调器）
 *     → BookmarkSection（工具栏 + 批量操作 + 内容区）
 *       → BookmarkGrid / BookmarkListView（本文件）
 */

import { useState } from "react"
import { cn, isPreviewableExt, VIDEO_EXTS } from "@/lib/utils"
import { getFileExt } from "@/lib/safe"
import { Video } from "lucide-react"
import type { Bookmark } from "../../../bindings/collections/internal/model"


// ─── 书签网格视图 ─────────────────────────────────────────────

export function BookmarkGrid({
  bookmarks,
  onSelect,
  batchMode = false,
  selectedIds = new Set(),
}: {
  bookmarks: Bookmark[]
  onSelect: (id: string) => void
  batchMode?: boolean
  selectedIds?: Set<string>
}) {
  return (
    <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
      {bookmarks.map((bm) => {
        const cover = getCoverAttachment(bm)
        const isSelected = selectedIds.has(bm.id)
        return (
          <button
            key={bm.id}
            onClick={() => onSelect(bm.id)}
            className={cn(
              "flex flex-col rounded-md border overflow-hidden text-left transition-colors duration-150",
              isSelected ? "ring-2 ring-primary" : "hover:bg-muted/50",
            )}
          >
            <div className="aspect-[16/10] bg-muted flex items-center justify-center overflow-hidden">
              {cover ? (
                <ThumbnailImage bookmarkId={bm.id} filename={cover.filename} />
              ) : (
                <span className="text-2xl font-medium text-muted-foreground/50 select-none">
                  {bm.title.charAt(0)}
                </span>
              )}
            </div>
            <div className="px-2 py-1.5 flex items-center gap-1.5">
              {batchMode && (
                <input type="checkbox" checked={isSelected} readOnly className="rounded shrink-0" />
              )}
              <div className="text-sm font-medium truncate">{bm.title}</div>
            </div>
          </button>
        )
      })}
    </div>
  )
}

// ─── 书签列表视图 ─────────────────────────────────────────────

export function BookmarkListView({
  bookmarks,
  onSelect,
  batchMode = false,
  selectedIds = new Set(),
}: {
  bookmarks: Bookmark[]
  onSelect: (id: string) => void
  batchMode?: boolean
  selectedIds?: Set<string>
}) {
  return (
    <div className="flex flex-col">
      {bookmarks.map((bm) => {
        const isSelected = selectedIds.has(bm.id)
        return (
          <button
            key={bm.id}
            onClick={() => onSelect(bm.id)}
            className={cn(
              "flex items-center gap-3 px-3 py-2 rounded-md text-left transition-colors duration-150",
              isSelected ? "bg-muted" : "hover:bg-muted/50",
            )}
          >
            {batchMode && (
              <input type="checkbox" checked={isSelected} readOnly className="rounded shrink-0" />
            )}
            <div className="text-sm font-medium truncate">{bm.title}</div>
          </button>
        )
      })}
    </div>
  )
}

// ─── 缩略图（自动 fallback 到原图或视频图标） ───────────────
//
// 优先加载 .thumb.jpg 缩略图。如果缩略图不存在：
// - 图片文件：fallback 到原始文件
// - 视频文件：显示通用视频图标（视频无法在 img 标签中渲染）

export function ThumbnailImage({ bookmarkId, filename }: { bookmarkId: string; filename: string }) {
  const ext = getFileExt(filename)
  const isVideo = VIDEO_EXTS.has(ext)
  const [fallback, setFallback] = useState<"none" | "original" | "icon">("none")

  if (fallback === "icon") {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <Video size={32} className="text-muted-foreground/50" />
      </div>
    )
  }

  return (
    <img
      src={fallback === "original"
        ? `/persist/url-assets/attachments/${bookmarkId}/${filename}`
        : `/persist/url-assets/attachments/${bookmarkId}/${filename}.thumb.jpg`}
      alt=""
      className="w-full h-full object-cover"
      onError={() => {
        if (fallback === "none") {
          setFallback(isVideo ? "icon" : "original")
        } else {
          setFallback("icon")
        }
      }}
    />
  )
}

// ─── 工具函数 ─────────────────────────────────────────────────

/** 从书签的附件列表中找到第一个可预览的图片/视频作为封面 */
function getCoverAttachment(bm: Bookmark) {
  if (!bm.attachments || bm.attachments.length === 0) return null
  return bm.attachments.find((a) => isPreviewableExt(getFileExt(a.filename))) ?? null
}
