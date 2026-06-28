import { useCallback, useEffect, useRef, useState, type ClipboardEvent, type DragEvent } from "react"
import { Button } from "@/components/ui/button"
import { Upload, X, Loader2, FileText, ImageIcon, Video, Clipboard } from "lucide-react"
import { UploadService } from "../../bindings/collections/internal/service"
import { cn, extractError, IMAGE_EXTS, VIDEO_EXTS } from "@/lib/utils"

const TEXT_EXTS = new Set(["txt"])
const ALL_EXTS = new Set([...IMAGE_EXTS, ...VIDEO_EXTS, ...TEXT_EXTS])

/** 根据 MIME type 推断扩展名（粘贴的文件通常没有文件名） */
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

/**
 * 通用文件上传组件：支持粘贴（Ctrl+V）、拖拽、点击选择三种方式。
 * 粘贴时自动识别剪贴板中的文件（截图、复制的文件等）。
 */
export function FileUpload({ scene, entityId, files, onChange, className }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const dropZoneRef = useRef<HTMLDivElement>(null)
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [error, setError] = useState("")
  const [focused, setFocused] = useState(false)

  const uploadFiles = useCallback(async (fileList: File[]) => {
    setError("")
    const validFiles: File[] = []
    for (const file of fileList) {
      const ext = file.name.split(".").pop()?.toLowerCase() ?? ""
      if (!ALL_EXTS.has(ext)) {
        setError(`不支持的文件格式：${file.name}`)
        continue
      }
      validFiles.push(file)
    }
    if (validFiles.length === 0) return

    setUploading(true)
    try {
      const newFiles: UploadedFile[] = []
      for (const file of validFiles) {
        const base64 = await readFileAsBase64(file)
        const result = await UploadService.UploadFile({
          scene,
          entity_id: entityId,
          filename: file.name,
          data: base64,
        })
        if (result?.path) {
          newFiles.push({ filename: file.name, path: result.path })
        }
      }
      onChange([...files, ...newFiles])
    } catch (e: any) {
      setError(extractError(e))
    } finally {
      setUploading(false)
    }
  }, [scene, entityId, files, onChange])

  const handleFileInput = (fileList: FileList) => {
    uploadFiles(Array.from(fileList))
  }

  const handleDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    if (e.dataTransfer.files.length > 0) {
      handleFileInput(e.dataTransfer.files)
    }
  }

  /** 从剪贴板中提取文件（截图、复制的文件），为无文件名的 blob 生成名称 */
  const handlePaste = useCallback((e: ClipboardEvent | globalThis.ClipboardEvent) => {
    const items = (e as ClipboardEvent).clipboardData?.items
      ?? (e as globalThis.ClipboardEvent).clipboardData?.items
    if (!items || items.length === 0) return

    const pastedFiles: File[] = []
    for (const item of Array.from(items)) {
      if (item.kind !== "file") continue
      const file = item.getAsFile()
      if (!file) continue

      // 粘贴的截图通常文件名为 "image.png"，用时间戳区分
      let name = file.name
      if (!name || name === "image.png" || name === "image.jpeg") {
        const ext = MIME_TO_EXT[file.type] ?? "png"
        name = `paste-${Date.now()}.${ext}`
      }

      // 检查扩展名是否合法
      const ext = name.split(".").pop()?.toLowerCase() ?? ""
      if (!ALL_EXTS.has(ext)) {
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
  }, [uploadFiles])

  // 全局 paste 监听：当上传区域获得焦点时响应 Ctrl+V
  useEffect(() => {
    const handler = (e: globalThis.ClipboardEvent) => {
      if (focused) handlePaste(e)
    }
    document.addEventListener("paste", handler)
    return () => document.removeEventListener("paste", handler)
  }, [focused, handlePaste])

  const handleDelete = async (filename: string) => {
    try {
      await UploadService.DeleteAttachment(entityId, filename)
      onChange(files.filter((f) => f.filename !== filename))
    } catch (e: any) {
      setError(extractError(e))
    }
  }

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {/* 上传区域：点击 / 拖拽 / 粘贴 */}
      <div
        ref={dropZoneRef}
        tabIndex={0}
        className={cn(
          "flex flex-col items-center justify-center gap-1 rounded-md border border-dashed py-4 cursor-pointer transition-colors duration-150 outline-none",
          dragOver ? "border-primary bg-muted/50" : focused ? "border-primary/60 bg-muted/30" : "border-input hover:border-primary/50"
        )}
        onClick={() => inputRef.current?.click()}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onPaste={handlePaste as any}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
      >
        {uploading ? (
          <Loader2 size={20} className="animate-spin text-muted-foreground" />
        ) : (
          <Upload size={20} className="text-muted-foreground" />
        )}
        <p className="text-xs text-muted-foreground text-center">
          {uploading ? "上传中..." : (
            <>
              点击选择 / 拖拽 / <Clipboard size={10} className="inline -mt-0.5" /> 粘贴文件
            </>
          )}
        </p>
        <input
          ref={inputRef}
          type="file"
          multiple
          className="hidden"
          accept={Array.from(ALL_EXTS).map((e) => `.${e}`).join(",")}
          onChange={(e) => {
            if (e.target.files) handleFileInput(e.target.files)
            e.target.value = ""
          }}
        />
      </div>

      {error && <p className="text-xs text-destructive">{error}</p>}

      {/* 已上传文件列表 */}
      {files.length > 0 && (
        <div className="flex flex-col gap-1">
          {files.map((f) => {
            const ext = f.filename.split(".").pop()?.toLowerCase() ?? ""
            return (
              <div key={f.filename} className="flex items-center gap-2 px-2 py-1 rounded-md hover:bg-muted/50">
                <FileTypeIcon ext={ext} />
                <span className="flex-1 text-xs truncate">{f.filename}</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5 shrink-0"
                  onClick={() => handleDelete(f.filename)}
                >
                  <X size={12} />
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

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
