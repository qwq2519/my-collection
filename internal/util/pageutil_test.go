package util

import (
	"testing"
)

func TestNormalizePageParams(t *testing.T) {
	tests := []struct {
		page, pageSize, defaultSize int
		wantPage, wantSize          int
	}{
		{0, 0, 20, 1, 20},
		{-1, -5, 20, 1, 20},
		{3, 10, 20, 3, 10},
		{1, 0, 40, 1, 40},
	}
	for _, tt := range tests {
		p, s := NormalizePageParams(tt.page, tt.pageSize, tt.defaultSize)
		if p != tt.wantPage || s != tt.wantSize {
			t.Errorf("NormalizePageParams(%d,%d,%d) = (%d,%d), want (%d,%d)",
				tt.page, tt.pageSize, tt.defaultSize, p, s, tt.wantPage, tt.wantSize)
		}
	}
}

func TestPaginate(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}
	empty := []string{}

	tests := []struct {
		name        string
		page, size  int
		wantLen     int
		wantTotal   int
		wantHasMore bool
	}{
		{"first page", 1, 2, 2, 5, true},
		{"middle page", 2, 2, 2, 5, true},
		{"last page", 3, 2, 1, 5, false},
		{"beyond last", 4, 2, 0, 5, false},
		{"all at once", 1, 10, 5, 5, false},
		{"empty input", 1, 10, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := items
			if tt.name == "empty input" {
				input = nil
			}
			r := Paginate(input, tt.page, tt.size, empty)
			if len(r.Items) != tt.wantLen {
				t.Errorf("len(Items) = %d, want %d", len(r.Items), tt.wantLen)
			}
			if r.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", r.Total, tt.wantTotal)
			}
			if r.HasMore != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", r.HasMore, tt.wantHasMore)
			}
		})
	}
}
