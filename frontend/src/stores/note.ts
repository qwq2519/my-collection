import { create } from "zustand"
import { NoteService } from "../../bindings/collections/internal/service"
import type { Note } from "../../bindings/collections/internal/model"
import { callService } from "../lib/async"
import { unpackList } from "../lib/safe"

const PAGE_SIZE = 50

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
  createNote: () => Promise<string | null>
  deleteNote: (id: string) => Promise<string | null>
}

let searchVersion = 0

export const useNoteStore = create<NoteState>((set, get) => ({
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

  loadNotes: async () => {
    set({ notesLoading: true })
    const [result] = await callService(() =>
      NoteService.ListNotes({ page: 1, page_size: PAGE_SIZE }),
    )
    const { items, total, hasMore } = unpackList(result)
    set({
      notes: items,
      notesTotal: total,
      notesPage: 1,
      notesHasMore: hasMore,
      notesLoading: false,
    })
  },

  loadMoreNotes: async () => {
    const { notesHasMore, notesLoading, notesPage, notes, searchQuery, searchMode } = get()
    if (!notesHasMore || notesLoading) return
    const nextPage = notesPage + 1
    set({ notesLoading: true })
    const [result] = await callService(() =>
      NoteService.ListNotes({
        page: nextPage,
        page_size: PAGE_SIZE,
        search: searchMode ? searchQuery.trim() : undefined,
      }),
    )
    const { items, total, hasMore } = unpackList(result)
    set({
      notes: [...notes, ...items],
      notesTotal: total,
      notesPage: nextPage,
      notesHasMore: hasMore,
      notesLoading: false,
    })
  },

  selectNote: async (id) => {
    set({ selectedId: id, currentNote: null, currentLoading: true })
    const [note] = await callService(() => NoteService.GetNote(id))
    if (get().selectedId !== id) return
    set({ currentNote: note ?? null, currentLoading: false })
  },

  clearSelection: () => {
    set({ selectedId: null, currentNote: null, currentLoading: false })
  },

  search: async (query) => {
    if (!query.trim()) {
      get().clearSearch()
      return
    }
    const version = ++searchVersion
    set({ searchMode: true, searchQuery: query, notesLoading: true })
    const [result] = await callService(() =>
      NoteService.ListNotes({ page: 1, page_size: PAGE_SIZE, search: query.trim() }),
    )
    if (searchVersion !== version) {
      set({ notesLoading: false })
      return
    }
    const { items, total, hasMore } = unpackList(result)
    set({
      notes: items,
      notesTotal: total,
      notesPage: 1,
      notesHasMore: hasMore,
      notesLoading: false,
    })
  },

  clearSearch: () => {
    ++searchVersion
    set({ searchMode: false, searchQuery: "" })
    get().loadNotes()
  },

  refreshList: async () => {
    const { searchMode, searchQuery } = get()
    if (searchMode) {
      get().search(searchQuery)
    } else {
      get().loadNotes()
    }
  },

  createNote: async () => {
    const [note, err] = await callService(() =>
      NoteService.CreateNote({ title: "无标题笔记" }),
    )
    if (err) return err
    if (note) {
      await get().refreshList()
      set({ selectedId: note.id, currentNote: note, currentLoading: false })
    }
    return null
  },

  deleteNote: async (id) => {
    const [, err] = await callService(() => NoteService.DeleteNote(id))
    if (err) return err
    const { selectedId } = get()
    if (selectedId === id) {
      set({ selectedId: null, currentNote: null, currentLoading: false })
    }
    await get().refreshList()
    return null
  },
}))
