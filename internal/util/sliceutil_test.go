package util

import (
	"fmt"
	"testing"
)

func TestMergeUnique(t *testing.T) {
	tests := []struct {
		name      string
		existing  []string
		toAdd     []string
		wantMerge []string
		wantAdded []string
	}{
		{"both empty", nil, nil, []string{}, nil},
		{"add to empty", nil, []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"no new", []string{"a", "b"}, []string{"a"}, []string{"a", "b"}, nil},
		{"partial new", []string{"a"}, []string{"a", "b"}, []string{"a", "b"}, []string{"b"}},
		{"dedup in toAdd", []string{}, []string{"a", "a"}, []string{"a"}, []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, added := MergeUnique(tt.existing, tt.toAdd)
			if fmt.Sprint(merged) != fmt.Sprint(tt.wantMerge) {
				t.Errorf("merged = %v, want %v", merged, tt.wantMerge)
			}
			if fmt.Sprint(added) != fmt.Sprint(tt.wantAdded) {
				t.Errorf("added = %v, want %v", added, tt.wantAdded)
			}
		})
	}
}
