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
func RunPreflight(cfg Config, agent AgentRunner, log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
	}

	vlog(cfg, log, "jrdev: verbose: preflight — checking git on PATH\n")
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("jrdev preflight: git not on PATH: %w", err)
	}
	out, err := exec.Command("git", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("jrdev preflight: git version: %w\n%s", err, out)
	}
	vlog(cfg, log, "jrdev: verbose: preflight — %s\n", strings.TrimSpace(string(out)))

	vlog(cfg, log, "jrdev: verbose: preflight — checking gh (%q)\n", cfg.GhBin)
	if _, err := exec.LookPath(cfg.GhBin); err != nil {
		return fmt.Errorf("jrdev preflight: gh not found (%q): %w", cfg.GhBin, err)
	}
	ghStatus := ghCmd(cfg, "auth", "status")
	out, err = ghStatus.CombinedOutput()
	if err != nil {
		return fmt.Errorf("jrdev preflight: gh auth status (is gh logged in?): %w\n%s", err, out)
	}
	vlog(cfg, log, "jrdev: verbose: preflight — gh auth status ok (output %d bytes)\n", len(out))

	agentPath := cfg.AgentBin
	if agentPath == "" {
		agentPath = "agent"
	}
	vlog(cfg, log, "jrdev: verbose: preflight — resolving agent binary %q\n", agentPath)
	if resolved, err := exec.LookPath(agentPath); err != nil {
		return fmt.Errorf("jrdev preflight: Cursor CLI agent not found (try PATH or --agent): %w", err)
	} else {
		vlog(cfg, log, "jrdev: verbose: preflight — agent resolved to %q\n", resolved)
	}
	// Cheap check without model: many CLIs support -h/--help
	vlog(cfg, log, "jrdev: verbose: preflight — agent smoke check: running %q -h\n", agentPath)
	out, err = exec.Command(agentPath, "-h").CombinedOutput()
	if err != nil && !strings.Contains(strings.ToLower(string(out)), "print") {
		// Some builds exit non-zero for -h; still ensure binary runs
		vlog(cfg, log, "jrdev: verbose: preflight — agent -h inconclusive; trying --help\n")
		out2, err2 := exec.Command(agentPath, "--help").CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("jrdev preflight: agent -h/--help failed: %v / %v\n%s\n%s", err, err2, out, out2)
		}
		out = out2
	}
	vlog(cfg, log, "jrdev: verbose: preflight — agent help output %d bytes\n", len(out))

	if !ShouldRunAgentSmoke(cfg) {
		vlog(cfg, log, "jrdev: verbose: preflight — skipping agent smoke (--dry-run)\n")
		return nil
	}

	vlog(cfg, log, "jrdev: verbose: preflight — agent smoke (expect token %q in output)\n", AgentSmokeExpectedToken)
	smokeOut, smokeErr := agent.Run(cfg, cfg.RepoRoot, AgentSmokePrompt, AgentRunOptions{Print: true})
	if smokeErr != nil {
		return fmt.Errorf("jrdev preflight: agent smoke: %w", smokeErr)
	}
	if !strings.Contains(smokeOut, AgentSmokeExpectedToken) {
		return fmt.Errorf("jrdev preflight: agent smoke output missing %q (got output len %d)", AgentSmokeExpectedToken, len(smokeOut))
	}
	vlog(cfg, log, "jrdev: verbose: preflight — agent smoke ok\n")
	return nil
}
