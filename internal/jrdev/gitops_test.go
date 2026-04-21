package jrdev

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCommitHistoryForPrompt(t *testing.T) {
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
	runGit("commit", "-m", "first subject\n\nbody line")

	s, err := CommitHistoryForPrompt(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "---") {
		t.Fatalf("expected commit separator in output: %q", s)
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}\n[0-9]{4}-[0-9]{2}-[0-9]{2}\n`).MatchString(s) {
		t.Fatalf("expected hash then short date then newline: %q", s)
	}
	if !strings.Contains(s, "first subject") {
		t.Fatalf("expected message body: %q", s)
	}
}

func TestGitDiffForPrompt(t *testing.T) {
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
	runGit("commit", "-m", "on main")
	runGit("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(tmp, "f"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("commit", "-am", "on feature")

	s, err := GitDiffForPrompt(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "diff --git") || !strings.Contains(s, "-a") || !strings.Contains(s, "+b") {
		t.Fatalf("expected unified diff with change a->b: %q", s)
	}

	s2, err := GitDiffForPromptFromBase(tmp, "main")
	if err != nil {
		t.Fatal(err)
	}
	if s2 != s {
		t.Fatalf("GitDiffForPromptFromBase(main) mismatch\n%s\nvs\n%s", s2, s)
	}
}

func TestGitDiffForPromptFromBase_customIntegrationBase(t *testing.T) {
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
	runGit("init", "-b", "develop")
	_ = exec.Command("git", "-C", tmp, "config", "user.email", "t@t").Run()
	_ = exec.Command("git", "-C", tmp, "config", "user.name", "t").Run()
	if err := os.WriteFile(filepath.Join(tmp, "f"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "f")
	runGit("commit", "-m", "on develop")
	runGit("checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(tmp, "f"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("commit", "-am", "on feature")

	s, err := GitDiffForPromptFromBase(tmp, "develop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "diff --git") || !strings.Contains(s, "-a") || !strings.Contains(s, "+b") {
		t.Fatalf("expected unified diff from develop..HEAD: %q", s)
	}
}

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

func TestIssueWorkPathForBranch(t *testing.T) {
	root := filepath.FromSlash("/repo/.worktrees")
	if g, w := IssueWorkPathForBranch(root, "agent-queue/issue-83-short"), filepath.Join(root, "issue-83-short"); g != w {
		t.Fatalf("got %q want %q", g, w)
	}
	if g, w := IssueWorkPathForBranch(root, "agent-queue/nested/name"), filepath.Join(root, "nested_name"); g != w {
		t.Fatalf("got %q want %q", g, w)
	}
	if g, w := IssueWorkPathForBranch(root, ""), filepath.Join(root, "issue"); g != w {
		t.Fatalf("empty branch: got %q want %q", g, w)
	}
}

func TestCreateIssueWorktree_CleansStaleBranch(t *testing.T) {
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
	runGit("commit", "-m", "m0")

	wtRoot := filepath.Join(tmp, ".worktrees")
	if err := os.MkdirAll(wtRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	intPath := filepath.Join(wtRoot, "int")
	runGit("branch", "integration", "main")
	runGit("worktree", "add", intPath, "integration")

	issueBranch := "agent-queue/issue-1-slug"
	issuePath := IssueWorkPathForBranch(wtRoot, issueBranch)
	runGit("worktree", "add", issuePath, "-b", issueBranch, "integration")
	runGit("worktree", "remove", "--force", issuePath)

	g := GitOps{RepoRoot: tmp}
	if g.branchExists(issueBranch) != true {
		t.Fatal("expected stale issue branch after worktree remove")
	}
	if _, err := g.CreateIssueWorktree(issuePath, "integration", issueBranch); err != nil {
		t.Fatal(err)
	}
	if !g.branchExists(issueBranch) {
		t.Fatal("expected issue branch after successful create")
	}
	verify := exec.Command("git", "-C", issuePath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := verify.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != issueBranch {
		t.Fatalf("issue worktree HEAD: %q err=%v", strings.TrimSpace(string(out)), err)
	}
}

func TestGitWorkingTreeClean(t *testing.T) {
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
	runGit("commit", "-m", "first")

	ok, err := GitWorkingTreeClean(tmp)
	if err != nil || !ok {
		t.Fatalf("expected clean after commit, ok=%v err=%v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "dirty"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok2, err := GitWorkingTreeClean(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("expected dirty with untracked file")
	}
	if err := RequireGitWorkingTreeClean(tmp); err == nil {
		t.Fatal("expected RequireGitWorkingTreeClean error")
	}
}

func TestDefaultIntegrationBase_mainOnly(t *testing.T) {
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
	runGit("commit", "-m", "first")

	got, err := DefaultIntegrationBase(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != "main" {
		t.Fatalf("got %q want main", got)
	}
}

func TestDefaultIntegrationBase_prefersDev(t *testing.T) {
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
	runGit("commit", "-m", "first")
	runGit("branch", "dev")

	got, err := DefaultIntegrationBase(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != "dev" {
		t.Fatalf("got %q want dev", got)
	}
}

func TestDefaultIntegrationBase_prefersOriginDev(t *testing.T) {
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
	runGit("commit", "-m", "first")
	runGit("branch", "dev")
	runGit("update-ref", "refs/remotes/origin/dev", "refs/heads/dev")

	got, err := DefaultIntegrationBase(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if got != "origin/dev" {
		t.Fatalf("got %q want origin/dev", got)
	}
}

func TestResolveIntegrationBase_explicitFlag(t *testing.T) {
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
	runGit("commit", "-m", "first")

	got, err := ResolveIntegrationBase(tmp, "custom-base")
	if err != nil {
		t.Fatal(err)
	}
	if got != "custom-base" {
		t.Fatalf("got %q", got)
	}
}
