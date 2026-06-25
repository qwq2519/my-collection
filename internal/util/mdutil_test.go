package util

import "testing"

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "headings",
			input: "# Title\n## Subtitle\ntext",
			want:  "Title Subtitle text",
		},
		{
			name:  "bold and italic",
			input: "this is **bold** and *italic* text",
			want:  "this is bold and italic text",
		},
		{
			name:  "links",
			input: "click [here](https://example.com) to go",
			want:  "click here to go",
		},
		{
			name:  "images",
			input: "before ![alt](img.png) after",
			want:  "before after",
		},
		{
			name:  "code blocks",
			input: "before\n```go\nfunc main() {}\n```\nafter",
			want:  "before after",
		},
		{
			name:  "inline code",
			input: "use `fmt.Println` here",
			want:  "use here",
		},
		{
			name:  "blockquotes",
			input: "> this is a quote\nnormal text",
			want:  "this is a quote normal text",
		},
	}

	for _, tt := range tests {
		got := StripMarkdown(tt.input)
		if got != tt.want {
			t.Errorf("StripMarkdown[%s] = %q, want %q", tt.name, got, tt.want)
		}
	}
}
