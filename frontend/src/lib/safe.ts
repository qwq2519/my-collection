/** 空字符串安全：null/undefined → "" */
export function str(v: string | null | undefined): string {
  return v ?? ""
}

/** 空数组安全：null/undefined → [] */
export function arr<T>(v: T[] | null | undefined): T[] {
  return v ?? []
}

/** 空数字安全：null/undefined → 0 */
export function num(v: number | null | undefined): number {
  return v ?? 0
}

/** 空布尔安全：null/undefined → false */
export function bool(v: boolean | null | undefined): boolean {
  return v ?? false
}

/**
 * 统一拆包分页列表响应，消灭 store 中重复的 result?.items ?? [] 模式。
 *
 * Go 类比：把 *ListResult 的指针字段转为值类型
 */
export function unpackList<T>(
  result: { items?: T[] | null; total?: number; has_more?: boolean } | null,
): { items: T[]; total: number; hasMore: boolean } {
  return {
    items: result?.items ?? [],
    total: result?.total ?? 0,
    hasMore: result?.has_more ?? false,
  }
}

/** 提取文件扩展名（小写，不含点），无扩展名时返回空字符串 */
export function getFileExt(filename: string): string {
  return filename.split(".").pop()?.toLowerCase() ?? ""
}

/**
 * 检查字符串是否为合法 URL。
 * 封装 new URL() 的 try-catch，业务代码无需 eslint-disable。
 */
export function isValidURL(value: string): boolean {
  try {
    new URL(value)
    return true
  } catch {
    return false
  }
}
