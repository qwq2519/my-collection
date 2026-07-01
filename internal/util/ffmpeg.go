package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"collections/internal/model"
)

func ffmpegBinName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

// DetectFFmpeg checks for ffmpeg availability.
// Search order: persist/bin/ (project-local) → system PATH.
// When unavailable, returns platform-specific install guide
// pointing to scripts/install-ffmpeg.{sh,bat}.
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

// probeFFmpeg runs "ffmpeg -version" and extracts path + version.
func probeFFmpeg(binPath string) (absPath, version string, ok bool) {
	abs, err := filepath.Abs(binPath)
	if err != nil {
		return "", "", false
	}
	out, err := exec.Command(abs, "-version").Output()
	if err != nil {
		return "", "", false
	}
	return abs, parseFFmpegVersion(string(out)), true
}

// parseFFmpegVersion extracts version from "ffmpeg version X.Y.Z ..." first line.
func parseFFmpegVersion(output string) string {
	line, _, _ := strings.Cut(output, "\n")
	line = strings.TrimPrefix(line, "ffmpeg version ")
	if sp := strings.IndexByte(line, ' '); sp > 0 {
		return line[:sp]
	}
	return strings.TrimSpace(line)
}

// buildInstallGuide returns install commands.
// "方式二" uses the project scripts (scripts/install-ffmpeg.sh or .bat),
// resolving the script path relative to persistDir's parent (project root).
func buildInstallGuide(persistDir string) []model.InstallCommand {
	scriptPath := resolveScriptPath(persistDir)

	switch runtime.GOOS {
	case "darwin":
		return []model.InstallCommand{
			{Label: "方式一：Homebrew（推荐）", Command: "brew install ffmpeg"},
			{Label: "方式二：运行安装脚本", Command: scriptPath},
		}
	case "windows":
		return []model.InstallCommand{
			{Label: "方式一：winget（推荐）", Command: "winget install Gyan.FFmpeg"},
			{Label: "方式二：运行安装脚本", Command: scriptPath},
		}
	default:
		return []model.InstallCommand{
			{Label: "方式一：包管理器（推荐）", Command: "sudo apt install ffmpeg"},
			{Label: "方式二：运行安装脚本", Command: scriptPath},
		}
	}
}

// resolveScriptPath finds the install script's absolute path.
// Assumes project layout: {project_root}/persist/ and {project_root}/scripts/.
func resolveScriptPath(persistDir string) string {
	abs, _ := filepath.Abs(persistDir)
	projectRoot := filepath.Dir(abs)

	var script string
	if runtime.GOOS == "windows" {
		script = filepath.Join(projectRoot, "scripts", "install-ffmpeg.bat")
	} else {
		script = filepath.Join(projectRoot, "scripts", "install-ffmpeg.sh")
	}

	if _, err := os.Stat(script); err == nil {
		return script
	}
	return script
}
