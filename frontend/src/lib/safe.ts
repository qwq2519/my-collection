/**
 * 条件选择，替代三元运算符。
 * Go 类比：无（Go 用 if-else 赋值）
 */
export function pick<T>(cond: boolean, ifTrue: T, ifFalse: T): T {
  if (cond) return ifTrue
  return ifFalse
}

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
