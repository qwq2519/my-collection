import { useRef, useState, type DragEvent } from "react"
import { Button } from "@/components/ui/button"
import { Upload, X, Loader2, FileText, ImageIcon, Video } from "lucide-react"
import { UploadService } from "../../bindings/collections/internal/service"
import { cn } from "@/lib/utils"

/** 允许的图片扩展名 */
const IMAGE_EXTS = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "svg"])
/** 允许的视频扩展名 */
const VIDEO_EXTS = new Set(["mp4", "mkv", "avi", "mov", "webm", "wmv", "flv"])
/** 允许的文本扩展名 */
const TEXT_EXTS = new Set(["txt"])

const ALL_EXTS = new Set([...IMAGE_EXTS, ...VIDEO_EXTS, ...TEXT_EXTS])

interface UploadedFile {
  filename: string
  path: string
}

interface FileUploadProps {
  /** 上传场景，决定存储路径 */
  scene: string
  /** 关联实体 ID */
  entityId: string
  /** 已上传的文件列表（展示用） */
  files: UploadedFile[]
  /** 文件列表变化回调 */
  onChange: (files: UploadedFile[]) => void
  className?: string
}

/**
 * 通用文件上传组件：拖拽或点击选择文件，读取为 base64 后调用 UploadService。
 * 支持图片、视频、txt 文件。上传完成后返回存储路径。
 */
export function FileUpload({ scene, entityId, files, onChange, className }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [error, setError] = useState("")

  const handleFiles = async (fileList: FileList) => {
    setError("")
    const validFiles: File[] = []
    for (const file of Array.from(fileList)) {
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
      setError(e?.message ?? "上传失败")
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = (e: DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    if (e.dataTransfer.files.length > 0) {
      handleFiles(e.dataTransfer.files)
    }
  }

  const handleDelete = async (filename: string) => {
    try {
      await UploadService.DeleteAttachment(entityId, filename)
      onChange(files.filter((f) => f.filename !== filename))
    } catch (e: any) {
      setError(e?.message ?? "删除失败")
    }
  }

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {/* 拖拽上传区域 */}
      <div
        className={cn(
          "flex flex-col items-center justify-center gap-1 rounded-md border border-dashed py-4 cursor-pointer transition-colors duration-150",
          dragOver ? "border-primary bg-muted/50" : "border-input hover:border-primary/50"
        )}
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
      >
        {uploading ? (
          <Loader2 size={20} className="animate-spin text-muted-foreground" />
        ) : (
          <Upload size={20} className="text-muted-foreground" />
        )}
        <p className="text-xs text-muted-foreground">
          {uploading ? "上传中..." : "点击或拖拽文件到此处"}
        </p>
        <input
          ref={inputRef}
          type="file"
          multiple
          className="hidden"
          accept={Array.from(ALL_EXTS).map((e) => `.${e}`).join(",")}
          onChange={(e) => {
            if (e.target.files) handleFiles(e.target.files)
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

/** 将 File 对象读取为不含 data URL 前缀的纯 base64 字符串 */
function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      // 去掉 "data:xxx;base64," 前缀
      const base64 = result.split(",")[1] ?? ""
      resolve(base64)
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}
