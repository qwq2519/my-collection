/**
 * 文件上传组件：支持点击、拖拽、粘贴（Ctrl+V）三种方式上传文件。
 *
 * 结构：
 *   FileUpload（协调器）
 *     → DropZone（拖拽/点击/粘贴上传区）
 *     → UploadedFileList（已上传文件列表，支持拖拽排序和删除）
 *
 * 上传/粘贴/验证逻辑集中在 useFileUpload hook，UI 只负责渲染。
 */

import { useRef, useState, type ClipboardEvent, type DragEvent } from "react"
import { Button } from "@/components/ui/button"
import {
  Upload,
  X,
  Loader2,
  FileText,
  ImageIcon,
  Video,
  Clipboard,
  GripVertical,
} from "lucide-react"
import { UploadService } from "../../bindings/collections/internal/service"
import { cn, IMAGE_EXTS, VIDEO_EXTS } from "@/lib/utils"
import { callService, runAsync } from "@/lib/async"
import { getFileExt } from "@/lib/safe"

const TEXT_EXTS = new Set(["txt"])
const ALL_EXTS = new Set([...IMAGE_EXTS, ...VIDEO_EXTS, ...TEXT_EXTS])

/** MIME → 扩展名映射，粘贴的截图通常没有文件名，需要靠 MIME 推断 */
const MIME_TO_EXT: Record<string, string> = {
  "image/png": "png",
  "image/jpeg": "jpg",
  "image/gif": "gif",
  "image/webp": "webp",
  "image/bmp": "bmp",
  "image/avif": "avif",
  "image/svg+xml": "svg",
  "video/mp4": "mp4",
  "video/webm": "webm",
  "text/plain": "txt",
}

interface UploadedFile {
  filename: string
  path: string
}

interface FileUploadProps {
  scene: string
  entityId: string
  files: UploadedFile[]
  onChange: (files: UploadedFile[]) => void
  className?: string
}

// ─── 主组件 ──────────────────────────────────────────────────

export function FileUpload({ scene, entityId, files, onChange, className }: FileUploadProps) {
  const { uploading, error, setError, uploadFiles, handlePaste } = useFileUpload(
    scene,
    entityId,
    files,
    onChange,
  )

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <DropZone uploading={uploading} uploadFiles={uploadFiles} handlePaste={handlePaste} />
      {error && <p className="text-xs text-destructive">{error}</p>}
      {files.length > 0 && (
        <UploadedFileList files={files} entityId={entityId} onChange={onChange} onError={setError} />
      )}
    </div>
  )
}

// ─── 上传 hook：封装文件验证、上传、粘贴处理 ────────────────
//
// 类比 Go：这相当于把 uploadHandler struct 的方法集中到一起，
// 组件只持有 handler 引用，不关心内部实现。

function useFileUpload(
  scene: string,
  entityId: string,
  files: UploadedFile[],
  onChange: (files: UploadedFile[]) => void,
) {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState("")

  // ref 持有最新闭包值，避免 uploadFiles 闭包过期
  const stateRef = useRef({ scene, entityId, files, onChange })
  stateRef.current = { scene, entityId, files, onChange }

  async function uploadFiles(fileList: File[]) {
    setError("")
    const validFiles: File[] = []
    for (const file of fileList) {
      if (!ALL_EXTS.has(getFileExt(file.name))) {
        setError(`不支持的文件格式：${file.name}`)
        continue
      }
      validFiles.push(file)
    }
    if (validFiles.length === 0) return

    setUploading(true)
    const { scene: s, entityId: eid, files: currentFiles, onChange: cb } = stateRef.current
    const newFiles: UploadedFile[] = []
    for (const file of validFiles) {
      const base64 = await readFileAsBase64(file)
      const [result, err] = await callService(() =>
        UploadService.UploadFile({
          scene: s,
          entity_id: eid,
          filename: file.name,
          data: base64,
        }),
      )
      if (err) {
        setError(err)
        break
      }
      if (result?.path) {
        newFiles.push({ filename: file.name, path: result.path })
      }
    }
    if (newFiles.length > 0) {
      cb([...currentFiles, ...newFiles])
    }
    setUploading(false)
  }

  /** 从剪贴板中提取文件（截图、复制的文件），为无文件名的 blob 生成名称 */
  function handlePaste(e: ClipboardEvent | globalThis.ClipboardEvent) {
    const items =
      (e as ClipboardEvent).clipboardData?.items ??
      (e as globalThis.ClipboardEvent).clipboardData?.items
    if (!items || items.length === 0) return

    const pastedFiles: File[] = []
    for (const item of Array.from(items)) {
      if (item.kind !== "file") continue
      const file = item.getAsFile()
      if (!file) continue

      let name = file.name
      if (!name || name === "image.png" || name === "image.jpeg") {
        const guessed = MIME_TO_EXT[file.type] ?? "png"
        name = `paste-${Date.now()}.${guessed}`
      }

      if (!ALL_EXTS.has(getFileExt(name))) {
        const guessedExt = MIME_TO_EXT[file.type]
        if (guessedExt && ALL_EXTS.has(guessedExt)) {
          name = `paste-${Date.now()}.${guessedExt}`
        }
      }

      pastedFiles.push(new File([file], name, { type: file.type }))
    }

    if (pastedFiles.length > 0) {
      e.preventDefault()
      uploadFiles(pastedFiles)
    }
  }

  return { uploading, error, setError, uploadFiles, handlePaste }
}

// ─── 拖拽/点击/粘贴上传区 ───────────────────────────────────

