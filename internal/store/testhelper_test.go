package store

import "testing"

// newTestStore 创建临时 Store 实例（BuntDB + Bleve），测试结束自动清理。
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
