import { useNoteStore } from "@/stores/note"
import { SearchBar } from "@/components/SearchBar"
import { NoteList } from "./NoteList"
import { NoteEditor } from "./NoteEditor"
import { EmptyState } from "@/components/EmptyState"
import { FileText } from "lucide-react"

export function NotesPage() {
  const searchMode = useNoteStore((s) => s.searchMode)
  const searchQuery = useNoteStore((s) => s.searchQuery)
  const search = useNoteStore((s) => s.search)
  const clearSearch = useNoteStore((s) => s.clearSearch)
  const selectedId = useNoteStore((s) => s.selectedId)
  const currentNote = useNoteStore((s) => s.currentNote)
  const currentLoading = useNoteStore((s) => s.currentLoading)

  return (
    <div className="flex h-full">
      {/* 左栏：搜索 + 笔记列表 */}
      <div className="w-[260px] shrink-0 border-r border-border flex flex-col h-full">
        <div className="p-3 border-b border-border">
          <SearchBar
            value={searchMode ? searchQuery : ""}
            onChange={search}
            onClear={clearSearch}
            placeholder="搜索笔记..."
          />
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
      </div>
    </div>
  )
}
