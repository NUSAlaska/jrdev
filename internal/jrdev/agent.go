package jrdev

import (
	"bytes"
	"fmt"
	"os/exec"
)

// AgentRunOptions configures a single agent invocation.
type AgentRunOptions struct {
	Print bool // pass -p / --print for non-interactive
}

// AgentRunner runs the Cursor CLI agent subprocess.
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
	args := []string{}
	if opts.Print {
		args = append(args, "-p", prompt, "--output-format", "text")
	} else {
		args = append(args, prompt)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return buf.String(), fmt.Errorf("agent %v (cwd=%s): %w\n%s", args, dir, err, buf.String())
	}
	out := buf.String()
	if r.Log != nil {
		r.Log("agent output:\n%s", out)
	}
	return out, nil
}
