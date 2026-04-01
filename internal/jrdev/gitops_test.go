package jrdev

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBranchMergedIntoHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	tmp := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = tmp
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-b", "main")
	_ = exec.Command("git", "-C", tmp, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", tmp, "config", "user.name", "t").Run()
	if err := os.WriteFile(filepath.Join(tmp, "f"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "f")
	runGit("commit", "-m", "m1")
	runGit("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(tmp, "f"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("commit", "-am", "m2")
	runGit("checkout", "main")
	runGit("merge", "feature", "--no-edit")

	ok, err := BranchMergedIntoHead(tmp, "feature")
	if err != nil || !ok {
		t.Fatalf("merged=%v err=%v", ok, err)
	}
}
