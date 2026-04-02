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

//go:embed prompt_plan.md prompt_implement.md prompt_review.md prompt_merge.md prompt_pr.md prompt_pre_pr_review_pass1.md
var promptFS embed.FS

func main() {
	os.Exit(run())
}

const jrdevInitHint = "Run `jrdev init` in an interactive terminal to create or finish repository setup."

func run() int {
	name := programName()
	args := os.Args[1:]
	sub := "run"
	if len(args) > 0 {
		switch args[0] {
		case "init":
			sub = "init"
			args = args[1:]
		case "pre-pr-review":
			sub = "pre-pr-review"
			args = args[1:]
		case "help", "-help", "--help":
			usage(name, os.Stdout)
			return 0
		}
	}
	if len(args) > 0 && (args[0] == "help" || args[0] == "-help" || args[0] == "--help") {
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
	fresh := flag.Bool("fresh", false, "discard prior jrdev worktrees/branches (--worktrees + agent-queue/*); skip resume prompt")
	cleanupOnly := flag.Bool("cleanup", false, "remove jrdev worktrees and agent-queue/* branches from a prior run, then exit (no config, no pipeline)")
	configPath := flag.String("config", "", "repository jrdev YAML (default: <repo>/.jrdev/config.yaml)")
	integrationBlocked := flag.String("integration-blocked", "", "when merge agent prints JRDEV_INTEGRATION_BLOCKED — force abort or merge (waive); overrides meta; empty uses meta.integration_blocked_action or prompt/default")
	agentBin := flag.String("agent", "", "Cursor agent binary (default: agent on PATH)")
	agentModel := flag.String("agent-model", jrdev.DefaultAgentModel, "Cursor agent --model name")
	agentCursorDir := flag.String("agent-cursor-config-dir", "", "set CURSOR_CONFIG_DIR to this path (must contain cli-config.json); see Cursor CLI configuration docs")
	agentPermissions := flag.String("agent-permissions", "", "JSON {\"allow\":[\"git\",\"go\"],\"deny\":[]}; bare entries become Shell(); if unset: use <repo>/.cursor/"+jrdev.ProjectCursorCLIConfigName+" when present, else "+jrdev.DefaultAgentPermissionsName+" next to the jrdev executable")
	ghBin := flag.String("gh", "gh", "GitHub CLI binary")
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "show usage and exit")
	flag.BoolVar(&showHelp, "h", false, "show usage and exit (shorthand)")

	flag.CommandLine.Parse(args)
	if showHelp {
		usage(name, os.Stdout)
		return 0
	}

	if *agentCursorDir != "" && *agentPermissions != "" {
		fmt.Fprintf(os.Stderr, "jrdev: choose at most one of --agent-cursor-config-dir and --agent-permissions\n")
		return 1
	}
	if s := strings.TrimSpace(*integrationBlocked); s != "" {
		switch strings.ToLower(s) {
		case "abort", "merge":
		default:
			fmt.Fprintf(os.Stderr, "jrdev: --integration-blocked must be abort or merge\n")
			return 1
		}
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

	if *cleanupOnly {
		workRootAbs, err := filepath.Abs(filepath.Join(repoRoot, *worktrees))
		if err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: worktrees path: %v\n", err)
			return 1
		}
		git := jrdev.GitOps{RepoRoot: repoRoot}
		if verbose {
			git.Log = func(format string, args ...any) { fmt.Printf(format, args...) }
		}
		if err := git.CleanupJrdevWorkstate(workRootAbs); err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: cleanup: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stdout, "jrdev: cleaned worktrees under %s and local agent-queue/* branches.\n", workRootAbs)
		return 0
	}

	cursorDir := *agentCursorDir
	permPath := *agentPermissions
	if cursorDir == "" && permPath == "" {
		if d, ok := jrdev.ProjectCursorConfigDir(repoRoot); ok {
			cursorDir = d
		} else if exe, xerr := os.Executable(); xerr == nil {
			cand := filepath.Join(filepath.Dir(exe), jrdev.DefaultAgentPermissionsName)
			if st, serr := os.Stat(cand); serr == nil && !st.IsDir() {
				permPath = cand
			}
		}
	}

	cfgYAML := *configPath
	if cfgYAML == "" {
		cfgYAML = filepath.Join(repoRoot, ".jrdev", "config.yaml")
	} else {
		var err error
		cfgYAML, err = filepath.Abs(cfgYAML)
		if err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: config path: %v\n", err)
			return 1
		}
	}

	runSetupWizard := func() error {
		presets, err := jrdev.EmbeddedPresetsFS()
		if err != nil {
			return err
		}
		wio := jrdev.InitWizardIO{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
		return jrdev.RunInitWizard(cfgYAML, presets, wio)
	}

	if sub == "init" {
		if !jrdev.StdinIsTTY(int(os.Stdin.Fd())) {
			fmt.Fprintf(os.Stderr, "jrdev init requires an interactive terminal.\n")
			return 1
		}
		if _, err := os.Stat(cfgYAML); err == nil {
			if pc, lerr := jrdev.LoadProjectConfig(cfgYAML); lerr == nil && pc.ConfigReady {
				fmt.Fprintf(os.Stdout, "jrdev: %s already has config_ready: true; nothing to do.\n", cfgYAML)
				return 0
			}
		}
		if err := runSetupWizard(); err != nil {
			fmt.Fprintf(os.Stderr, "jrdev init: %v\n", err)
			return 1
		}
		return 0
	}

	tty := jrdev.StdinIsTTY(int(os.Stdin.Fd()))
	projectCfg, err := jrdev.LoadProjectConfig(cfgYAML)
	if err != nil {
		if os.IsNotExist(err) {
			if tty {
				if err := runSetupWizard(); err != nil {
					fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
					return 1
				}
				projectCfg, err = jrdev.LoadProjectConfig(cfgYAML)
			} else {
				if err := jrdev.WriteNonInteractiveStubConfig(cfgYAML); err != nil {
					fmt.Fprintf(os.Stderr, "jrdev: write stub config: %v\n", err)
					return 1
				}
				fmt.Fprintf(os.Stderr, "jrdev: repository config not found — wrote stub at %s\n%s\n", cfgYAML, jrdevInitHint)
				return 1
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: config: %v\n", err)
			return 1
		}
	}
	if !projectCfg.ConfigReady {
		if tty {
			if err := runSetupWizard(); err != nil {
				fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
				return 1
			}
			projectCfg, err = jrdev.LoadProjectConfig(cfgYAML)
			if err != nil {
				fmt.Fprintf(os.Stderr, "jrdev: config: %v\n", err)
				return 1
			}
		} else {
			fmt.Fprintf(os.Stderr, "jrdev: config_ready is false in %s — finish setup before running the pipeline.\n%s\n", cfgYAML, jrdevInitHint)
			return 1
		}
	}
	if !projectCfg.ConfigReady {
		fmt.Fprintf(os.Stderr, "jrdev: config_ready is still false in %s after setup.\n%s\n", cfgYAML, jrdevInitHint)
		return 1
	}

	prompts, err := loadPrompts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "jrdev: prompts: %v\n", err)
		return 1
	}

	cfg := jrdev.Config{
		RepoRoot:             repoRoot,
		Worktrees:            *worktrees,
		Label:                *label,
		DryRun:               *dryRun,
		SkipPR:               *skipPR,
		MaxIters:             *maxIter,
		Verbose:              verbose,
		AgentBin:             *agentBin,
		AgentModel:           *agentModel,
		AgentCursorConfigDir: cursorDir,
		AgentPermissionsFile: permPath,
		GhBin:                *ghBin,
		Integration:          *integrationBase,
		FreshStart:           *fresh,
		Project:              projectCfg,
		ProjectPath:          cfgYAML,
		IntegrationBlocked:   strings.TrimSpace(*integrationBlocked),
	}

	log := func(format string, args ...any) {
		fmt.Printf(format, args...)
	}
	var a jrdev.AgentRunner = jrdev.OSAgentRunner{Log: nil}
	if verbose {
		a = jrdev.OSAgentRunner{Log: log}
	}

	if sub == "pre-pr-review" {
		if err := jrdev.RunPrePrReview(cfg, prompts, a, startDir, log); err != nil {
			fmt.Fprintf(os.Stderr, "jrdev: %v\n", err)
			return 1
		}
		return 0
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
	pr, err := read("prompt_pr.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	p1, err := read("prompt_pre_pr_review_pass1.md")
	if err != nil {
		return jrdev.PromptBundle{}, err
	}
	return jrdev.PromptBundle{Plan: plan, Implement: impl, Review: rev, Merge: mer, PR: pr, PrePRReviewPass1: p1}, nil
}

func programName() string {
	return filepath.Base(os.Args[0])
}

func usage(name string, w io.Writer) {
	var b strings.Builder
	fmt.Fprintf(&b, "jrdev — plan → implement → review → merge loop for GitHub issues with a queue label.\n\n")
	fmt.Fprintf(&b, "Syntax:\n")
	fmt.Fprintf(&b, "  %s [flags]\n", name)
	fmt.Fprintf(&b, "  %s init [flags]   interactive setup wizard for .jrdev/config.yaml\n", name)
	fmt.Fprintf(&b, "  %s pre-pr-review [flags]   issue discovery + Pass 1 PRD matrix on integration branch checkout\n", name)
	fmt.Fprintf(&b, "  %s help\n", name)
	fmt.Fprintf(&b, "  %s -h | -help | --help\n\n", name)
	fmt.Fprintf(&b, "Typical calls:\n")
	fmt.Fprintf(&b, "  %s                         run from inside a git repo (repo root is found by walking up for .git)\n", name)
	fmt.Fprintf(&b, "  %s --dry-run               preflight only; no agent smoke, stops before worktrees\n", name)
	fmt.Fprintf(&b, "  %s --repo <path>           set the git repository root explicitly\n", name)
	fmt.Fprintf(&b, "  %s -v --max-iterations 20  verbose logs; cap the outer iteration loop\n\n", name)
	fmt.Fprintf(&b, "After a successful run (pull request created, or --skip-pr), an interactive terminal prompts to remove jrdev worktrees under --worktrees and local agent-queue/* branches, so you can inspect prompts and code first. Non-interactive stdin skips that cleanup.\n\n")
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
		{"--config path", "Repository jrdev YAML; default <repo>/.jrdev/config.yaml"},
		{"--integration-blocked abort|merge", "When merge agent emits JRDEV_INTEGRATION_BLOCKED — force stop or waive (overrides meta); default non-interactive is abort unless meta sets integration_blocked_action"},
		{"--fresh", "Remove jrdev worktrees and agent-queue/* branches; do not resume a prior integration run"},
		{"--cleanup", "Only remove jrdev worktrees (under --worktrees) and local agent-queue/* branches, then exit"},
		{"--agent path", "Cursor agent binary; default is agent on PATH"},
		{"--agent-model name", "Cursor agent --model; default " + jrdev.DefaultAgentModel},
		{"--agent-permissions file", "Cursor permission JSON (jrdev format); if unset, uses <repo>/.cursor/" + jrdev.ProjectCursorCLIConfigName + " when present, else " + jrdev.DefaultAgentPermissionsName + " beside the binary"},
		{"--agent-cursor-config-dir path", "CURSOR_CONFIG_DIR (must contain cli-config.json); mutually exclusive with --agent-permissions; overrides repo .cursor discovery"},
		{"--gh path", "GitHub CLI binary, default gh"},
		{"-h, -help", "Show this help and exit"},
	}
	for _, row := range rows {
		fmt.Fprintf(tw, "  %s\t%s\n", row[0], row[1])
	}
	_ = tw.Flush()
}
