import { Globe, FileText, Image, Tags, Settings, PanelLeftClose, PanelLeft } from "lucide-react"
import { cn } from "@/lib/utils"
import { useAppStore, type Page } from "@/stores/app"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { pick } from "@/lib/safe"

/** 导航项配置：功能导航和底部设置分开定义 */
const navItems: { page: Page; label: string; icon: typeof Globe }[] = [
  { page: "url", label: "URL 收藏", icon: Globe },
  { page: "notes", label: "笔记", icon: FileText },
  { page: "media", label: "媒体", icon: Image },
  { page: "tags", label: "标签管理", icon: Tags },
]

const settingsItem = { page: "settings" as Page, label: "设置", icon: Settings }

export function Sidebar() {
  const currentPage = useAppStore((s) => s.currentPage)
  const collapsed = useAppStore((s) => s.sidebarCollapsed)
  const setPage = useAppStore((s) => s.setPage)
  const toggleSidebar = useAppStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        "flex flex-col h-full bg-sidebar-background border-r border-sidebar-border transition-[width] duration-150",
        pick(collapsed, "w-12", "w-[220px]"),
      )}
    >
      {/* 折叠/展开按钮 */}
      <div
        className={cn("flex items-center p-2", pick(collapsed, "justify-center", "justify-end"))}
      >
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground"
          onClick={toggleSidebar}
        >
          {pick(collapsed, <PanelLeft size={16} />, <PanelLeftClose size={16} />)}
        </Button>
      </div>

      {/* 功能导航 */}
      <nav className="flex-1 flex flex-col gap-1 px-2">
        {navItems.map((item) => (
          <NavItem
            key={item.page}
            label={item.label}
            icon={item.icon}
            active={currentPage === item.page}
            collapsed={collapsed}
            onClick={() => setPage(item.page)}
          />
        ))}
      </nav>

      {/* 底部设置项，用分割线隔开 */}
      <div className="border-t border-sidebar-border px-2 py-2">
        <NavItem
          label={settingsItem.label}
          icon={settingsItem.icon}
          active={currentPage === "settings"}
          collapsed={collapsed}
          onClick={() => setPage("settings")}
        />
      </div>
    </aside>
  )
}

/** 单个导航项：展开时显示图标+文字，折叠时只显示图标+tooltip */
function NavItem({
  label,
  icon: Icon,
  active,
  collapsed,
  onClick,
}: {
  label: string
  icon: typeof Globe
  active: boolean
  collapsed: boolean
  onClick: () => void
}) {
  const content = (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 w-full rounded-md text-sm transition-colors duration-150",
        pick(collapsed, "justify-center px-0 py-2", "px-3 py-2"),
        active
          ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
          : "text-sidebar-foreground hover:bg-sidebar-accent/50",
      )}
    >
      <Icon size={16} />
      {!collapsed && <span>{label}</span>}
    </button>
  )

  // 折叠态用 tooltip 显示文字
  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{content}</TooltipTrigger>
        <TooltipContent side="right" sideOffset={8}>
          {label}
        </TooltipContent>
      </Tooltip>
    )
  }

  return content
}
