package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/NUSAlaska/jrdev/internal/jrdev"
)

//go:embed prompt_plan.md prompt_implement.md prompt_review.md prompt_merge.md
var promptFS embed.FS

func main() {
	os.Exit(run())
}

func run() int {
	repo := flag.String("repo", "", "git repository root (default: walk up from cwd)")
	worktrees := flag.String("worktrees", ".worktrees", "directory under repo for git worktrees (gitignored)")
	label := flag.String("label", "agent-queue", "GitHub label for queued issues")
	dryRun := flag.Bool("dry-run", false, "preflight without agent smoke; stop before orchestration")
	skipPR := flag.Bool("skip-pr", false, "do not run gh pr create at end")
	maxIter := flag.Int("max-iterations", 0, "outer loop cap (default 2N+3 from queue count at start)")
	verbose := flag.Bool("v", false, "verbose logging")
	integrationBase := flag.String("integration-base", "origin/main", "rev to branch integration run from")
	agentBin := flag.String("agent", "", "Cursor agent binary (default: agent on PATH)")
	ghBin := flag.String("gh", "gh", "GitHub CLI binary")
	flag.Parse()

	startDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
		return 1
	}
	repoRoot := *repo
	if repoRoot == "" {
		repoRoot, err = jrdev.FindRepoRoot(startDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
	} else {
		repoRoot, err = filepath.Abs(repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
			return 1
		}
	}

	prompts, err := loadPrompts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "jrdev: prompts: %v\n", err)
		return 1
	}

	cfg := jrdev.Config{
		RepoRoot:    repoRoot,
		Worktrees:   *worktrees,
		Label:       *label,
		DryRun:      *dryRun,
		SkipPR:      *skipPR,
		MaxIters:    *maxIter,
		Verbose:     *verbose,
		AgentBin:    *agentBin,
		GhBin:       *ghBin,
		Integration: *integrationBase,
	}

	log := func(format string, args ...any) {
		fmt.Printf(format, args...)
	}
	var a jrdev.AgentRunner = jrdev.OSAgentRunner{Log: nil}
	if *verbose {
		a = jrdev.OSAgentRunner{Log: log}
	}

	if err := jrdev.Run(cfg, prompts, a, log); err != nil {
		fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
		return 1
	}
	return 0
}

func loadPrompts() (jrdev.PromptBundle, error) {
	read := func(name string) (string, error) {
		b, err := fs.ReadFile(promptFS, name)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	plan, err := read("prompt_plan.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	impl, err := read("prompt_implement.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	rev, err := read("prompt_review.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	mer, err := read("prompt_merge.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	return jrdev.PromptBundle{Plan: plan, Implement: impl, Review: rev, Merge: mer}, nil
}
