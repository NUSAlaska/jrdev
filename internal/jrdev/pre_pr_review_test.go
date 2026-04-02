package jrdev

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPrePRRunID_length(t *testing.T) {
	id, err := newPrePRRunID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 8 {
		t.Fatalf("runId %q len=%d", id, len(id))
	}
	for _, r := range id {
		if r < '0' || r > 'f' || (r > '9' && r < 'a') {
			t.Fatalf("non-hex in %q", id)
		}
	}
}

func TestFormatPriorPassArtifacts_firstRound(t *testing.T) {
	s, err := formatPriorPassArtifacts(t.TempDir(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if s != "(none — first round)" {
		t.Fatalf("got %q", s)
	}
}

func TestFormatPriorPassArtifacts_missingFiles(t *testing.T) {
	s, err := formatPriorPassArtifacts(t.TempDir(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if s != "(prior artifact files not found)" {
		t.Fatalf("got %q", s)
	}
}

func TestFormatPriorPassArtifacts_roundFiles(t *testing.T) {
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "round-01-pass-1.json")
	if err := os.WriteFile(p1, []byte(`{"k":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p2 := filepath.Join(tmp, "round-01-pass-2.json")
	if err := os.WriteFile(p2, []byte(`{"round":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := formatPriorPassArtifacts(tmp, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "round-01-pass-1") || !strings.Contains(s, `"k":1`) {
		t.Fatalf("missing pass1: %q", s)
	}
	if !strings.Contains(s, "round-01-pass-2") || !strings.Contains(s, `"round":1`) {
		t.Fatalf("missing pass2: %q", s)
	}
}

func TestParsePass2Handoff_valid(t *testing.T) {
	out := "preamble\n```" + pass2HandoffFenceTag + "\n" + `{"round":2,"gapSummary":"g","matrixDelta":"m","draftPRTitle":"t","draftPRBody":"b","gapNotes":"gn","conflictNotes":"cn"}` + "\n```\nCOMPLETE\n"
	h, raw, perr := parsePass2Handoff(out)
	if perr != "" {
		t.Fatalf("parseErr: %s", perr)
	}
	if raw == "" || !strings.Contains(raw, "gapSummary") {
		t.Fatalf("raw=%q", raw)
	}
	if h.Round != 2 || h.GapSummary != "g" || h.DraftPRTitle != "t" || h.ConflictNotes != "cn" {
		t.Fatalf("%+v", h)
	}
}

func TestParsePass2Handoff_noFence(t *testing.T) {
	_, _, perr := parsePass2Handoff("no fence here")
	if perr == "" {
		t.Fatal("expected error")
	}
	if !strings.Contains(perr, "fenced") {
		t.Fatalf("got %q", perr)
	}
}

func TestParsePass2Handoff_invalidJSON(t *testing.T) {
	out := "```" + pass2HandoffFenceTag + "\nnot-json\n```"
	_, raw, perr := parsePass2Handoff(out)
	if perr == "" {
		t.Fatal("expected JSON error")
	}
	if raw != "not-json" {
		t.Fatalf("raw=%q", raw)
	}
}

func TestParsePass3Artifact_valid(t *testing.T) {
	raw := `{"summary":"s","testDesign":"td","coverage":"c","prdTestingAlignment":"p","strictnessInference":"i","followUps":"f"}`
	out := "intro\n```" + pass3ArtifactFenceTag + "\n" + raw + "\n```\nCOMPLETE\n"
	a, inner, perr := parsePass3Artifact(out)
	if perr != "" {
		t.Fatalf("parseErr: %s", perr)
	}
	if inner != raw {
		t.Fatalf("inner=%q", inner)
	}
	if a.Summary != "s" || a.TestDesign != "td" || a.FollowUps != "f" {
		t.Fatalf("%+v", a)
	}
}

func TestParsePass3Artifact_noFence(t *testing.T) {
	_, _, perr := parsePass3Artifact("no fence")
	if perr == "" {
		t.Fatal("expected error")
	}
}

func TestParsePass3Artifact_invalidJSON(t *testing.T) {
	out := "```" + pass3ArtifactFenceTag + "\nnot-json\n```"
	_, raw, perr := parsePass3Artifact(out)
	if perr == "" {
		t.Fatal("expected JSON error")
	}
	if raw != "not-json" {
		t.Fatalf("raw=%q", raw)
	}
}

func TestParsePass3Artifact_twoFences(t *testing.T) {
	two := "```" + pass3ArtifactFenceTag + "\n{}\n```\n```" + pass3ArtifactFenceTag + "\n{}\n```"
	_, _, perr := parsePass3Artifact(two)
	if perr == "" || !strings.Contains(perr, "exactly one") {
		t.Fatalf("got %q", perr)
	}
}

func TestParsePass3Artifact_unclosedFence(t *testing.T) {
	_, _, perr := parsePass3Artifact("```" + pass3ArtifactFenceTag + "\n{\"x\":1}\n")
	if perr == "" || !strings.Contains(perr, "unclosed") {
		t.Fatalf("expected unclosed fence error, got %q", perr)
	}
}

func TestWritePass3Artifact_success(t *testing.T) {
	tmp := t.TempDir()
	art := PrePRPass3Artifact{
		Summary:             "s",
		TestDesign:          "td",
		Coverage:            "c",
		PRDTestingAlignment: "p",
		StrictnessInference: "i",
		FollowUps:           "f",
	}
	raw := `{"summary":"s","testDesign":"td"}`
	if err := writePass3Artifact(tmp, 2, art, raw, ""); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(tmp, "pass-3.json"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		FinalRound  int                `json:"finalRound"`
		Artifact    PrePRPass3Artifact `json:"artifact"`
		ArtifactRaw string             `json:"artifactFenceInner,omitempty"`
		ParseErr    string             `json:"artifactParseError,omitempty"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	if w.FinalRound != 2 || w.ArtifactRaw != "" || w.ParseErr != "" {
		t.Fatalf("%+v", w)
	}
	if w.Artifact.Summary != "s" || w.Artifact.TestDesign != "td" {
		t.Fatalf("%+v", w.Artifact)
	}
}

func TestWritePass3Artifact_parseErrorPreservesRaw(t *testing.T) {
	tmp := t.TempDir()
	if err := writePass3Artifact(tmp, 1, PrePRPass3Artifact{}, "not-json", "invalid character"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(tmp, "pass-3.json"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		FinalRound int `json:"finalRound"`
		Artifact   PrePRPass3Artifact
		Raw        string `json:"artifactFenceInner,omitempty"`
		ParseErr   string `json:"artifactParseError,omitempty"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	if w.FinalRound != 1 || w.Raw != "not-json" || !strings.Contains(w.ParseErr, "invalid") {
		t.Fatalf("finalRound=%d raw=%q err=%q", w.FinalRound, w.Raw, w.ParseErr)
	}
}

func TestParsePass5HandoffOptional_absent(t *testing.T) {
	h, raw, perr := parsePass5HandoffOptional("COMPLETE\nno fence")
	if perr != "" || raw != "" || h != (PrePRPass5Handoff{}) {
		t.Fatalf("h=%+v raw=%q perr=%q", h, raw, perr)
	}
}

func TestParsePass5HandoffOptional_valid(t *testing.T) {
	body := "```" + pass5HandoffFenceTag + "\n" + `{"badTestByDesign":"x","operatorNotes":"y"}` + "\n```\n"
	h, raw, perr := parsePass5HandoffOptional(body)
	if perr != "" {
		t.Fatal(perr)
	}
	if h.BadTestByDesign != "x" || h.OperatorNotes != "y" || raw == "" {
		t.Fatalf("%+v raw=%q", h, raw)
	}
}

func TestParsePass5HandoffOptional_proseMentionWithoutFence(t *testing.T) {
	out := "COMPLETE\nSee docs for " + pass5HandoffFenceTag + " format.\n"
	h, raw, perr := parsePass5HandoffOptional(out)
	if perr != "" || raw != "" || h != (PrePRPass5Handoff{}) {
		t.Fatalf("optional handoff absent when only tag is mentioned in prose: h=%+v raw=%q perr=%q", h, raw, perr)
	}
}

func TestParsePass5HandoffOptional_invalidJSONInFence(t *testing.T) {
	body := "```" + pass5HandoffFenceTag + "\nnot-json\n```\n"
	_, raw, perr := parsePass5HandoffOptional(body)
	if perr == "" {
		t.Fatal("expected JSON parse error")
	}
	if raw != "not-json" {
		t.Fatalf("raw=%q", raw)
	}
}

func TestParsePass5HandoffOptional_twoFencesErrors(t *testing.T) {
	tag := pass5HandoffFenceTag
	body := "```" + tag + "\n{}\n```\n```" + tag + "\n{}\n```\n"
	_, _, perr := parsePass5HandoffOptional(body)
	if perr == "" || !strings.Contains(perr, "exactly one") {
		t.Fatalf("got %q", perr)
	}
}

func TestMergePass5HandoffIntoSessionFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "handoff.json")
	sess := PrePRSessionHandoff{DraftPRTitle: "t", FinalRound: 2, MatrixHadGaps: false}
	b, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergePass5HandoffIntoSessionFile(path, PrePRPass5Handoff{BadTestByDesign: "bad", OperatorNotes: "note"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got PrePRSessionHandoff
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.DraftPRTitle != "t" || got.Pass5BadTestByDesign != "bad" || got.Pass5OperatorNotes != "note" {
		t.Fatalf("%+v", got)
	}
}

func TestMergeSessionHandoff_trimsAndFlags(t *testing.T) {
	last := PrePRPass2Handoff{
		DraftPRTitle:  "  t  ",
		DraftPRBody:   " b ",
		GapNotes:      "\tx\n",
		ConflictNotes: " y ",
	}
	got := mergeSessionHandoff(last, 3, true)
	if got.DraftPRTitle != "t" || got.DraftPRBody != "b" || got.GapNotes != "x" || got.ConflictNotes != "y" {
		t.Fatalf("%+v", got)
	}
	if got.FinalRound != 3 || !got.MatrixHadGaps {
		t.Fatalf("%+v", got)
	}
}
