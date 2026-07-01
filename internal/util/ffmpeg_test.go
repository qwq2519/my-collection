package util

import (
	"testing"
)

func TestParseFFmpegVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "standard version line",
			input:  "ffmpeg version 7.1 Copyright (c) 2000-2024 the FFmpeg developers\nbuilt with ...",
			expect: "7.1",
		},
		{
			name:   "semver",
			input:  "ffmpeg version 6.1.2 Copyright (c) 2000-2024 the FFmpeg developers",
			expect: "6.1.2",
		},
		{
			name:   "git snapshot",
			input:  "ffmpeg version N-12345-gabcdef0 Copyright ...",
			expect: "N-12345-gabcdef0",
		},
		{
			name:   "empty",
			input:  "",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFFmpegVersion(tt.input)
			if got != tt.expect {
				t.Errorf("parseFFmpegVersion() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestDetectFFmpeg_NonexistentDir(t *testing.T) {
	status := DetectFFmpeg("/nonexistent/path/that/does/not/exist")
	// persist/bin/ffmpeg won't exist; system ffmpeg may or may not
	if !status.Available && len(status.Guide) == 0 {
		t.Error("unavailable status should include install guide")
	}
}
