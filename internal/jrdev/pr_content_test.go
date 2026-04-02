package jrdev

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPRPromptPrePRReviewHandoff_noLatest(t *testing.T) {
	tmp := t.TempDir()
	h, warn := LoadPRPromptPrePRReviewHandoff(tmp)
	if h.Present || h.Summary != "" || h.ArtifactPaths != "" {
		t.Fatalf("expected empty handoff, got present=%v summary=%q paths=%q", h.Present, h.Summary, h.ArtifactPaths)
	}
	if warn == "" || !strings.Contains(warn, "latest") {
		t.Fatalf("expected warning about latest, got %q", warn)
	}
}

func TestLoadPRPromptPrePRReviewHandoff_missingRunDir(t *testing.T) {
	tmp := t.TempDir()
	latest := filepath.Join(tmp, AgentArtifactsDir, PrePrReviewArtifactsRoot, "latest")
	if err := os.MkdirAll(filepath.Dir(latest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latest, []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, warn := LoadPRPromptPrePRReviewHandoff(tmp)
	if h.Present {
		t.Fatal("expected present false")
	}
	if warn == "" || !strings.Contains(warn, "missing") {
		t.Fatalf("expected missing dir warning, got %q", warn)
	}
}

func TestLoadPRPromptPrePRReviewHandoff_withArtifacts(t *testing.T) {
	tmp := t.TempDir()
	runID := "a1b2c3d4"
	artDir := filepath.Join(tmp, AgentArtifactsDir, PrePrReviewArtifactsRoot, runID)
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	latest := filepath.Join(tmp, AgentArtifactsDir, PrePrReviewArtifactsRoot, "latest")
	if err := os.WriteFile(latest, []byte(runID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := PrePRSessionHandoff{
		DraftPRTitle:  "Draft title",
		DraftPRBody:   "Draft body line",
		GapNotes:      "gaps",
		FinalRound:    2,
		MatrixHadGaps: false,
	}
	sb, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "handoff.json"), sb, 0o644); err != nil {
		t.Fatal(err)
	}
	p3 := struct {
		FinalRound int                `json:"finalRound"`
		Artifact   PrePRPass3Artifact `json:"artifact"`
	}{FinalRound: 2, Artifact: PrePRPass3Artifact{Summary: "pass3sum", FollowUps: "fu"}}
	p3b, err := json.MarshalIndent(p3, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "pass-3.json"), p3b, 0o644); err != nil {
		t.Fatal(err)
	}
	p4 := prePRPass4ArtifactFile{FinalPass4Success: true}
	p4b, err := json.MarshalIndent(p4, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "pass-4.json"), p4b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "round-01-pass-1.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	h, warn := LoadPRPromptPrePRReviewHandoff(tmp)
	if warn != "" {
		t.Fatalf("unexpected warn: %s", warn)
	}
	if !h.Present {
		t.Fatal("expected present")
	}
	if !strings.Contains(h.Summary, "Draft title") || !strings.Contains(h.Summary, "pass3sum") {
		t.Fatalf("summary: %q", h.Summary)
	}
	if !strings.Contains(h.Summary, "Final Pass 4 success:** true") {
		t.Fatalf("expected pass4 in summary: %q", h.Summary)
	}
	if !strings.Contains(h.ArtifactPaths, "handoff.json") || !strings.Contains(h.ArtifactPaths, "round-01-pass-1.json") {
		t.Fatalf("paths: %q", h.ArtifactPaths)
	}
}

func TestLoadPRPromptPrePRReviewHandoff_warnPartial(t *testing.T) {
	tmp := t.TempDir()
	runID := "a1b2c3d4"
	artDir := filepath.Join(tmp, AgentArtifactsDir, PrePrReviewArtifactsRoot, runID)
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	latest := filepath.Join(tmp, AgentArtifactsDir, PrePrReviewArtifactsRoot, "latest")
	if err := os.WriteFile(latest, []byte(runID+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Only handoff — missing pass-3 triggers secondary warning
	if err := os.WriteFile(filepath.Join(artDir, "handoff.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	h, warn := LoadPRPromptPrePRReviewHandoff(tmp)
	if !h.Present {
		t.Fatal("expected present")
	}
	if warn == "" || !strings.Contains(warn, "pass-3.json") {
		t.Fatalf("expected pass-3 warning, got %q", warn)
	}
}
