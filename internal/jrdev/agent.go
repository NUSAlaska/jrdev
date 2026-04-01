package jrdev

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"
)

// DefaultAgentModel is used when Config.AgentModel is empty.
const DefaultAgentModel = "composer-2-fast"

// AgentArtifactsDir is the per-worktree root for saved prompts and agent output.
const AgentArtifactsDir = ".jrdev"

const agentRunsSubdir = "agent-runs"

const agentNestedGitignore = "# jrdev: local agent prompts and transcripts (do not commit)\n" + agentRunsSubdir + "/\n"

// AgentRunOptions configures a single agent invocation.
type AgentRunOptions struct {
	Print bool // pass -p / --print for non-interactive
}

// AgentRunner runs the Cursor agent subprocess.
type AgentRunner interface {
	Run(cfg Config, dir string, prompt string, opts AgentRunOptions) (stdout string, err error)
}

// OSAgentRunner is the production runner (captures combined stdout+stderr for logs).
type OSAgentRunner struct {
	Log func(format string, args ...any)
}

// Run implements AgentRunner.
func (r OSAgentRunner) Run(cfg Config, dir string, prompt string, opts AgentRunOptions) (string, error) {
	bin := cfg.AgentBin
	if bin == "" {
		bin = "agent"
	}
	model := cfg.AgentModel
	if model == "" {
		model = DefaultAgentModel
	}
	runDir, err := newAgentRunDir(dir)
	if err != nil {
		return "", fmt.Errorf("agent: artifact dir in %s: %w", dir, err)
	}
	promptPath := filepath.Join(runDir, "prompt.txt")
	if err := writeTextFile(promptPath, prompt); err != nil {
		return "", fmt.Errorf("agent: write prompt file: %w", err)
	}
	if !pathUnderRoot(promptPath, dir) {
		return "", fmt.Errorf("agent: prompt path %q not under cwd %q", promptPath, dir)
	}
	relPrompt, err := filepath.Rel(dir, promptPath)
	if err != nil {
		return "", fmt.Errorf("agent: prompt path rel to cwd: %w", err)
	}
	promptArg := shortPromptForRelFile(filepath.ToSlash(relPrompt))

	// --trust: non-interactive workspace trust for headless runs (e.g. git worktrees).
	args := []string{"--model", model, "--trust"}
	if opts.Print {
		args = append(args, "-p", promptArg, "--output-format", "text")
	} else {
		args = append(args, promptArg)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	extraEnv, envCleanup, err := PrepareAgentCursorEnv(cfg)
	if err != nil {
		return "", fmt.Errorf("agent cursor config: %w", err)
	}
	defer envCleanup()
	if len(extraEnv) > 0 {
		cmd.Env = append(slices.Clone(os.Environ()), extraEnv...)
		if r.Log != nil {
			r.Log("jrdev: verbose: agent CURSOR_CONFIG_DIR from jrdev permissions / --agent-cursor-config-dir\n")
		}
	}
	if r.Log != nil {
		r.Log("jrdev: verbose: agent artifacts %q (prompt.txt, output.txt)\n", runDir)
		if opts.Print {
			r.Log("jrdev: verbose: agent cwd=%s\n", dir)
			r.Log("jrdev: verbose: agent argv: %q --model %q --trust -p [file %q, %d-byte prompt] --output-format text\n",
				bin, model, relPrompt, len(prompt))
		} else {
			r.Log("jrdev: verbose: agent cwd=%s\n", dir)
			r.Log("jrdev: verbose: agent argv: %q --model %q --trust [file %q, %d-byte prompt]\n",
				bin, model, relPrompt, len(prompt))
		}
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	runErr := cmd.Run()
	_ = writeTextFile(filepath.Join(runDir, "output.txt"), buf.String())
	if runErr != nil {
		return buf.String(), fmt.Errorf("agent (cwd=%s, prompt=%d bytes): %w\n%s", dir, len(prompt), runErr, buf.String())
	}
	out := buf.String()
	if r.Log != nil {
		r.Log("agent output:\n%s", out)
	}
	return out, nil
}

func ensureAgentArtifactsTree(workDir string) error {
	root := filepath.Join(workDir, AgentArtifactsDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	gi := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gi); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(gi, []byte(agentNestedGitignore), 0o644)
}

func newAgentRunDir(workDir string) (string, error) {
	if err := ensureAgentArtifactsTree(workDir); err != nil {
		return "", err
	}
	id := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	runDir := filepath.Join(workDir, AgentArtifactsDir, agentRunsSubdir, id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", err
	}
	return runDir, nil
}

func writeTextFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func shortPromptForRelFile(relPath string) string {
	return fmt.Sprintf("The full task specification is in the file %q (path relative to the current working directory). Read that file in full, then follow it exactly—including any required output structure or formatting. This message is only a pointer to that file.", relPath)
}
