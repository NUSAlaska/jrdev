package jrdev

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DefaultAgentPermissionsName is the file name jrdev looks for next to the executable when no flag is set.
const DefaultAgentPermissionsName = "jrdev-agent-permissions.json"

// ProjectCursorCLIConfigName is the Cursor CLI config file jrdev looks for under the repository root.
const ProjectCursorCLIConfigName = "cli-config.json"

// ProjectCursorConfigDir returns <repoRoot>/.cursor when that directory contains ProjectCursorCLIConfigName.
// repoRoot should be absolute (as from FindRepoRoot / flag --repo).
func ProjectCursorConfigDir(repoRoot string) (cursorDir string, ok bool) {
	cli := filepath.Join(repoRoot, ".cursor", ProjectCursorCLIConfigName)
	st, err := os.Stat(cli)
	if err != nil || st.IsDir() {
		return "", false
	}
	return filepath.Join(repoRoot, ".cursor"), true
}

// agentPermissionsJSON is the on-disk shape for --agent-permissions and DefaultAgentPermissionsName.
// Each entry is either a bare shell command base (e.g. "git", "go") or a full Cursor permission token (e.g. "Shell(git)", "Read(**)").
type agentPermissionsJSON struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// cursorCLIConfig is a minimal cli-config.json compatible with Cursor Agent (version 1).
type cursorCLIConfig struct {
	Version int `json:"version"`
	Editor  struct {
		VimMode bool `json:"vimMode"`
	} `json:"editor"`
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	} `json:"permissions"`
}

// PrepareAgentCursorEnv returns extra environment entries for agent subprocesses so Cursor CLI picks up permissions.
// When cfg.AgentPermissionsFile is set, a temporary directory containing cli-config.json is created; cleanup removes it.
// When cfg.AgentCursorConfigDir is set, cleanup is a no-op and env points at that directory (must contain cli-config.json).
func PrepareAgentCursorEnv(cfg Config) (extraEnv []string, cleanup func(), err error) {
	noCleanup := func() {}
	cleanup = noCleanup

	if cfg.AgentCursorConfigDir != "" {
		abs, err := filepath.Abs(cfg.AgentCursorConfigDir)
		if err != nil {
			return nil, noCleanup, err
		}
		cli := filepath.Join(abs, "cli-config.json")
		if _, err := os.Stat(cli); err != nil {
			return nil, noCleanup, fmt.Errorf("agent cursor config: %q: %w", cli, err)
		}
		return []string{"CURSOR_CONFIG_DIR=" + abs}, noCleanup, nil
	}

	if cfg.AgentPermissionsFile == "" {
		return nil, noCleanup, nil
	}

	path, err := filepath.Abs(cfg.AgentPermissionsFile)
	if err != nil {
		return nil, noCleanup, err
	}
	raw, yerr := os.ReadFile(path)
	if yerr != nil {
		return nil, noCleanup, fmt.Errorf("read agent permissions file %q: %w", path, yerr)
	}

	var perm agentPermissionsJSON
	if err := json.Unmarshal(raw, &perm); err != nil {
		return nil, noCleanup, fmt.Errorf("parse agent permissions JSON %q: %w", path, err)
	}

	// Workspace file tools: a minimal cli-config.json must still allow editing the repo; global config is bypassed when CURSOR_CONFIG_DIR is set.
	baseAllow := []string{"Read(**)", "Write(**)"}
	allow, err := normalizePermissionTokens(append(baseAllow, perm.Allow...))
	if err != nil {
		return nil, noCleanup, fmt.Errorf("agent permissions allow: %w", err)
	}
	deny, err := normalizePermissionTokens(perm.Deny)
	if err != nil {
		return nil, noCleanup, fmt.Errorf("agent permissions deny: %w", err)
	}

	var c cursorCLIConfig
	c.Version = 1
	c.Editor.VimMode = false
	c.Permissions.Allow = allow
	c.Permissions.Deny = deny

	dir, err := os.MkdirTemp("", "jrdev-cursor-config-*")
	if err != nil {
		return nil, noCleanup, err
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		cleanup()
		cleanup = noCleanup
		return nil, noCleanup, err
	}
	if err := os.WriteFile(filepath.Join(dir, "cli-config.json"), out, 0o644); err != nil {
		cleanup()
		cleanup = noCleanup
		return nil, noCleanup, err
	}

	return []string{"CURSOR_CONFIG_DIR=" + dir}, cleanup, nil
}

func normalizePermissionTokens(in []string) ([]string, error) {
	var out []string
	seen := make(map[string]struct{})
	for i, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("entry %d is empty", i)
		}
		tok := toCursorPermissionToken(s)
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		out = append(out, tok)
	}
	slices.Sort(out)
	return out, nil
}

func toCursorPermissionToken(s string) string {
	if strings.Contains(s, "(") {
		return s
	}
	return "Shell(" + s + ")"
}
