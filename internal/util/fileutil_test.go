package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	t.Run("write and read back", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")
		data := []byte("hello world")

		if err := AtomicWrite(path, data, 0644); err != nil {
			t.Fatalf("AtomicWrite failed: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(got) != string(data) {
			t.Errorf("content = %q, want %q", got, data)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "deep", "test.txt")
		data := []byte("nested")

		if err := AtomicWrite(path, data, 0644); err != nil {
			t.Fatalf("AtomicWrite failed: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(got) != string(data) {
			t.Errorf("content = %q, want %q", got, data)
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.txt")

		AtomicWrite(path, []byte("old"), 0644)
		AtomicWrite(path, []byte("new"), 0644)

		got, _ := os.ReadFile(path)
		if string(got) != "new" {
			t.Errorf("content = %q, want %q", got, "new")
		}
	})
}

func TestSafePath(t *testing.T) {
	base := t.TempDir()

	t.Run("valid subpath", func(t *testing.T) {
		got, err := SafePath(base, "sub/file.txt")
		if err != nil {
			t.Fatalf("SafePath failed: %v", err)
		}
		want := filepath.Join(base, "sub", "file.txt")
		if got != want {
			t.Errorf("SafePath = %q, want %q", got, want)
		}
	})

	t.Run("reject traversal", func(t *testing.T) {
		_, err := SafePath(base, "../../../etc/passwd")
		if err == nil {
			t.Error("SafePath should reject path traversal")
		}
	})

	t.Run("reject absolute escape", func(t *testing.T) {
		_, err := SafePath(base, "sub/../../..")
		if err == nil {
			t.Error("SafePath should reject escaping base directory")
		}
	})

	t.Run("base itself", func(t *testing.T) {
		got, err := SafePath(base, ".")
		if err != nil {
			t.Fatalf("SafePath failed: %v", err)
		}
		if got != base {
			t.Errorf("SafePath(base, \".\") = %q, want %q", got, base)
		}
	})
}
