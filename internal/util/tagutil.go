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
		return fmt.Errorf("标签名不能为空")
	}

	if strings.Contains(name, "::") {
		for _, seg := range strings.Split(name, "::") {
			if strings.TrimSpace(seg) == "" {
				return fmt.Errorf("标签层级之间不能为空")
			}
		}
	}

	name = strings.ReplaceAll(name, "::", "")
	if strings.Contains(name, ":") {
		return fmt.Errorf("标签名中不允许使用单独的 :")
	}

	for _, r := range name {
		if unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == ' ' || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("标签名包含不允许的字符: %c", r)
	}

	return nil
}

// NormalizeTagName 归一化标签名：trim 首尾空白、转小写、连续空格归一为单空格
func NormalizeTagName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.Join(strings.Fields(name), " ")
}
