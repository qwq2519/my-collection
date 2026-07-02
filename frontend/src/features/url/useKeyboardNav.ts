import { useEffect, useRef } from "react"
import { useURLStore, getActiveSiteId } from "@/stores/url"
import { useAppStore } from "@/stores/app"


/**
 * 键盘导航：↑↓ 切换站点列表项，Enter 选中，Esc 返回/清除搜索。
 * 在 input/textarea/select 聚焦时不拦截键盘事件。
 */
export function useKeyboardNav() {
  const focusedIndexRef = useRef(-1)

  const detailView = useURLStore((s) => s.detailView)
  const sites = useURLStore((s) => s.sites)
  const searchResults = useURLStore((s) => s.searchResults)
  const searchMode = useURLStore((s) => s.searchMode)

  // 触发：detailView/sites/searchResults 变化时，同步 focusedIndex 与当前选中项
  useEffect(() => {
    const selectedId = getActiveSiteId(detailView)
    if (!selectedId) {
      focusedIndexRef.current = -1
      return
    }
    const ids = searchMode
      ? searchResults.map((r) => r.site.id)
      : sites.map((s) => s.id)
    const idx = ids.indexOf(selectedId)
    if (idx !== -1) focusedIndexRef.current = idx
  }, [detailView, sites, searchResults, searchMode])

  // 触发：组件挂载时注册全局 keydown 监听，卸载时清理
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (useAppStore.getState().currentPage !== "url") return
      if (document.querySelector("[role=dialog]")) return

      if (e.target instanceof HTMLElement) {
        const tag = e.target.tagName
        if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return
        if (e.target.isContentEditable) return
      }

      const state = useURLStore.getState()
      const listLen = state.searchMode ? state.searchResults.length : state.sites.length

      switch (e.key) {
        case "ArrowDown": {
          if (listLen === 0) return
          e.preventDefault()
          const cur = focusedIndexRef.current
          const next = cur < 0 ? 0 : Math.min(cur + 1, listLen - 1)
          focusedIndexRef.current = next
          selectByIndex(state, next)
          scrollToIndex(next)
          break
        }
        case "ArrowUp": {
          if (listLen === 0) return
          e.preventDefault()
          const next = focusedIndexRef.current <= 0 ? 0 : focusedIndexRef.current - 1
          focusedIndexRef.current = next
          selectByIndex(state, next)
          scrollToIndex(next)
          break
        }
        case "Enter": {
          if (focusedIndexRef.current < 0 || focusedIndexRef.current >= listLen) return
          e.preventDefault()
          selectByIndex(state, focusedIndexRef.current)
          break
        }
        case "Escape": {
          if (state.detailView.type === "bookmark") {
            e.preventDefault()
            state.backToSite()
          } else if (state.searchMode) {
            e.preventDefault()
            state.clearSearch()
          }
          break
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [])
}

function selectByIndex(state: ReturnType<typeof useURLStore.getState>, index: number) {
  if (state.searchMode) {
    const item = state.searchResults[index]
    if (item) state.selectSearchResult(item)
  } else {
    const site = state.sites[index]
    if (site) state.selectSite(site.id)
  }
}

function scrollToIndex(index: number) {
  requestAnimationFrame(() => {
    document.querySelector(`[data-site-index="${index}"]`)?.scrollIntoView({ block: "nearest" })
  })
}
