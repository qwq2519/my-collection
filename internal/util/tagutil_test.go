package util

import "testing"

func TestValidateTagName(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"前端", false},
		{"前端::react", false},
		{"前端::react::hooks", false},
		{"go", false},
		{"2026", false},
		{"my-tag", false},
		{"my_tag", false},
		{"tag with spaces", false},
		{"", true},
		{"   ", true},
		{"::react", true},
		{"react::", true},
		{"a::::b", true},
		{"tag:name", true},
		{"tag@name", true},
		{"tag#name", true},
		{"tag/name", true},
	}

	for _, tt := range tests {
		err := ValidateTagName(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateTagName(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
	}
}

func TestNormalizeTagName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  React  ", "react"},
		{"前端::React", "前端::react"},
		{"a  b  c", "a b c"},
		{"  MY TAG  ", "my tag"},
	}

	for _, tt := range tests {
		got := NormalizeTagName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeTagName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
