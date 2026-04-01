package jrdev

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestToCursorPermissionToken(t *testing.T) {
	if got := toCursorPermissionToken("git"); got != "Shell(git)" {
		t.Fatalf("bare: %q", got)
	}
	if got := toCursorPermissionToken("Shell(go)"); got != "Shell(go)" {
		t.Fatalf("already wrapped: %q", got)
	}
	if got := toCursorPermissionToken("Read(**)"); got != "Read(**)" {
		t.Fatalf("read glob: %q", got)
	}
}

func TestNormalizePermissionTokens_dedupeAndSort(t *testing.T) {
	out, err := normalizePermissionTokens([]string{"git", " Shell(git) ", "go"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Shell(git)", "Shell(go)"}
	if !slices.Equal(out, want) {
		t.Fatalf("got %v want %v", out, want)
	}
}

func TestNormalizePermissionTokens_rejectsEmpty(t *testing.T) {
	_, err := normalizePermissionTokens([]string{"git", ""})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProjectCursorConfigDir(t *testing.T) {
	dir := t.TempDir()
	if _, ok := ProjectCursorConfigDir(dir); ok {
		t.Fatal("expected missing")
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := ProjectCursorConfigDir(dir); ok {
		t.Fatal("expected missing without cli-config.json")
	}
	cli := filepath.Join(dir, ".cursor", ProjectCursorCLIConfigName)
	if err := os.WriteFile(cli, []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := ProjectCursorConfigDir(dir)
	if !ok || got != filepath.Join(dir, ".cursor") {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestPrepareAgentCursorEnv_permissionsFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "perm.json")
	if err := os.WriteFile(p, []byte(`{"allow":["git","go","Shell(gh)"],"deny":["rm"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{AgentPermissionsFile: p}
	env, cleanup, err := PrepareAgentCursorEnv(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(env) != 1 || !strings.HasPrefix(env[0], "CURSOR_CONFIG_DIR=") {
		t.Fatalf("env: %v", env)
	}
	configDir := strings.TrimPrefix(env[0], "CURSOR_CONFIG_DIR=")
	raw, err := os.ReadFile(filepath.Join(configDir, "cli-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var parsed cursorCLIConfig
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	allowStr := strings.Join(parsed.Permissions.Allow, ",")
	for _, tok := range []string{"Shell(git)", "Shell(go)", "Shell(gh)", "Read(**)", "Write(**)"} {
		if !strings.Contains(allowStr, tok) {
			t.Fatalf("allow missing %q: %v", tok, parsed.Permissions.Allow)
		}
	}
	denyStr := strings.Join(parsed.Permissions.Deny, ",")
	if !strings.Contains(denyStr, "Shell(rm)") {
		t.Fatalf("deny: %v", parsed.Permissions.Deny)
	}
}
