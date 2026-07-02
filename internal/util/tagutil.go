package util

import (
	"fmt"
	"strings"
	"unicode"
)

// ValidateTagName 校验标签名是否符合规则。
//
// 允许：中文、英文字母、数字、:: (层级分隔符)、空格、-、_
// 禁止：空字符串、单独的 :、前导/尾随/连续的 ::
func ValidateTagName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("tag name required")
	}

	if strings.Contains(name, "::") {
		for _, seg := range strings.Split(name, "::") {
			if strings.TrimSpace(seg) == "" {
				return fmt.Errorf("empty segment between ::")
			}
		}
	}

	name = strings.ReplaceAll(name, "::", "")
	if strings.Contains(name, ":") {
		return fmt.Errorf("single colon not allowed in tag name")
	}

	for _, r := range name {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("invalid character in tag name: %c", r)
	}

	return nil
}

// NormalizeTagName 归一化标签名：trim 首尾空白、转小写、连续空格归一为单空格
func NormalizeTagName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.Join(strings.Fields(name), " ")
}

// ComputeTagDeltas 计算新旧标签列表之间的差值。
// 正值表示新增引用，负值表示减少引用，零差值已排除。
func ComputeTagDeltas(newTags, oldTags []string) map[string]int {
	deltas := make(map[string]int)
	for _, t := range newTags {
		deltas[t]++
	}
	for _, t := range oldTags {
		deltas[t]--
	}
	for k, v := range deltas {
		if v == 0 {
			delete(deltas, k)
		}
	}
	return deltas
}
