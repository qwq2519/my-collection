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

func TestComputeTagDeltas(t *testing.T) {
	tests := []struct {
		name    string
		newTags []string
		oldTags []string
		want    map[string]int
	}{
		{"both nil", nil, nil, map[string]int{}},
		{"add only", []string{"a", "b"}, nil, map[string]int{"a": 1, "b": 1}},
		{"remove only", nil, []string{"a", "b"}, map[string]int{"a": -1, "b": -1}},
		{"no change", []string{"a", "b"}, []string{"a", "b"}, map[string]int{}},
		{"mixed", []string{"a", "c"}, []string{"a", "b"}, map[string]int{"c": 1, "b": -1}},
		{"duplicate in new", []string{"a", "a"}, nil, map[string]int{"a": 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTagDeltas(tt.newTags, tt.oldTags)
			if len(got) != len(tt.want) {
				t.Fatalf("ComputeTagDeltas() = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("ComputeTagDeltas()[%q] = %d, want %d", k, got[k], v)
				}
			}
		})
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
