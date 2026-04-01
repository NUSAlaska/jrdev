package jrdev

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRepoRoot returns the directory containing .git (directory or file worktree marker).
func FindRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("jrdev: abs path: %w", err)
	}
	for {
		if gitMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("jrdev: no git repository found from %q (use --repo); walk up until a .git directory or file exists", start)
		}
		dir = parent
	}
}

func gitMarker(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	return st.IsDir() || st.Mode().IsRegular()
}
