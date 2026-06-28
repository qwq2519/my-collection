import { create } from "zustand"
import { DEFAULT_THEME_KEY } from "@/lib/themes"

/**
 * 一级页面枚举，对应侧边栏的 5 个导航项。
 * 用字符串字面量而非 enum，Zustand 序列化更简单。
 */
export type Page = "url" | "notes" | "media" | "tags" | "settings"

interface AppState {
  /** 当前激活的一级页面 */
  currentPage: Page
  /** 侧边栏是否折叠（折叠后只显示图标） */
  sidebarCollapsed: boolean
  /** 当前主题 key，对应 themes.ts 中的预设 */
  themeKey: string

  /** 切换到指定页面 */
  setPage: (page: Page) => void
  /** 切换侧边栏折叠状态 */
  toggleSidebar: () => void
  /** 设置主题（后续接入 SettingService 持久化） */
  setTheme: (key: string) => void
}

export const useAppStore = create<AppState>((set) => ({
  currentPage: "url",
  sidebarCollapsed: false,
  themeKey: DEFAULT_THEME_KEY,

  setPage: (page) => set({ currentPage: page }),
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
  setTheme: (key) => set({ themeKey: key }),
}))
