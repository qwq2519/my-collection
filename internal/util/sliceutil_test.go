package util

import "testing"

func TestStringIndex(t *testing.T) {
	tests := []struct {
		slice []string
		s     string
		want  int
	}{
		{[]string{"a", "b", "c"}, "b", 1},
		{[]string{"a", "b", "c"}, "a", 0},
		{[]string{"a", "b", "c"}, "c", 2},
		{[]string{"a", "b", "c"}, "d", -1},
		{[]string{}, "a", -1},
		{nil, "a", -1},
		{[]string{"a", "a", "b"}, "a", 0},
	}

	for _, tt := range tests {
		got := StringIndex(tt.slice, tt.s)
		if got != tt.want {
			t.Errorf("StringIndex(%v, %q) = %d, want %d", tt.slice, tt.s, got, tt.want)
		}
	}
}

func TestStringRemove(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  []string
	}{
		{"remove existing", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"remove first", []string{"a", "b", "c"}, "a", []string{"b", "c"}},
		{"remove last", []string{"a", "b", "c"}, "c", []string{"a", "b"}},
		{"remove nonexistent", []string{"a", "b"}, "z", []string{"a", "b"}},
		{"remove all duplicates", []string{"a", "b", "a", "c"}, "a", []string{"b", "c"}},
		{"empty slice", []string{}, "a", []string{}},
		{"nil slice", nil, "a", []string{}},
	}

	for _, tt := range tests {
		got := StringRemove(tt.slice, tt.s)
		if len(got) != len(tt.want) {
			t.Errorf("StringRemove[%s] len = %d, want %d", tt.name, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("StringRemove[%s][%d] = %q, want %q", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}
