import { useState, useEffect, useCallback, useRef } from "react"
import MDEditor from "@uiw/react-md-editor"
import { useNoteStore } from "@/stores/note"
import { NoteService } from "../../../bindings/collections/internal/service"
import { UploadService } from "../../../bindings/collections/internal/service"
import type { Note } from "../../../bindings/collections/internal/model"
import { callService } from "@/lib/async"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Save, Trash2, Loader2 } from "lucide-react"
import { toast } from "sonner"

const IMAGE_MIME: Record<string, string> = {
  "image/png": ".png",
  "image/jpeg": ".jpg",
  "image/gif": ".gif",
  "image/webp": ".webp",
  "image/bmp": ".bmp",
}

interface NoteEditorProps {
  note: Note
}

export function NoteEditor({ note }: NoteEditorProps) {
  const [title, setTitle] = useState(note.title)
  const [body, setBody] = useState(note.body)
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)
  const dirtyRef = useRef(false)
  const bodyRef = useRef(body)

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [orphanFiles, setOrphanFiles] = useState<string[]>([])
  const [showOrphanConfirm, setShowOrphanConfirm] = useState(false)

  const refreshList = useNoteStore((s) => s.refreshList)
  const selectNote = useNoteStore((s) => s.selectNote)
  const deleteNote = useNoteStore((s) => s.deleteNote)

  useEffect(() => {
    setTitle(note.title)
    setBody(note.body)
    bodyRef.current = note.body
    setDirty(false)
    dirtyRef.current = false
  }, [note.id, note.title, note.body])

  const markDirty = useCallback(() => {
    setDirty(true)
    dirtyRef.current = true
  }, [])

  const handleTitleChange = useCallback(
    (value: string) => {
      setTitle(value)
      markDirty()
    },
    [markDirty],
  )

  const handleBodyChange = useCallback(
    (value: string | undefined) => {
      const v = value ?? ""
      setBody(v)
      bodyRef.current = v
      markDirty()
    },
    [markDirty],
  )

  // ── Save ──

  const save = useCallback(async () => {
    if (!dirtyRef.current) return
    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      toast.error("标题不能为空")
      return
    }
    setSaving(true)
    const currentBody = bodyRef.current
    const [result, err] = await callService(() =>
      NoteService.UpdateNote({ id: note.id, title: trimmedTitle, body: currentBody }),
    )
    setSaving(false)
    if (err) {
      toast.error(err)
      return
    }
    setDirty(false)
    dirtyRef.current = false
    if (result) {
      selectNote(result.id)
    }
    refreshList()
    toast.success("已保存")

    // 保存后检测孤儿图片
    const [orphanResult] = await callService(() =>
      NoteService.DetectOrphanImages({ note_id: note.id, body: currentBody }),
    )
    if (orphanResult && orphanResult.orphan_files.length > 0) {
      setOrphanFiles(orphanResult.orphan_files)
      setShowOrphanConfirm(true)
    }
  }, [title, note.id, refreshList, selectNote])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault()
        save()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [save])

  // 组件卸载或切换笔记前自动保存
  useEffect(() => {
    return () => {
      if (dirtyRef.current) {
        const trimmedTitle = title.trim()
        if (trimmedTitle) {
          NoteService.UpdateNote({ id: note.id, title: trimmedTitle, body: bodyRef.current })
        }
      }
    }
  }, [note.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Paste Image Upload ──

  const editorRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = editorRef.current
    if (!container) return

    const handlePaste = async (e: ClipboardEvent) => {
      const items = e.clipboardData?.items
      if (!items) return

      for (let i = 0; i < items.length; i++) {
        const item = items[i]
        const ext = IMAGE_MIME[item.type]
        if (!ext) continue

        e.preventDefault()
        const file = item.getAsFile()
        if (!file) continue

        const base64 = await readFileAsBase64(file)
        const filename = `paste-${Date.now()}${ext}`
        const [result, err] = await callService(() =>
          UploadService.UploadFile({
            scene: "note-image",
            entity_id: note.id,
            filename,
            data: base64,
          }),
        )
        if (err) {
          toast.error(`图片上传失败：${err}`)
          return
        }
        if (result?.path) {
          const imageMarkdown = `![image](/persist/${result.path})`
          const newBody = insertAtCursor(container, imageMarkdown, bodyRef.current)
          setBody(newBody)
          bodyRef.current = newBody
          markDirty()
        }
        return
      }
    }

    container.addEventListener("paste", handlePaste)
    return () => container.removeEventListener("paste", handlePaste)
  }, [note.id, markDirty])

  return (
    <div className="flex flex-col h-full">
      {/* 顶部：标题 + 操作按钮 */}
      <div className="flex items-center gap-3 px-6 py-4 border-b border-border">
        <Input
          value={title}
          onChange={(e) => handleTitleChange(e.target.value)}
          placeholder="笔记标题"
          className="text-lg font-semibold border-none shadow-none px-0 focus-visible:ring-0"
        />
        <Button
          variant="ghost"
          size="sm"
          onClick={save}
          disabled={!dirty || saving}
          className="shrink-0"
        >
          {saving ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
          <span className="ml-1.5">保存</span>
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowDeleteConfirm(true)}
          className="shrink-0 text-destructive hover:text-destructive"
        >
          <Trash2 size={16} />
        </Button>
      </div>

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="删除笔记"
        description={`确定删除「${note.title}」？此操作不可撤销。`}
        onConfirm={async () => {
          dirtyRef.current = false
          const err = await deleteNote(note.id)
          if (err) toast.error(err)
        }}
      />

      <ConfirmDialog
        open={showOrphanConfirm}
        onOpenChange={setShowOrphanConfirm}
        title="清理未引用图片"
        description={`发现 ${orphanFiles.length} 个未被引用的图片文件，是否删除？`}
        confirmLabel="删除"
        onConfirm={async () => {
          const [, err] = await callService(() =>
            NoteService.DeleteOrphanImages({ note_id: note.id, files: orphanFiles }),
          )
          if (err) toast.error(err)
          else toast.success("已清理孤儿图片")
          setOrphanFiles([])
        }}
      />

      {/* Markdown 编辑器 */}
      <div ref={editorRef} className="flex-1 overflow-hidden" data-color-mode="light">
        <MDEditor
          value={body}
          onChange={handleBodyChange}
          height="100%"
          visibleDragbar={false}
          preview="live"
        />
      </div>
    </div>
  )
}

// ── Helpers ──

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve(result.split(",")[1] ?? "")
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

function insertAtCursor(container: HTMLElement, text: string, currentBody: string): string {
  const textarea = container.querySelector("textarea")
  if (!textarea) {
    return currentBody ? currentBody + "\n" + text : text
  }

  const { selectionStart, selectionEnd } = textarea
  const before = textarea.value.substring(0, selectionStart)
  const after = textarea.value.substring(selectionEnd)
  const newValue = before + text + after

  const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype,
    "value",
  )?.set
  if (nativeInputValueSetter) {
    nativeInputValueSetter.call(textarea, newValue)
    textarea.dispatchEvent(new Event("input", { bubbles: true }))
  }

  return newValue
}
