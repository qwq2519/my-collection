package util

// MergeUnique 将 toAdd 追加到 existing 中（去重），返回合并后的切片和实际新增的元素列表
func MergeUnique(existing, toAdd []string) (merged, added []string) {
	set := make(map[string]struct{}, len(existing))
	for _, s := range existing {
		set[s] = struct{}{}
	}

	merged = make([]string, len(existing))
	copy(merged, existing)
	for _, s := range toAdd {
		if _, ok := set[s]; !ok {
			merged = append(merged, s)
			set[s] = struct{}{}
			added = append(added, s)
		}
	}
	return merged, added
}
