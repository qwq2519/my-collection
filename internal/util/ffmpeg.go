package util

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"collections/internal/model"
)

// ffmpegBinName returns the platform-specific ffmpeg binary name.
func ffmpegBinName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

// DetectFFmpeg checks for ffmpeg availability.
// Search order: persist/bin/ (project-local) → system PATH.
// Returns the resolved status with install guide when unavailable.
func DetectFFmpeg(persistDir string) model.FFmpegStatus {
	localBin := filepath.Join(persistDir, "bin", ffmpegBinName())
	if path, ver, ok := probeFFmpeg(localBin); ok {
		return model.FFmpegStatus{Available: true, BinPath: path, Version: ver}
	}

	if sysPath, err := exec.LookPath("ffmpeg"); err == nil {
		if path, ver, ok := probeFFmpeg(sysPath); ok {
			return model.FFmpegStatus{Available: true, BinPath: path, Version: ver}
		}
	}

	return model.FFmpegStatus{
		Available: false,
		Guide:     buildInstallGuide(persistDir),
	}
}

// probeFFmpeg runs "ffmpeg -version" at the given path and extracts the version string.
func probeFFmpeg(binPath string) (absPath, version string, ok bool) {
	abs, err := filepath.Abs(binPath)
	if err != nil {
		return "", "", false
	}

	out, err := exec.Command(abs, "-version").Output()
	if err != nil {
		return "", "", false
	}

	ver := parseFFmpegVersion(string(out))
	return abs, ver, true
}

// parseFFmpegVersion extracts version from "ffmpeg version X.Y.Z ..." first line.
func parseFFmpegVersion(output string) string {
	line, _, _ := strings.Cut(output, "\n")
	// "ffmpeg version 7.1 Copyright ..."
	line = strings.TrimPrefix(line, "ffmpeg version ")
	if sp := strings.IndexByte(line, ' '); sp > 0 {
		return line[:sp]
	}
	return strings.TrimSpace(line)
}

// buildInstallGuide returns platform-specific install commands.
// The "local install" command downloads ffmpeg into persist/bin/.
func buildInstallGuide(persistDir string) []model.InstallCommand {
	abs, _ := filepath.Abs(persistDir)
	binDir := filepath.Join(abs, "bin")

	switch runtime.GOOS {
	case "darwin":
		return []model.InstallCommand{
			{
				Label:   "方式一：Homebrew 安装（推荐）",
				Command: "brew install ffmpeg",
			},
			{
				Label: "方式二：下载到项目本地",
				Command: fmt.Sprintf(
					`mkdir -p "%s" && curl -L "https://evermeet.cx/ffmpeg/getrelease/zip" -o /tmp/ffmpeg.zip && unzip -o /tmp/ffmpeg.zip -d "%s" && rm /tmp/ffmpeg.zip`,
					binDir, binDir,
				),
			},
		}

	case "windows":
		return []model.InstallCommand{
			{
				Label:   "方式一：winget 安装（推荐）",
				Command: "winget install Gyan.FFmpeg",
			},
			{
				Label: "方式二：下载到项目本地",
				Command: fmt.Sprintf(
					`mkdir "%s" & curl -L "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip" -o %%TEMP%%\ffmpeg.zip & tar -xf %%TEMP%%\ffmpeg.zip --strip-components=2 -C "%s" "*/bin/ffmpeg.exe" & del %%TEMP%%\ffmpeg.zip`,
					binDir, binDir,
				),
			},
		}

	default: // linux
		return []model.InstallCommand{
			{
				Label:   "方式一：包管理器安装（推荐）",
				Command: "sudo apt install ffmpeg   # Debian/Ubuntu\nsudo dnf install ffmpeg   # Fedora",
			},
			{
				Label: "方式二：下载到项目本地",
				Command: fmt.Sprintf(
					`mkdir -p "%s" && curl -L "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz" -o /tmp/ffmpeg.tar.xz && tar -xf /tmp/ffmpeg.tar.xz --strip-components=1 -C "%s" --wildcards "*/ffmpeg" && rm /tmp/ffmpeg.tar.xz`,
					binDir, binDir,
				),
			},
		}
	}
}
