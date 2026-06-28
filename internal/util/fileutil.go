package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite 原子写入文件：先写临时文件，再 rename 替换目标文件
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	closed := false

	defer func() {
		if err != nil {
			if !closed {
				tmp.Close()
			}
			os.Remove(tmpPath)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err = tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	closed = true

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp to target: %w", err)
	}

	return nil
}

// SafePath 校验 target 解析后的绝对路径是否仍在 baseDir 内，防止目录遍历攻击。
// 返回清理后的绝对路径。
//
// TODO: 当前使用 filepath.Abs 不解析符号链接，攻击者可在 baseDir 内创建
// symlink 指向外部目录来绕过前缀检查。后续需改用 filepath.EvalSymlinks
// 替代 filepath.Abs（注意 EvalSymlinks 要求路径存在，需额外处理不存在的情况）。
// 当前为个人桌面应用，暂不存在安全风险。
func SafePath(baseDir, target string) (string, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}

	absTarget, err := filepath.Abs(filepath.Join(baseDir, target))
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
		return "", fmt.Errorf("path traversal: %s", target)
	}

	return absTarget, nil
}
