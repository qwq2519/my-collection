import { useEffect } from "react"
import { cn, formatRelativeTime } from "@/lib/utils"
import { useNoteStore } from "@/stores/note"
import { useInfiniteScroll } from "@/hooks/useInfiniteScroll"
import { EmptyState } from "@/components/EmptyState"
import { FileText, Loader2 } from "lucide-react"

export function NoteList() {
  const notes = useNoteStore((s) => s.notes)
  const loading = useNoteStore((s) => s.notesLoading)
  const hasMore = useNoteStore((s) => s.notesHasMore)
  const selectedId = useNoteStore((s) => s.selectedId)
  const loadNotes = useNoteStore((s) => s.loadNotes)
  const loadMore = useNoteStore((s) => s.loadMoreNotes)
  const selectNote = useNoteStore((s) => s.selectNote)

  const sentinelRef = useInfiniteScroll(loadMore, hasMore)

  useEffect(() => {
    loadNotes()
  }, [loadNotes])

  if (!loading && notes.length === 0) {
    return <EmptyState icon={FileText} message="暂无笔记" className="h-full" />
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      {notes.map((note) => (
        <button
          key={note.id}
          onClick={() => selectNote(note.id)}
          className={cn(
            "flex flex-col gap-0.5 px-3 py-2.5 text-left border-b border-border/50 transition-colors",
            "hover:bg-accent/50",
            selectedId === note.id && "bg-accent",
          )}
        >
          <span className="text-sm font-medium truncate">
            {note.title || "无标题"}
          </span>
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(note.updated_at)}
          </span>
        </button>
      ))}

      {loading && (
        <div className="flex justify-center py-3">
          <Loader2 size={16} className="animate-spin text-muted-foreground" />
        </div>
      )}

      {hasMore && <div ref={sentinelRef} className="h-4 shrink-0" />}
    </div>
  )
}
