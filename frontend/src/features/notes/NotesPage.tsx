import { useNoteStore } from "@/stores/note"
import { SearchBar } from "@/components/SearchBar"
import { NoteList } from "./NoteList"
import { NoteEditor } from "./NoteEditor"
import { EmptyState } from "@/components/EmptyState"
import { Button } from "@/components/ui/button"
import { FileText, Plus } from "lucide-react"
import { toast } from "sonner"

export function NotesPage() {
  const searchMode = useNoteStore((s) => s.searchMode)
  const searchQuery = useNoteStore((s) => s.searchQuery)
  const search = useNoteStore((s) => s.search)
  const clearSearch = useNoteStore((s) => s.clearSearch)
  const selectedId = useNoteStore((s) => s.selectedId)
  const currentNote = useNoteStore((s) => s.currentNote)
  const currentLoading = useNoteStore((s) => s.currentLoading)
  const createNote = useNoteStore((s) => s.createNote)

  const handleCreate = async () => {
    const err = await createNote()
    if (err) toast.error(err)
  }

  return (
    <div className="flex h-full">
      {/* 左栏：搜索 + 新建 + 笔记列表 */}
      <div className="w-[260px] shrink-0 border-r border-border flex flex-col h-full">
        <div className="flex items-center gap-2 p-3 border-b border-border">
          <SearchBar
            value={searchMode ? searchQuery : ""}
            onChange={search}
            onClear={clearSearch}
            placeholder="搜索笔记..."
            className="flex-1"
          />
          <Button variant="ghost" size="icon" className="shrink-0 h-8 w-8" onClick={handleCreate}>
            <Plus size={16} />
          </Button>
        </div>
        <div className="flex-1 overflow-hidden">
          <NoteList />
        </div>
      </div>

      {/* 右栏：Markdown 编辑器 */}
      <div className="flex-1 h-full overflow-hidden">
        {!selectedId && (
          <EmptyState icon={FileText} message="选择一篇笔记" className="h-full" />
        )}
        {selectedId && currentLoading && (
          <EmptyState icon={FileText} message="加载中..." className="h-full" />
        )}
        {selectedId && !currentLoading && currentNote && (
          <NoteEditor key={currentNote.id} note={currentNote} />
        )}
        {selectedId && !currentLoading && !currentNote && (
          <EmptyState icon={FileText} message="笔记加载失败" className="h-full" />
        )}
      </div>
    </div>
  )
}
