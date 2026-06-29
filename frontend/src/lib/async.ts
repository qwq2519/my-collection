import { useState } from "react"
import { extractError } from "./utils"

/**
 * 异步调用结果元组，类比 Go 的 (T, error)。
 * 成功：[result, null]，失败：[null, errorMessage]
 */
export type Result<T> = [T, null] | [null, string]

/**
 * 包装异步调用，将 try-catch 转换为 Go 风格的 [result, err] 元组。
 *
 * Go 类比：
 *   result, err := callService(fn)
 *   if err != nil { ... }
 */
export async function callService<T>(fn: () => Promise<T>): Promise<Result<T>> {
  try {
    const result = await fn()
    return [result, null]
  } catch (e: unknown) {
    return [null, extractError(e)]
  }
}

/**
 * 管理 loading 状态的 hook，内置 callService 调用。
 *
 * 用法：
 *   const saving = useLoading()
 *   const [result, err] = await saving.run(() => URLService.CreateSite(req))
 *   if (err) { setError(err); return }
 */
export function useLoading() {
  const [loading, setLoading] = useState(false)

  async function run<T>(fn: () => Promise<T>): Promise<Result<T>> {
    setLoading(true)
    const result = await callService(fn)
    setLoading(false)
    return result
  }

  return { loading, run }
}
