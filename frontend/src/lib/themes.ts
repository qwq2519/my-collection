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

/** 石板蓝 — 冷调蓝灰，高对比度，清晰的视觉结构。 */
const slate: ThemePreset = {
  key: "slate",
  label: "石板蓝",
  description: "冷调蓝灰，清晰醒目",
  colors: {
    "--background": "0 0% 100%",
    "--foreground": "224 71% 4%",
    "--primary": "221 83% 53%",
    "--primary-foreground": "210 20% 98%",
    "--secondary": "220 15% 95%",
    "--secondary-foreground": "220 20% 18%",
    "--muted": "220 15% 95%",
    "--muted-foreground": "220 9% 46%",
    "--accent": "220 15% 93%",
    "--accent-foreground": "224 71% 4%",
    "--destructive": "0 72% 51%",
    "--destructive-foreground": "0 0% 98%",
    "--border": "220 13% 87%",
    "--input": "220 13% 85%",
    "--ring": "221 83% 53%",
    "--card": "0 0% 100%",
    "--card-foreground": "224 71% 4%",
    "--popover": "0 0% 100%",
    "--popover-foreground": "224 71% 4%",
    "--sidebar-background": "220 20% 94%",
    "--sidebar-foreground": "224 71% 4%",
    "--sidebar-accent": "220 16% 90%",
    "--sidebar-accent-foreground": "224 71% 4%",
    "--sidebar-border": "220 13% 87%",
  },
}

// ─── 注册表 ────────────────────────────────────────────────────

/** 所有可用主题，按展示顺序排列 */
export const themePresets: ThemePreset[] = [ink, slate]

/** 默认主题 key */
export const DEFAULT_THEME_KEY = "ink"

/** 按 key 快速查找主题 */
export function getThemeByKey(key: string): ThemePreset | undefined {
  return themePresets.find((t) => t.key === key)
}
