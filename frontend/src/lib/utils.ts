import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

/**
 * 合并 Tailwind CSS 类名，处理冲突（如 "p-2 p-4" → "p-4"）。
 * shadcn/ui 组件内部和业务组件统一使用此函数。
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
