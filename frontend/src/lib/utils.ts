import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

/**
 * 合并 Tailwind CSS 类名，处理冲突（如 "p-2 p-4" → "p-4"）。
 * shadcn/ui 组件内部和业务组件统一使用此函数。
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Extract error message from Wails CallError or plain Error.
 * Wails wraps Go errors as JSON: {"message":"...","cause":"...","kind":"RuntimeError"}
 */
export function extractError(e: unknown, fallback = "operation failed"): string {
  if (!(e instanceof Error)) return fallback
  try {
    const parsed = JSON.parse(e.message)
    if (parsed?.message) return parsed.message
  } catch {
    // not JSON, use as-is
  }
  return e.message || fallback
}

/**
 * 将时间戳格式化为中文相对时间。
 * 规则：<1min "刚刚"，<1h "N分钟前"，<24h "N小时前"，<30d "N天前"，>=30d "YYYY/M/D"
 */
export function formatRelativeTime(input: string | Date | null | undefined): string {
  if (!input) return ""
  const date = input instanceof Date ? input : new Date(input)
  if (isNaN(date.getTime())) return ""

  const now = Date.now()
  const diff = now - date.getTime()
  if (diff < 0) return "刚刚"

  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return "刚刚"

  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}分钟前`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`

  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}天前`

  const y = date.getFullYear()
  const m = date.getMonth() + 1
  const d = date.getDate()
  return `${y}/${m}/${d}`
}

export const IMAGE_EXTS = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "svg"])
export const VIDEO_EXTS = new Set(["mp4", "mkv", "avi", "mov", "webm", "wmv", "flv"])

export function isPreviewableExt(ext: string): boolean {
  return IMAGE_EXTS.has(ext) || VIDEO_EXTS.has(ext)
}

/**
 * 将时间戳格式化为中文日期（YYYY/M/D）。
 * 用于详情页展示创建时间、更新时间等精确日期。
 * 与 formatRelativeTime 互补：后者用于列表的"N分钟前"形式。
 */
export function formatDate(input: string | Date | null | undefined): string {
  if (!input) return ""
  const date = input instanceof Date ? input : new Date(input)
  if (isNaN(date.getTime())) return ""
  return date.toLocaleDateString("zh-CN")
}
