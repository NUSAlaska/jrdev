package jrdev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"
)

const agentQueueRunBranchPrefix = "refs/heads/agent-queue/run-"

func pathUnderRoot(child, root string) bool {
	c, err := filepath.Abs(filepath.Clean(child))
	if err != nil {
		return false
	}
	r, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(r, c)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func verifyGitWorktreeDir(abs string) error {
	c := exec.Command("git", "-C", abs, "rev-parse", "--is-inside-work-tree")
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git -C %q rev-parse: %w\n%s", abs, err, out)
	}
	if strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("path %q is not a git worktree (got %q)", abs, strings.TrimSpace(string(out)))
	}
	return nil
}

// FindResumableIntegrationWorktree returns the newest agent-queue/run-* branch that has a linked,
// valid worktree under worktreesRootAbs.
func (g GitOps) FindResumableIntegrationWorktree(worktreesRootAbs string) (branch string, workAbs string, ok bool, err error) {
	entries, err := g.ListWorktreeEntries()
	if err != nil {
		return "", "", false, err
	}
	type cand struct {
		branchRef, path string
	}
	var cands []cand
	for _, e := range entries {
		if e.BranchRef == "" || !strings.HasPrefix(e.BranchRef, agentQueueRunBranchPrefix) {
			continue
		}
		if !pathUnderRoot(e.Path, worktreesRootAbs) {
			continue
		}
		if err := verifyGitWorktreeDir(e.Path); err != nil {
			if g.Log != nil {
				g.Log("jrdev: verbose: skip stale integration worktree %q: %v\n", e.Path, err)
			}
			continue
		}
		cands = append(cands, cand{branchRef: e.BranchRef, path: e.Path})
	}
	if len(cands) == 0 {
		return "", "", false, nil
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].branchRef > cands[j].branchRef
	})
	ref := cands[0].branchRef
	short := strings.TrimPrefix(ref, "refs/heads/")
	return short, cands[0].path, true, nil
}

// CleanupJrdevWorkstate removes every linked worktree under worktreesRootAbs and deletes local
// branches matching agent-queue/run-* and agent-queue/issue-*.
func (g GitOps) CleanupJrdevWorkstate(worktreesRootAbs string) error {
	entries, err := g.ListWorktreeEntries()
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, e := range entries {
		if !pathUnderRoot(e.Path, worktreesRootAbs) {
			continue
		}
		if _, ok := seen[e.Path]; ok {
			continue
		}
		seen[e.Path] = struct{}{}
		if e.IsMainWork {
			continue
		}
		if err := g.git("worktree", "remove", "--force", e.Path); err != nil {
			return fmt.Errorf("remove worktree %q: %w", e.Path, err)
		}
	}
	if err := g.git("worktree", "prune"); err != nil {
		return err
	}
	out, err := g.gitOutput("branch", "--list", "agent-queue/*")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "*") {
			name = strings.TrimSpace(strings.TrimPrefix(name, "*"))
		}
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "agent-queue/run-") || strings.HasPrefix(name, "agent-queue/issue-") {
			if err := g.git("branch", "-D", name); err != nil {
				return fmt.Errorf("delete branch %q: %w", name, err)
			}
		}
	}
	entries2, err := os.ReadDir(worktreesRootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, de := range entries2 {
		if !de.IsDir() {
			continue
		}
		p := filepath.Join(worktreesRootAbs, de.Name())
		if _, err := os.Stat(filepath.Join(p, ".git")); err != nil {
			_ = os.RemoveAll(p)
		}
	}
	return nil
}

// PromptResumeOrCleanIntegration asks whether to keep an existing integration worktree or clean and start over.
// Returns true to resume, false to clean up and create a new run.
func PromptResumeOrCleanIntegration(stdin io.Reader, stdout io.Writer, branch, worktreePath string) (resume bool, err error) {
	fmt.Fprintf(stdout, "\njrdev: found an existing integration branch from a prior run:\n  branch:   %s\n  worktree: %s\n\n", branch, worktreePath)
	fmt.Fprintf(stdout, "  [C]ontinue — use this branch and worktree (pick up where you left off)\n")
	fmt.Fprintf(stdout, "  [F]resh  — remove jrdev worktrees under --worktrees and agent-queue/* branches, then start a new run\n\n")
	fmt.Fprintf(stdout, "Enter C or F (default C): ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	s := strings.ToLower(strings.TrimSpace(line))
	switch s {
	case "", "c", "continue":
		return true, nil
	case "f", "fresh", "n", "no":
		return false, nil
	default:
		return true, nil
	}
}

// PromptCleanupJrdevWorkstate asks whether to remove jrdev-linked worktrees and local agent-queue/* branches.
// prCreated should be true when a PR was opened; false when --skip-pr was used or no PR step ran.
// Returns true if the user confirms cleanup. Empty input and anything other than y/yes means decline.
func PromptCleanupJrdevWorkstate(stdin io.Reader, stdout io.Writer, prCreated bool) (cleanup bool, err error) {
	if prCreated {
		fmt.Fprintf(stdout, "\njrdev: pull request created.\n\n")
	} else {
		fmt.Fprintf(stdout, "\njrdev: run finished (--skip-pr; no PR created).\n\n")
	}
	fmt.Fprintf(stdout, "Remove all jrdev worktrees under --worktrees and delete local agent-queue/* branches?\n")
	fmt.Fprintf(stdout, "  [Y]es — delete worktrees and branches\n")
	fmt.Fprintf(stdout, "  [N]o  — leave them so you can inspect prompts, outputs, and diffs (default)\n\n")
	fmt.Fprintf(stdout, "Enter Y or N (default N): ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	s := strings.ToLower(strings.TrimSpace(line))
	switch s {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// StdinIsInteractive reports whether jrdev can prompt on stdin (TTY).
func StdinIsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
