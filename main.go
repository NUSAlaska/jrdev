package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/NUSAlaska/jrdev/internal/jrdev"
)

//go:embed prompt_plan.md prompt_implement.md prompt_review.md prompt_merge.md
var promptFS embed.FS

func main() {
	os.Exit(run())
}

func run() int {
	name := programName()
	if len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-help" || os.Args[1] == "--help") {
		usage(name, os.Stdout)
		return 0
	}

	flag.CommandLine.SetOutput(os.Stderr)
	flag.Usage = func() { usage(name, os.Stderr) }

	repo := flag.String("repo", "", "git repository root (default: walk up from cwd)")
	worktrees := flag.String("worktrees", ".worktrees", "directory under repo for git worktrees (gitignored)")
	label := flag.String("label", "agent-queue", "GitHub label for queued issues")
	dryRun := flag.Bool("dry-run", false, "preflight without agent smoke; stop before orchestration")
	skipPR := flag.Bool("skip-pr", false, "do not run gh pr create at end")
	maxIter := flag.Int("max-iterations", 0, "outer loop cap (default 2N+3 from queue count at start)")
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging (same as -v)")
	integrationBase := flag.String("integration-base", "origin/main", "rev to branch integration run from")
	agentBin := flag.String("agent", "", "Cursor agent binary (default: agent on PATH)")
	agentModel := flag.String("agent-model", jrdev.DefaultAgentModel, "Cursor agent --model name")
	ghBin := flag.String("gh", "gh", "GitHub CLI binary")
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "show usage and exit")
	flag.BoolVar(&showHelp, "h", false, "show usage and exit (shorthand)")

	flag.Parse()
	if showHelp {
		usage(name, os.Stdout)
		return 0
	}

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
		Verbose:     verbose,
		AgentBin:    *agentBin,
		AgentModel:  *agentModel,
		GhBin:       *ghBin,
		Integration: *integrationBase,
	}

	log := func(format string, args ...any) {
		fmt.Printf(format, args...)
	}
	var a jrdev.AgentRunner = jrdev.OSAgentRunner{Log: nil}
	if verbose {
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

func programName() string {
	return filepath.Base(os.Args[0])
}

func usage(name string, w io.Writer) {
	var b strings.Builder
	fmt.Fprintf(&b, "jrdev — plan → implement → review → merge loop for GitHub issues with a queue label.\n\n")
	fmt.Fprintf(&b, "Syntax:\n")
	fmt.Fprintf(&b, "  %s [flags]\n", name)
	fmt.Fprintf(&b, "  %s help\n", name)
	fmt.Fprintf(&b, "  %s -h | -help | --help\n\n", name)
	fmt.Fprintf(&b, "Typical calls:\n")
	fmt.Fprintf(&b, "  %s                         run from inside a git repo (repo root is found by walking up for .git)\n", name)
	fmt.Fprintf(&b, "  %s --dry-run               preflight only; no agent smoke, stops before worktrees\n", name)
	fmt.Fprintf(&b, "  %s --repo <path>           set the git repository root explicitly\n", name)
	fmt.Fprintf(&b, "  %s -v --max-iterations 20  verbose logs; cap the outer iteration loop\n\n", name)
	fmt.Fprintf(&b, "Flags:\n")
	_, _ = io.WriteString(w, b.String())

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	rows := [][2]string{
		{"--repo path", "Git repository root (default: walk up from current directory for .git)"},
		{"--worktrees name", "Directory under repo for git worktrees, default .worktrees"},
		{"--label name", "GitHub label for queued issues, default agent-queue"},
		{"--dry-run", "Preflight without agent smoke; exit before orchestration"},
		{"--skip-pr", "Do not run gh pr create when the loop finishes"},
		{"--max-iterations n", "Outer loop cap; 0 means 2N+3 where N is open labeled issues at start"},
		{"-v, --verbose", "Verbose logging (preflight steps, loop phases, agent argv, git/gh subprocess output)"},
		{"--integration-base rev", "Revision to branch integration runs from, default origin/main"},
		{"--agent path", "Cursor agent binary; default is agent on PATH"},
		{"--agent-model name", "Cursor agent --model; default " + jrdev.DefaultAgentModel},
		{"--gh path", "GitHub CLI binary, default gh"},
		{"-h, -help", "Show this help and exit"},
	}
	for _, row := range rows {
		fmt.Fprintf(tw, "  %s\t%s\n", row[0], row[1])
	}
	_ = tw.Flush()
}
