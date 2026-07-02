import { useState, useEffect, useCallback, useRef } from "react"
import MDEditor from "@uiw/react-md-editor"
import { useNoteStore } from "@/stores/note"
import { NoteService } from "../../../bindings/collections/internal/service"
import type { Note } from "../../../bindings/collections/internal/model"
import { callService } from "@/lib/async"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Save, Loader2 } from "lucide-react"
import { toast } from "sonner"

interface NoteEditorProps {
  note: Note
}

export function NoteEditor({ note }: NoteEditorProps) {
  const [title, setTitle] = useState(note.title)
  const [body, setBody] = useState(note.body)
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)
  const dirtyRef = useRef(false)

  const refreshList = useNoteStore((s) => s.refreshList)
  const selectNote = useNoteStore((s) => s.selectNote)

  useEffect(() => {
    setTitle(note.title)
    setBody(note.body)
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
      setBody(value ?? "")
      markDirty()
    },
    [markDirty],
  )

  const save = useCallback(async () => {
    if (!dirtyRef.current) return
    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      toast.error("标题不能为空")
      return
    }
    setSaving(true)
    const [result, err] = await callService(() =>
      NoteService.UpdateNote({ id: note.id, title: trimmedTitle, body }),
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
  }, [title, body, note.id, refreshList, selectNote])

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
          NoteService.UpdateNote({ id: note.id, title: trimmedTitle, body })
        }
      }
    }
  }, [note.id]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="flex flex-col h-full">
      {/* 顶部：标题 + 保存按钮 */}
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
      </div>

      {/* Markdown 编辑器 */}
      <div className="flex-1 overflow-hidden" data-color-mode="light">
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