function DropZone({
  uploading,
  uploadFiles,
  handlePaste,
}: {
  uploading: boolean
  uploadFiles: (fileList: File[]) => void
  handlePaste: (e: ClipboardEvent | globalThis.ClipboardEvent) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragOver, setDragOver] = useState(false)
  const [focused, setFocused] = useState(false)

  const handleDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    if (e.dataTransfer.files.length > 0) {
      uploadFiles(Array.from(e.dataTransfer.files))
    }
  }

  const handleDragOver = (e: DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) uploadFiles(Array.from(e.target.files))
    e.target.value = ""
  }

  function getDropZoneClass(): string {
    if (dragOver) return "border-primary bg-muted/50"
    if (focused) return "border-primary/60 bg-muted/30"
    return "border-input hover:border-primary/50"
  }

  return (
    <div
      tabIndex={0}
      className={cn(
        "flex flex-col items-center justify-center gap-1 rounded-md border border-dashed py-4 cursor-pointer transition-colors duration-150 outline-none",
        getDropZoneClass(),
      )}
      onClick={() => inputRef.current?.click()}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
      onPaste={handlePaste as unknown as React.ClipboardEventHandler}
      onDragOver={handleDragOver}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
    >
      {uploading && <Loader2 size={20} className="animate-spin text-muted-foreground" />}
      {!uploading && <Upload size={20} className="text-muted-foreground" />}
      <DropZoneLabel uploading={uploading} />
      <input
        ref={inputRef}
        type="file"
        multiple
        className="hidden"
        accept={Array.from(ALL_EXTS)
          .map((e) => `.${e}`)
          .join(",")}
        onChange={handleFileInput}
      />
    </div>
  )
}

// ─── 上传区文案 ─────────────────────────────────────────────

function DropZoneLabel({ uploading }: { uploading: boolean }) {
  if (uploading) {
    return <p className="text-xs text-muted-foreground text-center">上传中...</p>
  }
  return (
    <p className="text-xs text-muted-foreground text-center">
      点击选择 / 拖拽 / <Clipboard size={10} className="inline -mt-0.5" /> 粘贴文件
    </p>
  )
}

// ─── 已上传文件列表（支持拖拽排序 + 删除） ──────────────────

function UploadedFileList({
  files,
  entityId,
  onChange,
  onError,
}: {
  files: UploadedFile[]
  entityId: string
  onChange: (files: UploadedFile[]) => void
  onError: (msg: string) => void
}) {
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [overIndex, setOverIndex] = useState<number | null>(null)

  function handleReorderDrop() {
    if (dragIndex === null || overIndex === null || dragIndex === overIndex) {
      setDragIndex(null)
      setOverIndex(null)
      return
    }
    const reordered = [...files]
    const [moved] = reordered.splice(dragIndex, 1)
    reordered.splice(overIndex, 0, moved)
    onChange(reordered)
    setDragIndex(null)
    setOverIndex(null)
  }

  async function handleDelete(filename: string) {
    const err = await runAsync(() => UploadService.DeleteAttachment(entityId, filename))
    if (err) {
      onError(err)
      return
    }
    onChange(files.filter((f) => f.filename !== filename))
  }

  const resetDragState = () => {
    setDragIndex(null)
    setOverIndex(null)
  }

  return (
    <div className="flex flex-col">
      {files.map((f, idx) => (
        <FileListItem
          key={f.filename}
          file={f}
          idx={idx}
          dragIndex={dragIndex}
          overIndex={overIndex}
          onDragStart={setDragIndex}
          onDragOver={setOverIndex}
          onDragEnd={resetDragState}
          onDrop={handleReorderDrop}
          onDelete={handleDelete}
        />
      ))}
    </div>
  )
}

function FileListItem({
  file,
  idx,
  dragIndex,
  overIndex,
  onDragStart,
  onDragOver,
  onDragEnd,
  onDrop,
  onDelete,
}: {
  file: UploadedFile
  idx: number
  dragIndex: number | null
  overIndex: number | null
  onDragStart: (idx: number) => void
  onDragOver: (idx: number) => void
  onDragEnd: () => void
  onDrop: () => void
  onDelete: (filename: string) => void
}) {
  const ext = getFileExt(file.filename)

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    onDragOver(idx)
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    onDrop()
  }

  return (
    <div
      draggable
      onDragStart={() => onDragStart(idx)}
      onDragOver={handleDragOver}
      onDragEnd={onDragEnd}
      onDrop={handleDrop}
      className={cn(
        "flex items-center gap-2 px-2 py-1 rounded-md hover:bg-muted/50",
        dragIndex === idx && "opacity-50",
        overIndex === idx && dragIndex !== null && dragIndex !== idx && "border-t border-primary",
      )}
    >
      <GripVertical size={14} className="shrink-0 text-muted-foreground/50 cursor-grab" />
      <FileTypeIcon ext={ext} />
      <span className="flex-1 text-xs truncate">{file.filename}</span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-5 w-5 shrink-0"
        onClick={() => onDelete(file.filename)}
      >
        <X size={12} />
      </Button>
    </div>
  )
}

// ─── 辅助 ────────────────────────────────────────────────────

function FileTypeIcon({ ext }: { ext: string }) {
  if (IMAGE_EXTS.has(ext)) return <ImageIcon size={14} className="shrink-0 text-muted-foreground" />
  if (VIDEO_EXTS.has(ext)) return <Video size={14} className="shrink-0 text-muted-foreground" />
  return <FileText size={14} className="shrink-0 text-muted-foreground" />
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      const base64 = result.split(",")[1] ?? ""
      resolve(base64)
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}
