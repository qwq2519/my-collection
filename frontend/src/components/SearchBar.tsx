import { useState, useEffect, useRef } from "react"
import { Search, X } from "lucide-react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface SearchBarProps {
  value: string
  onChange: (value: string) => void
  /** 清除时的回调（可选），不传则默认调用 onChange("") */
  onClear?: () => void
  placeholder?: string
  /** 防抖延迟（ms），默认 300 */
  debounceMs?: number
  className?: string
}

/**
 * 通用搜索框：带防抖的受控输入框。
 * 输入时立即更新显示值，防抖后触发 onChange 回调。
 * 右侧清除按钮在有输入时显示。
 */
export function SearchBar({
  value,
  onChange,
  onClear,
  placeholder = "搜索...",
  debounceMs = 300,
  className,
}: SearchBarProps) {
  const [localValue, setLocalValue] = useState(value)
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  // 触发：外部 value 变化时同步本地状态（如父组件清除搜索）
  useEffect(() => {
    setLocalValue(value)
  }, [value])

  const handleChange = (v: string) => {
    setLocalValue(v)
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => onChange(v), debounceMs)
  }

  const handleClear = () => {
    setLocalValue("")
    clearTimeout(timerRef.current)
    if (onClear) {
      onClear()
    } else {
      onChange("")
    }
  }

  // 触发：组件卸载时清除防抖定时器
  useEffect(() => () => clearTimeout(timerRef.current), [])

  return (
    <div className={cn("relative", className)}>
      <Search
        size={14}
        className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground"
      />
      <Input
        value={localValue}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder}
        className="pl-8 pr-8 h-8 text-sm"
      />
      {localValue && (
        <button
          onClick={handleClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        >
          <X size={14} />
        </button>
      )}
    </div>
  )
}
