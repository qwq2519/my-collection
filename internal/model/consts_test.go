package model

import "testing"

func TestLookupMediaType(t *testing.T) {
	tests := []struct {
		ext      string
		wantType string
		wantOK   bool
	}{
		{".jpg", MediaTypeImage, true},
		{".jpeg", MediaTypeImage, true},
		{".png", MediaTypeImage, true},
		{".gif", MediaTypeImage, true},
		{".webp", MediaTypeImage, true},
		{".avif", MediaTypeImage, true},
		{".svg", MediaTypeImage, true},
		{".bmp", MediaTypeImage, true},
		{".mp4", MediaTypeVideo, true},
		{".mkv", MediaTypeVideo, true},
		{".avi", MediaTypeVideo, true},
		{".mov", MediaTypeVideo, true},
		{".webm", MediaTypeVideo, true},
		{".mp3", MediaTypeAudio, true},
		{".flac", MediaTypeAudio, true},
		{".wav", MediaTypeAudio, true},
		{".ogg", MediaTypeAudio, true},
		{".m4a", MediaTypeAudio, true},
		{".txt", "", false},
		{".pdf", "", false},
		{".doc", "", false},
		{"", "", false},
		{".JPG", "", false},
	}

	for _, tt := range tests {
		gotType, gotOK := LookupMediaType(tt.ext)
		if gotType != tt.wantType || gotOK != tt.wantOK {
			t.Errorf("LookupMediaType(%q) = (%q, %v), want (%q, %v)",
				tt.ext, gotType, gotOK, tt.wantType, tt.wantOK)
		}
	}
}

func TestIsSupportedMediaExt(t *testing.T) {
	supported := []string{".jpg", ".png", ".mp4", ".mp3", ".webp", ".flac"}
	for _, ext := range supported {
		if !IsSupportedMediaExt(ext) {
			t.Errorf("IsSupportedMediaExt(%q) = false, want true", ext)
		}
	}

	unsupported := []string{".txt", ".pdf", ".exe", "", ".JPG"}
	for _, ext := range unsupported {
		if IsSupportedMediaExt(ext) {
			t.Errorf("IsSupportedMediaExt(%q) = true, want false", ext)
		}
	}
}
