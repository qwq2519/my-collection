import { useState, useRef, KeyboardEvent } from "react"
import { Badge } from "@/components/ui/badge"
import { X } from "lucide-react"

interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
}

/**
 * 标签输入组件：输入文字后回车添加标签，支持删除。
 * 标签用 :: 表示层级（如 "开发::前端"），输入时不做特殊处理，
 * 用户直接输入带 :: 的完整标签名即可。
 *
 * 后续可增加自动补全功能（基于 Command + Popover），
 * 当前先实现基础的手动输入。
 */
export function TagInput({ value, onChange, placeholder = "输入标签后按回车添加" }: TagInputProps) {
  const [input, setInput] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)

  const addTag = (tag: string) => {
    const trimmed = tag.trim()
    if (!trimmed) return
    // 去重
    if (value.includes(trimmed)) {
      setInput("")
      return
    }
    onChange([...value, trimmed])
    setInput("")
  }

  const removeTag = (tag: string) => {
    onChange(value.filter((t) => t !== tag))
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault()
      addTag(input)
    } else if (e.key === "Backspace" && !input && value.length > 0) {
      // 输入框为空时按退格删除最后一个标签
      removeTag(value[value.length - 1])
    }
  }

  return (
    <div
      className="flex flex-wrap items-center gap-1 min-h-[32px] w-full rounded-md border border-input bg-background px-2 py-1 text-sm focus-within:ring-1 focus-within:ring-ring cursor-text"
      onClick={() => inputRef.current?.focus()}
    >
      {value.map((tag) => (
        <Badge key={tag} variant="secondary" className="gap-0.5 pr-1">
          {tag}
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              removeTag(tag)
            }}
            className="ml-0.5 hover:text-foreground"
          >
            <X size={10} />
          </button>
        </Badge>
      ))}
      <input
        ref={inputRef}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => addTag(input)}
        placeholder={value.length === 0 ? placeholder : ""}
        className="flex-1 min-w-[80px] bg-transparent outline-none text-sm placeholder:text-muted-foreground"
      />
    </div>
  )
}
