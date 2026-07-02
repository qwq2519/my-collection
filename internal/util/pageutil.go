package util

// NormalizePageParams 规范化分页参数：page 最小为 1，pageSize 最小为 defaultSize
func NormalizePageParams(page, pageSize, defaultSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSize
	}
	return page, pageSize
}

// PaginateResult 内存分页结果
type PaginateResult[T any] struct {
	Items   []T
	Total   int
	HasMore bool
}

// Paginate 对已排序的切片执行内存分页，返回指定页的数据。
// empty 为空切片实例，确保 JSON 序列化为 [] 而非 null。
func Paginate[T any](items []T, page, pageSize int, empty []T) PaginateResult[T] {
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return PaginateResult[T]{Items: empty, Total: total, HasMore: false}
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return PaginateResult[T]{
		Items:   items[start:end],
		Total:   total,
		HasMore: end < total,
	}
}
