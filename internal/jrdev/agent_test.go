package jrdev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortPromptForFile_mentionsBasename(t *testing.T) {
	s := shortPromptForFile("jrdev-agent-prompt-abc123.txt")
	if !strings.Contains(s, "jrdev-agent-prompt-abc123.txt") {
		t.Fatalf("expected basename in meta-prompt: %q", s)
	}
}

func TestWriteAgentPromptFile_roundTrip(t *testing.T) {
	dir := t.TempDir()
	content := "hello\nworld"
	path, err := writeAgentPromptFile(dir, content)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(path), "jrdev-agent-prompt-") {
		t.Fatalf("unexpected basename %q", filepath.Base(path))
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("got %q want %q", b, content)
	}
}
