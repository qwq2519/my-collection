import { useEffect, useRef } from "react"

/**
 * 无限滚动 hook：通过 IntersectionObserver 监听哨兵元素，
 * 当哨兵进入可视区域时自动调用 loadMore 加载下一页。
 *
 * 用法：
 *   const sentinelRef = useInfiniteScroll(loadMore, hasMore)
 *   return (
 *     <div>
 *       {items.map(...)}
 *       {hasMore && <div ref={sentinelRef} className="h-4" />}
 *     </div>
 *   )
 *
 * 工作原理（类比 Go）：
 *   类似于在列表末尾放一个"触发器"，当用户滚动到触发器位置时，
 *   自动调用回调函数加载更多数据。IntersectionObserver 是浏览器原生 API，
 *   用于高效检测元素是否进入视口，无需手动监听 scroll 事件。
 */
export function useInfiniteScroll(
  loadMore: () => void,
  hasMore: boolean,
) {
  const sentinelRef = useRef<HTMLDivElement>(null)

  // 用 ref 持有最新的 loadMore 回调，避免 useEffect 依赖变化导致 observer 反复重建
  const loadMoreRef = useRef(loadMore)
  loadMoreRef.current = loadMore

  useEffect(() => {
    const sentinel = sentinelRef.current
    if (!sentinel || !hasMore) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadMoreRef.current()
        }
      },
      { threshold: 0.1 },
    )
    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore])

  return sentinelRef
}
