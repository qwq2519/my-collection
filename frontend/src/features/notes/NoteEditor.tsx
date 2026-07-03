import {
  useState,
  useEffect,
  useRef,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react"
import MDEditor from "@uiw/react-md-editor"
import { useNoteStore } from "@/stores/note"
import { useAppStore } from "@/stores/app"
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

interface NoteDraftState {
  title: string
  body: string
  dirty: boolean
  bodyRef: MutableRefObject<string>
  dirtyRef: MutableRefObject<boolean>
  titleRef: MutableRefObject<string>
  setBody: Dispatch<SetStateAction<string>>
  setDirty: Dispatch<SetStateAction<boolean>>
  markDirty: () => void
  handleTitleChange: (value: string) => void
  handleBodyChange: (value: string | undefined) => void
}

export function NoteEditor({ note }: NoteEditorProps) {
  const refreshList = useNoteStore((s) => s.refreshList)
  const updateCurrentNote = useNoteStore((s) => s.updateCurrentNote)
  const deleteNote = useNoteStore((s) => s.deleteNote)
  const currentPage = useAppStore((s) => s.currentPage)

  const draft = useNoteDraft(note)
  const rootRef = useRef<HTMLDivElement>(null)
  const { editorRef, editorHeight } = useEditorHeight()
  const { saving, orphanFiles, showOrphanConfirm, setShowOrphanConfirm, save, handleOrphanDelete } =
    useNoteSave(note.id, draft, updateCurrentNote, refreshList)
  const { showDeleteConfirm, setShowDeleteConfirm, handleDeleteConfirm } = useNoteDelete(
    note.id,
    draft.dirtyRef,
    deleteNote,
  )

  const saveRef = useRef(save)
  saveRef.current = save

  useSaveOnPageLeave(currentPage, saveRef)
  usePersistDraftOnUnmount(note.id, draft.dirtyRef, draft.titleRef, draft.bodyRef)
  useWarnBeforeUnload(draft.dirtyRef)
  useNotePasteUpload(note.id, editorRef, draft.bodyRef, draft.setBody, draft.markDirty)

  return (
    <div
      ref={rootRef}
      className="flex flex-col h-full"
      onKeyDownCapture={(e) => handleEditorKeyDown(e, saveRef)}
      onBlurCapture={() => handleEditorBlur(rootRef, saveRef)}
    >
      <NoteEditorToolbar
        title={draft.title}
        dirty={draft.dirty}
        saving={saving}
        onTitleChange={draft.handleTitleChange}
        onSave={save}
        onDelete={() => setShowDeleteConfirm(true)}
      />
      <NoteEditorDialogs
        noteTitle={note.title}
        orphanFiles={orphanFiles}
        showDeleteConfirm={showDeleteConfirm}
        showOrphanConfirm={showOrphanConfirm}
        setShowDeleteConfirm={setShowDeleteConfirm}
        setShowOrphanConfirm={setShowOrphanConfirm}
        onDeleteConfirm={handleDeleteConfirm}
        onOrphanDelete={handleOrphanDelete}
      />
      <NoteMarkdownEditor
        editorRef={editorRef}
        body={draft.body}
        editorHeight={editorHeight}
        onChange={draft.handleBodyChange}
      />
    </div>
  )
}

function useNoteDraft(note: Note): NoteDraftState {
  const [title, setTitle] = useState(note.title)
  const [body, setBody] = useState(note.body)
  const [dirty, setDirty] = useState(false)
  const dirtyRef = useRef(false)
  const bodyRef = useRef(note.body)
  const titleRef = useRef(note.title)

  useEffect(() => {
    const draftKey = `note-draft:${note.id}`
    const draft = localStorage.getItem(draftKey)
    if (draft) {
      const parsed = JSON.parse(draft) as { title: string; body: string }
      setTitle(parsed.title)
      setBody(parsed.body)
      bodyRef.current = parsed.body
      titleRef.current = parsed.title
      setDirty(true)
      dirtyRef.current = true
      localStorage.removeItem(draftKey)
      toast.info("已恢复上次未保存的修改")
      return
    }
    setTitle(note.title)
    setBody(note.body)
    bodyRef.current = note.body
    titleRef.current = note.title
    setDirty(false)
    dirtyRef.current = false
  }, [note.id, note.title, note.body])

  const markDirty = () => {
    setDirty(true)
    dirtyRef.current = true
  }

  const handleTitleChange = (value: string) => {
    setTitle(value)
    titleRef.current = value
    markDirty()
  }

  const handleBodyChange = (value: string | undefined) => {
    const nextBody = value ?? ""
    setBody(nextBody)
    bodyRef.current = nextBody
    markDirty()
  }

  return {
    title,
    body,
    dirty,
    bodyRef,
    dirtyRef,
    titleRef,
    setBody,
    setDirty,
    markDirty,
    handleTitleChange,
    handleBodyChange,
  }
}

function useNoteSave(
  noteId: string,
  draft: NoteDraftState,
  updateCurrentNote: (note: Note) => void,
  refreshList: () => Promise<void>,
) {
  const [saving, setSaving] = useState(false)
  const [orphanFiles, setOrphanFiles] = useState<string[]>([])
  const [showOrphanConfirm, setShowOrphanConfirm] = useState(false)

  async function save() {
    if (!draft.dirtyRef.current) return
    const trimmedTitle = draft.titleRef.current.trim()
    if (!trimmedTitle) {
      toast.error("标题不能为空")
      return
    }
    setSaving(true)
    const currentBody = draft.bodyRef.current
    const [result, err] = await callService(() =>
      NoteService.UpdateNote({ id: noteId, title: trimmedTitle, body: currentBody }),
    )
    setSaving(false)
    if (err) {
      toast.error(err)
      return
    }
    draft.setDirty(false)
    draft.dirtyRef.current = false
    if (result) updateCurrentNote(result)
    await refreshList()
    toast.success("已保存")
    await detectOrphanImages(noteId, currentBody, setOrphanFiles, setShowOrphanConfirm)
  }

  async function handleOrphanDelete() {
    const [, err] = await callService(() =>
      NoteService.DeleteOrphanImages({ note_id: noteId, files: orphanFiles }),
    )
    if (err) toast.error(err)
    else toast.success("已清理孤儿图片")
    setOrphanFiles([])
  }

  return { saving, orphanFiles, showOrphanConfirm, setShowOrphanConfirm, save, handleOrphanDelete }
}

function useNoteDelete(
  noteId: string,
  dirtyRef: MutableRefObject<boolean>,
  deleteNote: (id: string) => Promise<string | null>,
) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  async function handleDeleteConfirm() {
    const wasDirty = dirtyRef.current
    dirtyRef.current = false
    const err = await deleteNote(noteId)
    if (err) {
      dirtyRef.current = wasDirty
      toast.error(err)
    }
  }

  return { showDeleteConfirm, setShowDeleteConfirm, handleDeleteConfirm }
}

