import { useEffect, useRef, useState } from "react"
import { URLService } from "../../../bindings/collections/internal/service"

/**
 * URL 标准化 hook：输入 URL 后防抖调用后端 NormalizeURL，返回标准化结果。
 *
 * 工作原理：
 *   用户输入 URL 时不会每次按键都调用后端，而是等用户停止输入 delayMs 毫秒后
 *   才发起请求（防抖 debounce）。这和 Go 中用 time.AfterFunc + timer.Reset 的模式类似。
 *
 * @param urlValue - 当前输入的 URL 字符串
 * @param delayMs  - 防抖延迟（默认 400ms）
 * @returns 标准化后的 URL，输入不合法时返回空字符串
 */
export function useURLNormalize(urlValue: string, delayMs = 400): string {
  const [normalizedURL, setNormalizedURL] = useState("")
  const timerRef = useRef<ReturnType<typeof setTimeout>>()
  const versionRef = useRef(0)

  useEffect(() => {
    if (!urlValue || urlValue.trim().length < 8) {
      setNormalizedURL("")
      return
    }

    const version = ++versionRef.current
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(async () => {
      try {
        const result = await URLService.NormalizeURL(urlValue.trim())
        if (versionRef.current === version) setNormalizedURL(result ?? "")
      } catch {
        if (versionRef.current === version) setNormalizedURL("")
      }
    }, delayMs)

    return () => clearTimeout(timerRef.current)
  }, [urlValue, delayMs])

  return normalizedURL
}

// ─── 站点查找状态机 ─────────────────────────────────────────

/** 站点查找的四种状态，类似 Go 中的 enum/iota 模式 */
export type SiteLookupState =
  | { status: "idle" }
  | { status: "checking" }
  | { status: "found"; domain: string }
  | { status: "not_found"; domain: string }

/**
 * 书签创建时的站点查找 hook：输入 URL 后防抖查询该域名是否已有对应站点。
 * 同时返回标准化 URL（复用 NormalizeURL 调用，避免重复请求）。
 *
 * 状态流转：idle → checking → found / not_found
 *   - found: 域名已有站点，可直接创建书签
 *   - not_found: 域名无站点，提示用户先创建站点
 *
 * @param urlValue - 当前输入的 URL
 * @param enabled  - 是否启用查找（编辑模式下设为 false 跳过查找）
 */
export function useSiteLookup(
  urlValue: string,
  enabled: boolean,
): { normalizedURL: string; lookupState: SiteLookupState } {
  const [normalizedURL, setNormalizedURL] = useState("")
  const [lookupState, setLookupState] = useState<SiteLookupState>({ status: "idle" })
  const timerRef = useRef<ReturnType<typeof setTimeout>>()
  const versionRef = useRef(0)

  useEffect(() => {
    if (!enabled || !urlValue || urlValue.trim().length < 8) {
      setLookupState({ status: "idle" })
      setNormalizedURL("")
      return
    }

    // 验证是否为合法 URL 格式
    try {
      new URL(urlValue)
    } catch {
      setLookupState({ status: "idle" })
      setNormalizedURL("")
      return
    }

    const version = ++versionRef.current
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(async () => {
      // 步骤 1：标准化 URL
      try {
        const norm = await URLService.NormalizeURL(urlValue.trim())
        if (versionRef.current === version) setNormalizedURL(norm ?? "")
      } catch {
        if (versionRef.current === version) setNormalizedURL("")
      }

      if (versionRef.current !== version) return

      // 步骤 2：查找域名对应的站点
      setLookupState({ status: "checking" })
      try {
        const result = await URLService.LookupSiteByURL({ url: urlValue })
        if (versionRef.current !== version) return
        if (result?.found) {
          setLookupState({ status: "found", domain: result.domain })
        } else {
          setLookupState({ status: "not_found", domain: result?.domain ?? "" })
        }
      } catch {
        if (versionRef.current === version) setLookupState({ status: "idle" })
      }
    }, 500)

    return () => clearTimeout(timerRef.current)
  }, [urlValue, enabled])

  return { normalizedURL, lookupState }
}
