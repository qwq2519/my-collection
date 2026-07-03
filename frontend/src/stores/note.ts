import { create, type StateCreator } from "zustand"
import { NoteService } from "../../bindings/collections/internal/service"
import type { Note } from "../../bindings/collections/internal/model"
import { callService } from "../lib/async"
import { unpackList } from "../lib/safe"
import { toast } from "sonner"

const PAGE_SIZE = 50

let noteListVersion = 0
let searchVersion = 0

interface NoteState {
  notes: Note[]
  notesTotal: number
  notesHasMore: boolean
  notesLoading: boolean
  notesPage: number

  selectedId: string | null
  currentNote: Note | null
  currentLoading: boolean

  searchQuery: string
  searchMode: boolean

  loadNotes: () => Promise<void>
  loadMoreNotes: () => Promise<void>
  selectNote: (id: string) => Promise<void>
  clearSelection: () => void
  search: (query: string) => Promise<void>
  clearSearch: () => void
  refreshList: () => Promise<void>
  /** 保存后直接更新 currentNote，避免 selectNote 导致组件卸载重载 */
  updateCurrentNote: (note: Note) => void
  createNote: () => Promise<string | null>
  deleteNote: (id: string) => Promise<string | null>
}

type NoteStoreCreator = StateCreator<NoteState>
type NoteSet = Parameters<NoteStoreCreator>[0]
type NoteGet = Parameters<NoteStoreCreator>[1]

const initialNoteState: Pick<
  NoteState,
  | "notes"
  | "notesTotal"
  | "notesHasMore"
  | "notesLoading"
  | "notesPage"
  | "selectedId"
  | "currentNote"
  | "currentLoading"
  | "searchQuery"
  | "searchMode"
> = {
  notes: [],
  notesTotal: 0,
  notesHasMore: false,
  notesLoading: false,
  notesPage: 1,
  selectedId: null,
  currentNote: null,
  currentLoading: false,
  searchQuery: "",
  searchMode: false,
}

function applyNotePage(
  set: NoteSet,
  notes: Note[],
  total: number,
  hasMore: boolean,
  page: number,
) {
  set({
    notes,
    notesTotal: total,
    notesPage: page,
    notesHasMore: hasMore,
    notesLoading: false,
  })
}

function createLoadNotes(set: NoteSet): NoteState["loadNotes"] {
  return async () => {
    const version = ++noteListVersion
    set({ notesLoading: true })
    const [result, err] = await callService(() =>
      NoteService.ListNotes({ page: 1, page_size: PAGE_SIZE }),
    )
    if (noteListVersion !== version) return
    if (err) {
      toast.error("加载笔记列表失败：" + err)
      set({ notesLoading: false })
      return
    }
    const { items, total, hasMore } = unpackList(result)
    applyNotePage(set, items, total, hasMore, 1)
  }
}

function createLoadMoreNotes(set: NoteSet, get: NoteGet): NoteState["loadMoreNotes"] {
  return async () => {
    const { notesHasMore, notesLoading, notesPage, searchQuery, searchMode } = get()
    if (!notesHasMore || notesLoading) return

    const version = noteListVersion
    const nextPage = notesPage + 1
    set({ notesLoading: true })

    const [result, err] = await callService(() =>
      NoteService.ListNotes({
        page: nextPage,
        page_size: PAGE_SIZE,
        search: searchMode ? searchQuery.trim() : undefined,
      }),
    )
    if (noteListVersion !== version) return
    if (err) {
      toast.error("加载更多笔记失败：" + err)
      set({ notesLoading: false })
      return
    }
    const { items, total, hasMore } = unpackList(result)
    applyNotePage(set, [...get().notes, ...items], total, hasMore, nextPage)
  }
}

function createSelectNote(set: NoteSet, get: NoteGet): NoteState["selectNote"] {
  return async (id) => {
    set({ selectedId: id, currentNote: null, currentLoading: true })
    const [note, err] = await callService(() => NoteService.GetNote(id))
    if (get().selectedId !== id) {
      set({ currentLoading: false })
      return
    }
    if (err) {
      toast.error("加载笔记失败：" + err)
      set({ currentNote: null, currentLoading: false })
      return
    }
    set({ currentNote: note ?? null, currentLoading: false })
  }
}

function createSearch(set: NoteSet, get: NoteGet): NoteState["search"] {
  return async (query) => {
    if (!query.trim()) {
      get().clearSearch()
      return
    }

    const version = ++searchVersion
    const listVersion = ++noteListVersion
    set({ searchMode: true, searchQuery: query, notesLoading: true })

    const [result, err] = await callService(() =>
      NoteService.ListNotes({ page: 1, page_size: PAGE_SIZE, search: query.trim() }),
    )
    if (searchVersion !== version || noteListVersion !== listVersion) return
    if (err) {
      toast.error("搜索笔记失败：" + err)
      set({ notesLoading: false })
      return
    }
    const { items, total, hasMore } = unpackList(result)
    applyNotePage(set, items, total, hasMore, 1)
  }
}

function createRefreshList(get: NoteGet): NoteState["refreshList"] {
  return async () => {
    const { searchMode, searchQuery } = get()
    if (searchMode) {
      await get().search(searchQuery)
      return
    }
    await get().loadNotes()
  }
}

function createNote(set: NoteSet, get: NoteGet): NoteState["createNote"] {
  return async () => {
    const [note, err] = await callService(() =>
      NoteService.CreateNote({ title: "无标题笔记" }),
    )
    if (err) return err
    if (note) {
      await get().refreshList()
      set({ selectedId: note.id, currentNote: note, currentLoading: false })
    }
    return null
  }
}

function createDeleteNote(set: NoteSet, get: NoteGet): NoteState["deleteNote"] {
  return async (id) => {
    const [, err] = await callService(() => NoteService.DeleteNote(id))
    if (err) return err

    if (get().selectedId === id) {
      set({ selectedId: null, currentNote: null, currentLoading: false })
    }
    await get().refreshList()
    return null
  }
}

export const useNoteStore = create<NoteState>((set, get) => ({
  ...initialNoteState,
  loadNotes: createLoadNotes(set),
  loadMoreNotes: createLoadMoreNotes(set, get),
  selectNote: createSelectNote(set, get),
  clearSelection: () => {
    set({ selectedId: null, currentNote: null, currentLoading: false })
  },
  search: createSearch(set, get),
  clearSearch: () => {
    ++searchVersion
    ++noteListVersion
    set({ searchMode: false, searchQuery: "" })
    get().loadNotes()
  },
  refreshList: createRefreshList(get),
  updateCurrentNote: (note) => {
    set({ currentNote: note })
  },
  createNote: createNote(set, get),
  deleteNote: createDeleteNote(set, get),
}))
