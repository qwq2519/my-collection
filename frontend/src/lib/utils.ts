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

export const IMAGE_EXTS = new Set(["jpg", "jpeg", "png", "gif", "webp", "bmp", "avif", "svg"])
export const VIDEO_EXTS = new Set(["mp4", "mkv", "avi", "mov", "webm", "wmv", "flv"])

export function isPreviewableExt(ext: string): boolean {
  return IMAGE_EXTS.has(ext) || VIDEO_EXTS.has(ext)
}