function useSaveOnPageLeave(
  currentPage: string,
  saveRef: MutableRefObject<() => Promise<void>>,
) {
  useEffect(() => {
    if (currentPage !== "notes") {
      saveRef.current()
    }
  }, [currentPage, saveRef])
}

function usePersistDraftOnUnmount(
  noteId: string,
  dirtyRef: MutableRefObject<boolean>,
  titleRef: MutableRefObject<string>,
  bodyRef: MutableRefObject<string>,
) {
  useEffect(() => {
    return () => persistDraftOnUnmount(noteId, dirtyRef, titleRef, bodyRef)
  }, [noteId, dirtyRef, titleRef, bodyRef])
}

function useWarnBeforeUnload(dirtyRef: MutableRefObject<boolean>) {
  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) e.preventDefault()
    }
    window.addEventListener("beforeunload", handleBeforeUnload)
    return () => window.removeEventListener("beforeunload", handleBeforeUnload)
  }, [dirtyRef])
}

function useEditorHeight() {
  const editorRef = useRef<HTMLDivElement>(null)
  const [editorHeight, setEditorHeight] = useState(400)

  useEffect(() => {
    const container = editorRef.current
    if (!container) return
    const ro = new ResizeObserver((entries) => {
      const height = entries[0]?.contentRect.height
      if (height && height > 0) setEditorHeight(height)
    })
    ro.observe(container)
    return () => ro.disconnect()
  }, [])

  return { editorRef, editorHeight }
}

function useNotePasteUpload(
  noteId: string,
  editorRef: MutableRefObject<HTMLDivElement | null>,
  bodyRef: MutableRefObject<string>,
  setBody: Dispatch<SetStateAction<string>>,
  markDirty: () => void,
) {
  useEffect(() => {
    const container = editorRef.current
    if (!container) return

    const handlePaste = async (e: ClipboardEvent) => {
      const item = getFirstPasteImage(e.clipboardData?.items)
      if (!item) return

      e.preventDefault()
      const file = item.getAsFile()
      if (!file) return

      const [base64, readErr] = await callService(() => readFileAsBase64(file))
      if (readErr || base64 === null) {
        toast.error(`图片读取失败：${readErr ?? "unknown error"}`)
        return
      }
      const filename = `paste-${Date.now()}${IMAGE_MIME[item.type]}`
      const [result, err] = await callService(() =>
        UploadService.UploadFile({
          scene: "note-image",
          entity_id: noteId,
          filename,
          data: base64,
        }),
      )
      if (err) {
        toast.error(`图片上传失败：${err}`)
        return
      }
      if (result?.path) {
        const newBody = insertAtCursor(container, `![image](/persist/${result.path})`, bodyRef.current)
        setBody(newBody)
        bodyRef.current = newBody
        markDirty()
      }
    }

    container.addEventListener("paste", handlePaste)
    return () => container.removeEventListener("paste", handlePaste)
  }, [noteId, editorRef, bodyRef, setBody, markDirty])
}

