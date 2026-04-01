package jrdev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "a", "b")
	root, err := FindRepoRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(root) != filepath.Clean(tmp) {
		t.Fatalf("got %q want %q", root, tmp)
	}
}

func TestFindRepoRoot_Missing(t *testing.T) {
	tmp := t.TempDir()
	_, err := FindRepoRoot(tmp)
	if err == nil {
		t.Fatal("expected error")
	}
}
