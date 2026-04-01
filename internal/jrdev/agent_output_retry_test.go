package jrdev

import (
	"strings"
	"testing"
)

func TestAppendAgentOutputRetryInstructions(t *testing.T) {
	base := "BASE PROMPT"
	got := AppendAgentOutputRetryInstructions(base, "Plan phase", ErrPlanInvalidJSON, "first-out")
	if !strings.Contains(got, base) {
		t.Fatal("missing base")
	}
	if !strings.Contains(got, ErrPlanInvalidJSON.Error()) {
		t.Fatal("missing validation error")
	}
	if !strings.Contains(got, "first-out") {
		t.Fatal("missing previous output")
	}
	if !strings.Contains(got, "only retry") {
		t.Fatal("missing retry hint")
	}
}

func TestAppendAgentOutputRetryInstructions_truncatesPreviousOutput(t *testing.T) {
	long := strings.Repeat("x", agentOutputRetryPreviewMax+500)
	got := AppendAgentOutputRetryInstructions("base", "P", ErrPlanNotFound, long)
	if strings.Count(got, "x") < agentOutputRetryPreviewMax {
		t.Fatal("expected truncated preview to retain prefix runes")
	}
	if !strings.Contains(got, "truncated by jrdev") {
		t.Fatal("expected truncation notice")
	}
}

func TestValidateMergeAgentOutput(t *testing.T) {
	if err := ValidateMergeAgentOutput("ok\n" + CompletionMarker + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMergeAgentOutput("no marker here"); err == nil {
		t.Fatal("expected error")
	}
}
