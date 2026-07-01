package util

// StringIndex 返回 s 在 slice 中的索引，不存在返回 -1
func StringIndex(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

// StringRemove 返回移除所有 s 后的新切片
func StringRemove(slice []string, s string) []string {
	out := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