function NoteEditorToolbar({
  title,
  dirty,
  saving,
  onTitleChange,
  onSave,
  onDelete,
}: {
  title: string
  dirty: boolean
  saving: boolean
  onTitleChange: (value: string) => void
  onSave: () => Promise<void>
  onDelete: () => void
}) {
  return (
    <div className="flex items-center gap-3 px-6 py-4 pt-9 border-b border-border">
      <Input
        value={title}
        onChange={(e) => onTitleChange(e.target.value)}
        placeholder="笔记标题"
        className="text-lg font-semibold border-none shadow-none px-0 focus-visible:ring-0"
      />
      <Button variant="ghost" size="sm" onClick={onSave} disabled={!dirty || saving} className="shrink-0">
        {saving ? <Loader2 size={16} className="animate-spin" /> : <Save size={16} />}
        <span className="ml-1.5">保存</span>
      </Button>
      <Button
        variant="ghost"
        size="sm"
        onClick={onDelete}
        className="shrink-0 text-destructive hover:text-destructive"
      >
        <Trash2 size={16} />
      </Button>
    </div>
  )
}

function NoteEditorDialogs({
  noteTitle,
  orphanFiles,
  showDeleteConfirm,
  showOrphanConfirm,
  setShowDeleteConfirm,
  setShowOrphanConfirm,
  onDeleteConfirm,
  onOrphanDelete,
}: {
  noteTitle: string
  orphanFiles: string[]
  showDeleteConfirm: boolean
  showOrphanConfirm: boolean
  setShowDeleteConfirm: (open: boolean) => void
  setShowOrphanConfirm: (open: boolean) => void
  onDeleteConfirm: () => Promise<void>
  onOrphanDelete: () => Promise<void>
}) {
  return (
    <>
      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title="删除笔记"
        description={`确定删除「${noteTitle}」？此操作不可撤销。`}
        onConfirm={onDeleteConfirm}
      />
      <ConfirmDialog
        open={showOrphanConfirm}
        onOpenChange={setShowOrphanConfirm}
        title="清理未引用图片"
        description={`发现 ${orphanFiles.length} 个未被引用的图片文件，是否删除？`}
        confirmLabel="删除"
        onConfirm={onOrphanDelete}
      />
    </>
  )
}

function NoteMarkdownEditor({
  editorRef,
  body,
  editorHeight,
  onChange,
}: {
  editorRef: MutableRefObject<HTMLDivElement | null>
  body: string
  editorHeight: number
  onChange: (value: string | undefined) => void
}) {
  return (
    <div ref={editorRef} className="flex-1 overflow-hidden" data-color-mode="light">
      <MDEditor
        value={body}
        onChange={onChange}
        height={editorHeight}
        visibleDragbar={false}
        preview="live"
      />
    </div>
  )
}

async function detectOrphanImages(
  noteId: string,
  body: string,
  setOrphanFiles: Dispatch<SetStateAction<string[]>>,
  setShowOrphanConfirm: Dispatch<SetStateAction<boolean>>,
) {
  const [result, err] = await callService(() =>
    NoteService.DetectOrphanImages({ note_id: noteId, body }),
  )
  if (err) {
    toast.error("检测未引用图片失败：" + err)
    return
  }
  if (result && result.orphan_files.length > 0) {
    setOrphanFiles(result.orphan_files)
    setShowOrphanConfirm(true)
  }
}

function persistDraftOnUnmount(
  noteId: string,
  dirtyRef: MutableRefObject<boolean>,
  titleRef: MutableRefObject<string>,
  bodyRef: MutableRefObject<string>,
) {
  if (!dirtyRef.current) return
  const trimmedTitle = titleRef.current.trim()
  if (!trimmedTitle) return

  const draft = { id: noteId, title: trimmedTitle, body: bodyRef.current }
  NoteService.UpdateNote(draft).catch(() => {
    localStorage.setItem(`note-draft:${noteId}`, JSON.stringify(draft))
  })
}

function handleEditorKeyDown(
  e: React.KeyboardEvent<HTMLDivElement>,
  saveRef: MutableRefObject<() => Promise<void>>,
) {
  if ((e.ctrlKey || e.metaKey) && e.key === "s") {
    e.preventDefault()
    saveRef.current()
  }
}

function handleEditorBlur(
  rootRef: MutableRefObject<HTMLDivElement | null>,
  saveRef: MutableRefObject<() => Promise<void>>,
) {
  requestAnimationFrame(() => {
    const root = rootRef.current
    const active = document.activeElement
    if (!root || !active || root.contains(active)) return
    saveRef.current()
  })
}

function getFirstPasteImage(items: DataTransferItemList | null | undefined) {
  if (!items) return null
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (IMAGE_MIME[item.type]) return item
  }
  return null
}

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
