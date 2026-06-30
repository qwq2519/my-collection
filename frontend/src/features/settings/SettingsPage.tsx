import { useAppStore } from "@/stores/app"
import { themePresets } from "@/lib/themes"
import { cn } from "@/lib/utils"
import { Check } from "lucide-react"

/**
 * 设置页面：当前仅包含主题切换，后续可扩展更多配置项（数据目录、ffmpeg 等）。
 *
 * 主题预览卡片中的 const bg/fg/primary/... 是从主题预设中解构 CSS 变量值，
 * 用于在卡片内构建一个迷你版的 UI 布局预览（侧边栏 + 内容区 mockup）。
 * 这些 inline style 只用于预览，不影响实际主题（实际主题由 useTheme hook 注入）。
 */
export function SettingsPage() {
  const themeKey = useAppStore((s) => s.themeKey)
  const setTheme = useAppStore((s) => s.setTheme)

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-2xl mx-auto px-8 py-8">
        <h1 className="text-xl font-semibold mb-8">设置</h1>

        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-4">主题</h2>
          <div className="grid grid-cols-2 gap-3">
            {themePresets.map((preset) => {
              const active = preset.key === themeKey
              // 解构主题色值，用于构建 UI 预览 mockup
              const bg = preset.colors["--background"]
              const fg = preset.colors["--foreground"]
              const primary = preset.colors["--primary"]
              const muted = preset.colors["--muted"]
              const border = preset.colors["--border"]
              const sidebar = preset.colors["--sidebar-background"]

              return (
                <button
                  key={preset.key}
                  onClick={() => setTheme(preset.key)}
                  className={cn(
                    "relative flex flex-col rounded-lg border p-3 text-left transition-colors duration-150",
                    active
                      ? "border-primary bg-accent/30"
                      : "border-border hover:border-muted-foreground/40",
                  )}
                >
                  {active && (
                    <div className="absolute top-2 right-2 flex h-5 w-5 items-center justify-center rounded-full bg-primary">
                      <Check size={12} className="text-primary-foreground" />
                    </div>
                  )}

                  <div
                    className="flex h-16 w-full rounded-md overflow-hidden mb-3 border"
                    style={{ borderColor: `hsl(${border})` }}
                  >
                    <div
                      className="w-1/4 flex flex-col items-center justify-center gap-1 p-1"
                      style={{ backgroundColor: `hsl(${sidebar})` }}
                    >
                      {[1, 2, 3].map((i) => (
                        <div
                          key={i}
                          className="h-1.5 w-4/5 rounded-sm"
                          style={{ backgroundColor: `hsl(${muted})` }}
                        />
                      ))}
                    </div>
                    <div
                      className="flex-1 flex flex-col p-2 gap-1.5"
                      style={{ backgroundColor: `hsl(${bg})` }}
                    >
                      <div
                        className="h-2 w-3/5 rounded-sm"
                        style={{ backgroundColor: `hsl(${fg})` }}
                      />
                      <div
                        className="h-1.5 w-4/5 rounded-sm"
                        style={{ backgroundColor: `hsl(${muted})` }}
                      />
                      <div
                        className="h-4 w-1/3 rounded-sm mt-auto"
                        style={{ backgroundColor: `hsl(${primary})` }}
                      />
                    </div>
                  </div>

                  <span className="text-sm font-medium">{preset.label}</span>
                  <span className="text-xs text-muted-foreground">{preset.description}</span>
                </button>
              )
            })}
          </div>
        </section>
      </div>
    </div>
  )
}
