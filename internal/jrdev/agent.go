package jrdev

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DefaultAgentModel is used when Config.AgentModel is empty.
const DefaultAgentModel = "composer-2-fast"

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
	promptPath, err := writeAgentPromptFile(dir, prompt)
	if err != nil {
		return "", fmt.Errorf("agent: write prompt file in %s: %w", dir, err)
	}
	defer func() { _ = os.Remove(promptPath) }()
	promptArg := shortPromptForFile(filepath.Base(promptPath))

	// --trust: non-interactive workspace trust for headless runs (e.g. git worktrees).
	args := []string{"--model", model, "--trust"}
	if opts.Print {
		args = append(args, "-p", promptArg, "--output-format", "text")
	} else {
		args = append(args, promptArg)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	if r.Log != nil {
		if opts.Print {
			r.Log("jrdev: verbose: agent cwd=%s\n", dir)
			r.Log("jrdev: verbose: agent argv: %q --model %q --trust -p [file %q, %d-byte prompt] --output-format text\n",
				bin, model, filepath.Base(promptPath), len(prompt))
		} else {
			r.Log("jrdev: verbose: agent cwd=%s\n", dir)
			r.Log("jrdev: verbose: agent argv: %q --model %q --trust [file %q, %d-byte prompt]\n",
				bin, model, filepath.Base(promptPath), len(prompt))
		}
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("agent (cwd=%s, prompt=%d bytes): %w\n%s", dir, len(prompt), err, buf.String())
	}
	out := buf.String()
	if r.Log != nil {
		r.Log("agent output:\n%s", out)
	}
	return out, nil
}

func writeAgentPromptFile(dir, content string) (string, error) {
	f, err := os.CreateTemp(dir, "jrdev-agent-prompt-*.txt")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_, werr := f.WriteString(content)
	cerr := f.Close()
	if werr != nil {
		_ = os.Remove(path)
		return "", werr
	}
	if cerr != nil {
		_ = os.Remove(path)
		return "", cerr
	}
	return path, nil
}

func shortPromptForFile(basename string) string {
	return fmt.Sprintf("The full task specification is in the file %q in the current working directory. Read that file in full, then follow it exactly—including any required output structure or formatting. This message is only a pointer to that file.", basename)
}
