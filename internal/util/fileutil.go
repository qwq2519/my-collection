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

	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err = tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp to target: %w", err)
	}

	return nil
}

// SafePath 校验 target 解析后的绝对路径是否仍在 baseDir 内，防止目录遍历攻击。
// 返回清理后的绝对路径。
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
		return "", fmt.Errorf("路径越界: %s", target)
	}

	return absTarget, nil
}
