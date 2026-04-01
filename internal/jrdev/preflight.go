package jrdev

import (
	"fmt"
	"os/exec"
	"strings"
)

// AgentSmokeExpectedToken must appear in stdout when preflight runs the minimal agent -p check.
const AgentSmokeExpectedToken = "JRDEV_PING_OK"

// AgentSmokePrompt is hardcoded (not loaded from markdown templates).
const AgentSmokePrompt = "Reply with exactly this single line and nothing else:\n" + AgentSmokeExpectedToken + "\nDo not read or modify any files or run shell commands."

// ShouldRunAgentSmoke is false in dry-run (saves tokens).
func ShouldRunAgentSmoke(cfg Config) bool {
	return !cfg.DryRun
}

// RunPreflight runs once at startup: git, gh auth, agent resolvable, optional agent smoke.
func RunPreflight(cfg Config, agent AgentRunner) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("jrdev preflight: git not on PATH: %w", err)
	}
	out, err := exec.Command("git", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("jrdev preflight: git version: %w\n%s", err, out)
	}

	if _, err := exec.LookPath(cfg.GhBin); err != nil {
		return fmt.Errorf("jrdev preflight: gh not found (%q): %w", cfg.GhBin, err)
	}
	ghStatus := ghCmd(cfg, "auth", "status")
	out, err = ghStatus.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jrdev preflight: gh auth status (is gh logged in?): %w\n%s", err, out)
	}

	agentPath := cfg.AgentBin
	if agentPath == "" {
		agentPath = "agent"
	}
	if _, err := exec.LookPath(agentPath); err != nil {
		return fmt.Errorf("jrdev preflight: Cursor CLI agent not found (try PATH or --agent): %w", err)
	}
	// Cheap check without model: many CLIs support -h/--help
	out, err = exec.Command(agentPath, "-h").CombinedOutput()
	if err != nil && !strings.Contains(strings.ToLower(string(out)), "print") {
		// Some builds exit non-zero for -h; still ensure binary runs
		out2, err2 := exec.Command(agentPath, "--help").CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("jrdev preflight: agent -h/--help failed: %v / %v\n%s\n%s", err, err2, out, out2)
		}
		out = out2
	}

	if !ShouldRunAgentSmoke(cfg) {
		return nil
	}

	smokeOut, smokeErr := agent.Run(cfg, cfg.RepoRoot, AgentSmokePrompt, AgentRunOptions{Print: true})
	if smokeErr != nil {
		return fmt.Errorf("jrdev preflight: agent smoke: %w", smokeErr)
	}
	if !strings.Contains(smokeOut, AgentSmokeExpectedToken) {
		return fmt.Errorf("jrdev preflight: agent smoke output missing %q (got output len %d)", AgentSmokeExpectedToken, len(smokeOut))
	}
	return nil
}
