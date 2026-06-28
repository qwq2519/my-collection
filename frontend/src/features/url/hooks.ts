import { useEffect } from "react"
import { useURLStore } from "@/stores/url"

/**
 * URL 模块初始化 hook。
 * 在 URLPage 挂载时加载站点列表，保证进入页面就有数据。
 */
export function useURLInit() {
  const loadSites = useURLStore((s) => s.loadSites)

  useEffect(() => {
    loadSites()
  }, [loadSites])
}
