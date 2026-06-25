package util

import (
	"regexp"
	"strings"
)

var (
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+`)                      // 标题标记: # ~ ######
	reCodeBlock  = regexp.MustCompile("(?s)```.*?```")                       // 围栏代码块: ```...```
	reInlineCode = regexp.MustCompile("`[^`]+`")                             // 行内代码: `code`
	reImage      = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)                // 图片: ![alt](url)
	reLink       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)               // 链接: [text](url)，保留 text
	reBoldItalic = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)          // 加粗/斜体: **text** *text*
	reStrike     = regexp.MustCompile(`~~([^~]+)~~`)                         // 删除线: ~~text~~
	reBlockquote = regexp.MustCompile(`(?m)^>\s*`)                           // 引用块行首标记: >
	reHR         = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)                  // 水平分割线: --- / *** / ___
	reListMarker = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+|^[\s]*\d+\.\s+`)  // 列表标记: - / * / + / 1.
	reHTMLTag    = regexp.MustCompile(`<[^>]+>`)                             // HTML 标签: <tag>
	reSpaces     = regexp.MustCompile(`\s+`)                                 // 连续空白
)

// StripMarkdown 去除 Markdown 语法标记，保留纯文本内容，用于 Bleve 索引
func StripMarkdown(md string) string {
	s := md

	s = reCodeBlock.ReplaceAllString(s, " ")
	s = reInlineCode.ReplaceAllString(s, " ")
	s = reImage.ReplaceAllString(s, "")
	s = reLink.ReplaceAllString(s, "$1")
	s = reBoldItalic.ReplaceAllString(s, "$1")
	s = reStrike.ReplaceAllString(s, "$1")
	s = reHeading.ReplaceAllString(s, "")
	s = reBlockquote.ReplaceAllString(s, "")
	s = reHR.ReplaceAllString(s, "")
	s = reListMarker.ReplaceAllString(s, "")
	s = reHTMLTag.ReplaceAllString(s, "")

	s = reSpaces.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}
