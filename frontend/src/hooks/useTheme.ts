import { useEffect } from "react"
import { useAppStore } from "@/stores/app"
import { getThemeByKey, CSS_VAR_KEYS, DEFAULT_THEME_KEY } from "@/lib/themes"

/**
 * 主题注入 hook。
 *
 * 在 App 根组件中调用一次即可。监听 Zustand store 中的 themeKey 变化，
 * 从 themes 注册表取出对应色值，遍历写入 document.documentElement.style。
 *
 * ink 主题（默认）的色值已在 index.css :root 中定义，
 * 切换到 ink 时清除 JS 写入的 inline style，回退到 CSS 默认值。
 */
export function useTheme() {
  const themeKey = useAppStore((s) => s.themeKey)

  useEffect(() => {
    const root = document.documentElement
    const theme = getThemeByKey(themeKey)

    if (!theme || theme.key === DEFAULT_THEME_KEY) {
      // ink 是 CSS 默认值，清除所有 inline 覆盖即可回退
      CSS_VAR_KEYS.forEach((key) => root.style.removeProperty(key))
      return
    }

    // 将主题色值逐条写入 CSS 变量
    CSS_VAR_KEYS.forEach((key) => {
      const value = theme.colors[key]
      if (value) {
        root.style.setProperty(key, value)
      }
    })
  }, [themeKey])
}
