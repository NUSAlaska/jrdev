package jrdev

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildPass4CheckPlan_orderAndSkipEmpty(t *testing.T) {
	p := ProjectConfig{
		Lint:        []string{"lint-a", "  ", "lint-b"},
		Unit:        []string{"unit-1"},
		Integration: []string{"", "int-z"},
	}
	plan := BuildPass4CheckPlan(p)
	if len(plan) != 4 {
		t.Fatalf("len=%d %#v", len(plan), plan)
	}
	want := []Pass4PlannedCheck{
		{Kind: Pass4Lint, Command: "lint-a"},
		{Kind: Pass4Lint, Command: "lint-b"},
		{Kind: Pass4Unit, Command: "unit-1"},
		{Kind: Pass4Integration, Command: "int-z"},
	}
	for i := range plan {
		if plan[i].Kind != want[i].Kind || plan[i].Command != want[i].Command {
			t.Fatalf("i=%d got %#v want %#v", i, plan[i], want[i])
		}
	}
}

func TestNormalizeCheckFailureKey_timestampsAndWhitespace(t *testing.T) {
	raw := "2026-04-02T15:04:05Z boom\nfail at 2026-04-02\n12:34:56  ok	\n(1.23s) (44.00s)"
	got := NormalizeCheckFailureKey(raw)
	for _, w := range []string{"<ts>", "<date>", "<clock>", "<dur>"} {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %q", w, got)
		}
	}
	if strings.Contains(got, "2026") {
		t.Fatalf("raw date leaked: %q", got)
	}
}

func TestPass4FailureFingerprint_includesKind(t *testing.T) {
	fp := Pass4FailureFingerprint(Pass4Lint, "error: same")
	if !strings.HasPrefix(fp, "lint\x1e") {
		t.Fatalf("got %q", fp)
	}
}

func TestPass4FailureFingerprint_kindSeparatesSameLog(t *testing.T) {
	log := "FAIL same error line"
	lint := Pass4FailureFingerprint(Pass4Lint, log)
	unit := Pass4FailureFingerprint(Pass4Unit, log)
	if lint == unit {
		t.Fatalf("expected different fingerprints for different kinds: %q", lint)
	}
}

func TestNormalizeCheckFailureKey_empty(t *testing.T) {
	if got := NormalizeCheckFailureKey(""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeCheckFailureKey("   \n\t  "); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestTruncatePass4Output_longSuccessTruncates(t *testing.T) {
	const n = 2500
	s := strings.Repeat("x", n)
	got := truncatePass4Output(s, true)
	if !strings.HasSuffix(got, "\n…(truncated)") {
		t.Fatalf("missing suffix: len=%d", len(got))
	}
	if len(got) != 2048+len("\n…(truncated)") {
		t.Fatalf("len=%d want %d", len(got), 2048+len("\n…(truncated)"))
	}
}

func TestTruncatePass4Output_failureNotTruncated(t *testing.T) {
	s := strings.Repeat("e", 5000)
	if got := truncatePass4Output(s, false); got != s {
		t.Fatalf("expected raw failure log preserved")
	}
}

func TestRunPass4Checks_emptyPlanSucceeds(t *testing.T) {
	out := RunPass4Checks(t.TempDir(), nil)
	if !out.Success || len(out.Steps) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestRunPass4Checks_stopsOnFirstFailure(t *testing.T) {
	tmp := t.TempDir()
	var failLine, okLine string
	if runtime.GOOS == "windows" {
		failLine = "cmd /C exit /b 1"
		okLine = "cmd /C echo second-should-not-run"
	} else {
		failLine = "false"
		okLine = "echo second-should-not-run"
	}
	plan := []Pass4PlannedCheck{
		{Kind: Pass4Lint, Command: failLine},
		{Kind: Pass4Unit, Command: okLine},
	}
	out := RunPass4Checks(tmp, plan)
	if out.Success {
		t.Fatal("expected failure")
	}
	if len(out.Steps) != 1 {
		t.Fatalf("steps=%d want 1 (stop early)", len(out.Steps))
	}
	if out.FailedStep == nil || out.FailedStep.Kind != Pass4Lint {
		t.Fatalf("%+v", out.FailedStep)
	}
}

func TestRunPass4Checks_runsEchoInOrder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("trace file ordering test uses sh redirection")
	}
	tmp := t.TempDir()
	trace := filepath.Join(tmp, "trace.txt")
	plan := []Pass4PlannedCheck{
		{Kind: Pass4Lint, Command: "sh -c 'echo L >> " + trace + "'"},
		{Kind: Pass4Unit, Command: "sh -c 'echo U >> " + trace + "'"},
		{Kind: Pass4Integration, Command: "sh -c 'echo I >> " + trace + "'"},
	}
	out := RunPass4Checks(tmp, plan)
	if !out.Success || len(out.Steps) != 3 {
		t.Fatalf("%+v", out)
	}
	b, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "L\nU\nI\n" {
		t.Fatalf("trace=%q", b)
	}
}
