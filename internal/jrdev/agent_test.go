package jrdev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortPromptForRelFile_mentionsPath(t *testing.T) {
	s := shortPromptForRelFile(".jrdev/agent-runs/1/prompt.txt")
	if !strings.Contains(s, ".jrdev/agent-runs/1/prompt.txt") {
		t.Fatalf("expected relative path in meta-prompt: %q", s)
	}
}

func TestWriteTextFile_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	content := "hello\nworld"
	if err := writeTextFile(path, content); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("got %q want %q", string(b), content)
	}
}

func TestEnsureAgentArtifactsTree_writesGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := ensureAgentArtifactsTree(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, AgentArtifactsDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), agentRunsSubdir) {
		t.Fatalf("gitignore should mention runs dir: %q", string(b))
	}
}
