package jrdev

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitOps performs repository mutations for jrdev (git uses RepoRoot as process working directory).
type GitOps struct {
	RepoRoot string
	Log      func(format string, args ...any)
}

func (g GitOps) git(args ...string) error {
	c := exec.Command("git", args...)
	c.Dir = g.RepoRoot
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	if g.Log != nil {
		g.Log("git %v\n%s", args, strings.TrimSpace(string(out)))
	}
	return nil
}

func (g GitOps) gitOutput(args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = g.RepoRoot
	out, err := c.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// gitWithSSHRecover runs git once; on SSH public-key auth failure it runs ssh-add interactively and retries once.
func (g GitOps) gitWithSSHRecover(args ...string) error {
	err := g.git(args...)
	if err == nil {
		return nil
	}
	if !looksLikeSSHAuthFailure(err.Error()) {
		return err
	}
	recLog := g.Log
	if recLog == nil {
		recLog = func(string, ...any) {}
	}
	if rerr := trySSHInteractiveRecovery(recLog); rerr != nil {
		return fmt.Errorf("%w\nssh recover: %v", err, rerr)
	}
	return g.git(args...)
}

// FetchOrigin updates remotes.
func (g GitOps) FetchOrigin() error {
	return g.gitWithSSHRecover("fetch", "origin")
}

// PushUpstream runs git push -u origin branch, with the same SSH recovery as FetchOrigin.
func (g GitOps) PushUpstream(branch string) error {
	return g.gitWithSSHRecover("push", "-u", "origin", branch)
}

// EnsureWorktreesRoot creates dir if missing.
func EnsureWorktreesRoot(abs string) error {
	return os.MkdirAll(abs, 0o755)
}

// CreateIntegrationBranchAndWorktree creates agent-queue/run-<uniq> from baseRev and adds a worktree under worktreesRoot.
func (g GitOps) CreateIntegrationBranchAndWorktree(worktreesRoot, baseRev string) (branch, workAbs string, err error) {
	uniq := time.Now().UTC().Format("20060102-150405") + "-" + fmt.Sprintf("%09d", time.Now().UnixNano()%1_000_000_000)
	branch = "agent-queue/run-" + uniq
	workAbs = IntegrationWorkPath(worktreesRoot, branch)
	if err := os.MkdirAll(filepath.Dir(workAbs), 0o755); err != nil {
		return "", "", err
	}
	if err := g.git("branch", branch, baseRev); err != nil {
		return "", "", err
	}
	if err := g.git("worktree", "add", workAbs, branch); err != nil {
		return "", "", err
	}
	return branch, workAbs, nil
}

// CreateIssueWorktree adds a worktree on a new branch at integrationBranch's tip.
func (g GitOps) CreateIssueWorktree(workAbs, integrationBranch, issueBranch string) (baseSHA string, err error) {
	baseSHA, err = g.gitOutput("rev-parse", integrationBranch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(workAbs), 0o755); err != nil {
		return "", err
	}
	if err := g.git("worktree", "add", workAbs, "-b", issueBranch, integrationBranch); err != nil {
		return "", err
	}
	return baseSHA, nil
}

// RemoveWorktree removes a linked worktree.
func (g GitOps) RemoveWorktree(workAbs string) error {
	return g.git("worktree", "remove", "--force", workAbs)
}

// CommitCountAhead returns commits on current branch at workDir since baseSHA (exclusive base, inclusive HEAD).
func CommitCountAhead(workDir, baseSHA string) (int, error) {
	c := exec.Command("git", "-C", workDir, "rev-list", "--count", baseSHA+"..HEAD")
	out, err := c.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("git rev-list --count: %w\n%s", err, out)
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); err != nil {
		return 0, fmt.Errorf("parse rev-list count: %w", err)
	}
	return n, nil
}

// BranchMergedIntoHead reports whether branch tip is an ancestor of HEAD in workDir (already merged).
func BranchMergedIntoHead(workDir, branch string) (bool, error) {
	c := exec.Command("git", "-C", workDir, "merge-base", "--is-ancestor", branch, "HEAD")
	err := c.Run()
	if err == nil {
		return true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// MergeBranchInDir runs git merge branch --no-edit in workDir.
func MergeBranchInDir(workDir, branch string) error {
	c := exec.Command("git", "-C", workDir, "merge", branch, "--no-edit")
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge %s: %w\n%s", branch, err, out)
	}
	return nil
}

// GoVetTest runs go vet ./... and go test ./... in dir.
func GoVetTest(dir string) error {
	vet := exec.Command("go", "vet", "./...")
	vet.Dir = dir
	out, err := vet.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go vet ./...: %w\n%s", err, out)
	}
	test := exec.Command("go", "test", "./...")
	test.Dir = dir
	out, err = test.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go test ./...: %w\n%s", err, out)
	}
	return nil
}

// IntegrationWorkPath is the filesystem path for the integration worktree directory.
func IntegrationWorkPath(worktreesRoot, integrationBranch string) string {
	safe := strings.ReplaceAll(strings.TrimPrefix(integrationBranch, "agent-queue/"), "/", "_")
	return filepath.Join(worktreesRoot, safe)
}

// IssueWorkPath is the filesystem path for an issue worktree.
func IssueWorkPath(worktreesRoot string, issueNumber int, slug string) string {
	return filepath.Join(worktreesRoot, fmt.Sprintf("issue-%d-%s", issueNumber, slug))
}
