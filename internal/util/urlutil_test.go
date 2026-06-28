package util

import "testing"

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"https://github.com/golang/go", false},
		{"http://example.com/page", false},
		{"ftp://files.example.com", true},
		{"https://192.168.1.1/page", true},
		{"https://localhost/page", true},
		{"https://example.com:8080/page", true},
		{"https://user:pass@example.com", true},
		{"not-a-url", true},
	}

	for _, tt := range tests {
		err := ValidateURL(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateURL(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/golang/go", "https://github.com/golang/go"},
		{"http://www.example.com/page/", "http://example.com/page"},
		{"https://GitHub.COM/Go", "https://github.com/Go"},
		{"https://example.com/page#section", "https://example.com/page"},
		{"https://example.com/page?a=1&b=2", "https://example.com/page?a=1&b=2"},
		{"https://example.com/page?b=2&a=1", "https://example.com/page?a=1&b=2"},
		{"https://www.example.com/", "https://example.com"},
	}

	for _, tt := range tests {
		got, err := NormalizeURL(tt.input)
		if err != nil {
			t.Errorf("NormalizeURL(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/golang/go", "github.com"},
		{"https://www.Example.COM/page", "example.com"},
		{"http://sub.domain.org/path", "sub.domain.org"},
	}

	for _, tt := range tests {
		got, err := ExtractDomain(tt.input)
		if err != nil {
			t.Errorf("ExtractDomain(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExtractDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
