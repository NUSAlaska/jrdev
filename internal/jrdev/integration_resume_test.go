package jrdev

import (
	"path/filepath"
	"testing"
)

func TestPathUnderRoot(t *testing.T) {
	root := filepath.FromSlash("/repo/.worktrees")
	child := filepath.FromSlash("/repo/.worktrees/run-abc")
	if !pathUnderRoot(child, root) {
		t.Fatalf("expected %q under %q", child, root)
	}
	outside := filepath.FromSlash("/repo/other")
	if pathUnderRoot(outside, root) {
		t.Fatalf("did not expect %q under %q", outside, root)
	}
}

func TestParseWorktreeListPorcelain_twoWorktrees(t *testing.T) {
	const sample = `worktree /main/repo
HEAD abc123
branch refs/heads/main

worktree /main/repo/.worktrees/run-xyz
branch refs/heads/agent-queue/run-20260101-120000-123456789

`
	entries := parseWorktreeListPorcelain(sample)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %#v", len(entries), entries)
	}
	if !entries[0].IsMainWork || entries[0].BranchRef != "refs/heads/main" {
		t.Fatalf("first entry: %#v", entries[0])
	}
	if entries[1].IsMainWork {
		t.Fatalf("second entry should not be main")
	}
	if entries[1].Path != "/main/repo/.worktrees/run-xyz" {
		t.Fatalf("path: %q", entries[1].Path)
	}
	if entries[1].BranchRef != "refs/heads/agent-queue/run-20260101-120000-123456789" {
		t.Fatalf("branch: %q", entries[1].BranchRef)
	}
}
