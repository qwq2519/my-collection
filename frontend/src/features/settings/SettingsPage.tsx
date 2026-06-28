/**
 * 设置页面：当前仅包含主题占位提示，后续可扩展更多配置项。
 */
export function SettingsPage() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-2xl mx-auto px-8 py-8">
        <h1 className="text-xl font-semibold mb-8">设置</h1>

        <section>
          <h2 className="text-sm font-medium text-muted-foreground mb-4">主题</h2>
          <p className="text-sm text-muted-foreground">当前使用水墨主题，更多主题配色开发中。</p>
        </section>
      </div>
    </div>
  )
}
