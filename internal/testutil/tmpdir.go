// File: internal/testutil/tmpdir.go
// Created: 2026-06-20
// Description: Helpers for creating temporary directories in tests.

package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// MkdirTemp is a thin wrapper over t.TempDir() that returns a path joined
// with a subdirectory. It is intentionally one-line so tests can write
// "testutil.MkdirTemp(t, "sub")" and get a clean per-test directory.
func MkdirTemp(t testing.TB, sub string) string {
	t.Helper()
	root := t.TempDir()
	if sub == "" {
		return root
	}
	p := filepath.Join(root, sub)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("testutil.MkdirTemp mkdir %s: %v", p, err)
	}
	return p
}
