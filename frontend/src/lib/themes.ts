/**
 * 主题预设注册中心。
 *
 * 所有主题的 HSL 色值集中定义在此文件，新增主题只需往 themePresets 里加一组。
 * useTheme hook 根据 themeKey 取出对应色值，遍历写入 CSS 变量。
 * index.css 中的 :root 定义了 ink 主题默认值作为兜底。
 *
 * 色值格式："H S% L%"（纯 HSL 三元组，不含 hsl() 包裹），
 * 与 index.css 中 var(--xxx) 搭配使用：hsl(var(--background))。
 */

/** 单个 CSS 变量名 → HSL 值的映射 */
export type ThemeColors = Record<string, string>

export interface ThemePreset {
  /** 内部标识，与 Zustand store 中的 themeKey 对应 */
  key: string
  /** 用户可见的中文名称 */
  label: string
  /** 简短描述，展示在设置页 */
  description: string
  /** CSS 变量 → HSL 色值映射 */
  colors: ThemeColors
}

/**
 * CSS 变量名列表，用于遍历写入 / 清除。
 * 与 index.css 中 :root 定义的变量一一对应。
 */
export const CSS_VAR_KEYS = [
  "--background",
  "--foreground",
  "--primary",
  "--primary-foreground",
  "--secondary",
  "--secondary-foreground",
  "--muted",
  "--muted-foreground",
  "--accent",
  "--accent-foreground",
  "--destructive",
  "--destructive-foreground",
  "--border",
  "--input",
  "--ring",
  "--card",
  "--card-foreground",
  "--popover",
  "--popover-foreground",
  "--sidebar-background",
  "--sidebar-foreground",
  "--sidebar-accent",
  "--sidebar-accent-foreground",
  "--sidebar-border",
] as const

// ─── 四套内置主题 ─────────────────────────────────────────────

/** 水墨 — 纯灰阶，最克制。类似 iA Writer、Typora 的气质。 */
const ink: ThemePreset = {
  key: "ink",
  label: "水墨",
  description: "纯灰阶，最克制",
  colors: {
    "--background": "0 0% 100%",
    "--foreground": "220 10% 12%",
    "--primary": "220 10% 28%",
    "--primary-foreground": "0 0% 98%",
    "--secondary": "220 5% 96%",
    "--secondary-foreground": "220 10% 25%",
    "--muted": "220 5% 96%",
    "--muted-foreground": "220 5% 46%",
    "--accent": "220 5% 94%",
    "--accent-foreground": "220 10% 12%",
    "--destructive": "0 60% 50%",
    "--destructive-foreground": "0 0% 98%",
    "--border": "220 5% 90%",
    "--input": "220 5% 88%",
    "--ring": "220 10% 28%",
    "--card": "0 0% 100%",
    "--card-foreground": "220 10% 12%",
    "--popover": "0 0% 100%",
    "--popover-foreground": "220 10% 12%",
    "--sidebar-background": "220 5% 97%",
    "--sidebar-foreground": "220 10% 12%",
    "--sidebar-accent": "220 5% 93%",
    "--sidebar-accent-foreground": "220 10% 12%",
    "--sidebar-border": "220 5% 91%",
  },
}

/** 暖石 — 暖灰棕，像牛皮纸或旧书页，和"收藏"的语义契合。 */
const stone: ThemePreset = {
  key: "stone",
  label: "暖石",
  description: "暖灰棕，旧书页质感",
  colors: {
    "--background": "40 20% 99%",
    "--foreground": "30 10% 12%",
    "--primary": "30 30% 33%",
    "--primary-foreground": "40 20% 98%",
    "--secondary": "35 15% 95%",
    "--secondary-foreground": "30 12% 25%",
    "--muted": "35 12% 95%",
    "--muted-foreground": "30 8% 46%",
    "--accent": "35 15% 93%",
    "--accent-foreground": "30 10% 12%",
    "--destructive": "0 55% 48%",
    "--destructive-foreground": "40 20% 98%",
    "--border": "35 10% 89%",
    "--input": "35 10% 87%",
    "--ring": "30 30% 33%",
    "--card": "40 20% 99%",
    "--card-foreground": "30 10% 12%",
    "--popover": "40 18% 100%",
    "--popover-foreground": "30 10% 12%",
    "--sidebar-background": "38 18% 96%",
    "--sidebar-foreground": "30 10% 12%",
    "--sidebar-accent": "35 14% 92%",
    "--sidebar-accent-foreground": "30 10% 12%",
    "--sidebar-border": "35 10% 90%",
  },
}

/** 青竹 — 低饱和灰绿，安静自然，长时间看屏幕舒适。 */
const sage: ThemePreset = {
  key: "sage",
  label: "青竹",
  description: "灰绿色，安静自然",
  colors: {
    "--background": "150 10% 99%",
    "--foreground": "150 8% 12%",
    "--primary": "155 25% 30%",
    "--primary-foreground": "150 10% 98%",
    "--secondary": "150 8% 95%",
    "--secondary-foreground": "150 10% 25%",
    "--muted": "150 6% 95%",
    "--muted-foreground": "150 5% 46%",
    "--accent": "155 10% 93%",
    "--accent-foreground": "150 8% 12%",
    "--destructive": "0 55% 48%",
    "--destructive-foreground": "150 10% 98%",
    "--border": "150 5% 89%",
    "--input": "150 5% 87%",
    "--ring": "155 25% 30%",
    "--card": "150 10% 99%",
    "--card-foreground": "150 8% 12%",
    "--popover": "150 8% 100%",
    "--popover-foreground": "150 8% 12%",
    "--sidebar-background": "150 8% 97%",
    "--sidebar-foreground": "150 8% 12%",
    "--sidebar-accent": "152 8% 92%",
    "--sidebar-accent-foreground": "150 8% 12%",
    "--sidebar-border": "150 5% 90%",
  },
}

/** 靛青 — 低饱和蓝灰，经典工具类应用配色。 */
const indigo: ThemePreset = {
  key: "indigo",
  label: "靛青",
  description: "灰蓝色，经典工具风",
  colors: {
    "--background": "220 14% 99%",
    "--foreground": "224 12% 12%",
    "--primary": "224 30% 38%",
    "--primary-foreground": "220 14% 98%",
    "--secondary": "220 10% 96%",
    "--secondary-foreground": "224 14% 25%",
    "--muted": "220 8% 96%",
    "--muted-foreground": "220 8% 46%",
    "--accent": "220 10% 94%",
    "--accent-foreground": "224 12% 12%",
    "--destructive": "0 60% 50%",
    "--destructive-foreground": "220 14% 98%",
    "--border": "220 8% 90%",
    "--input": "220 8% 88%",
    "--ring": "224 30% 38%",
    "--card": "220 14% 99%",
    "--card-foreground": "224 12% 12%",
    "--popover": "220 12% 100%",
    "--popover-foreground": "224 12% 12%",
    "--sidebar-background": "220 10% 97%",
    "--sidebar-foreground": "224 12% 12%",
    "--sidebar-accent": "220 9% 93%",
    "--sidebar-accent-foreground": "224 12% 12%",
    "--sidebar-border": "220 8% 91%",
  },
}

// ─── 注册表 ────────────────────────────────────────────────────

/** 所有可用主题，按展示顺序排列 */
export const themePresets: ThemePreset[] = [ink, stone, sage, indigo]

/** 默认主题 key */
export const DEFAULT_THEME_KEY = "ink"

/** 按 key 快速查找主题 */
export function getThemeByKey(key: string): ThemePreset | undefined {
  return themePresets.find((t) => t.key === key)
}
