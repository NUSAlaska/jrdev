package jrdev

import (
	"strings"
	"testing"
)

func TestIntegrationBlockedFromStdout(t *testing.T) {
	tests := []struct {
		name        string
		stdout      string
		wantBlocked bool
		wantReason  string
	}{
		{
			name:        "no token",
			stdout:      "hello\nCOMPLETE\n",
			wantBlocked: false,
		},
		{
			name:        "token only",
			stdout:      "some log\nJRDEV_INTEGRATION_BLOCKED:\nCOMPLETE\n",
			wantBlocked: true,
		},
		{
			name:        "token with reason",
			stdout:      "JRDEV_INTEGRATION_BLOCKED: go test failed in pkg/foo",
			wantBlocked: true,
			wantReason:  "go test failed in pkg/foo",
		},
		{
			name:        "CRLF line",
			stdout:      "x\r\nJRDEV_INTEGRATION_BLOCKED: reason\r\n",
			wantBlocked: true,
			wantReason:  "reason",
		},
		{
			name:        "leading whitespace on line",
			stdout:      "  JRDEV_INTEGRATION_BLOCKED: trimmed still matches prefix after trim",
			wantBlocked: true,
			wantReason:  "trimmed still matches prefix after trim",
		},
		{
			name:        "substring in middle of line does not count",
			stdout:      "note: JRDEV_INTEGRATION_BLOCKED: not at line start",
			wantBlocked: false,
		},
		{
			name:        "first matching line wins",
			stdout:      "JRDEV_INTEGRATION_BLOCKED: first\nJRDEV_INTEGRATION_BLOCKED: second\n",
			wantBlocked: true,
			wantReason:  "first",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBlocked, gotReason := IntegrationBlockedFromStdout(tt.stdout)
			if gotBlocked != tt.wantBlocked {
				t.Fatalf("blocked=%v want %v", gotBlocked, tt.wantBlocked)
			}
			if gotReason != tt.wantReason {
				t.Fatalf("reason=%q want %q", gotReason, tt.wantReason)
			}
		})
	}
}

func TestResolveIntegrationBlockedDecision_CLIOverridesMeta(t *testing.T) {
	cfg := Config{
		IntegrationBlocked: "merge",
		Project: ProjectConfig{
			Meta: map[string]any{MetaIntegrationBlockedAction: "abort"},
		},
	}
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader(""), nil, nil, "", false)
	if err != nil || !w {
		t.Fatalf("want merge waive, got waive=%v err=%v", w, err)
	}
}

func TestResolveIntegrationBlockedDecision_metaAbort(t *testing.T) {
	cfg := Config{
		Project: ProjectConfig{
			Meta: map[string]any{MetaIntegrationBlockedAction: "abort"},
		},
	}
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader(""), nil, nil, "", false)
	if err != nil || w {
		t.Fatalf("want abort, got waive=%v err=%v", w, err)
	}
}

func TestResolveIntegrationBlockedDecision_metaMerge(t *testing.T) {
	cfg := Config{
		Project: ProjectConfig{
			Meta: map[string]any{MetaIntegrationBlockedAction: "merge"},
		},
	}
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader(""), nil, nil, "", false)
	if err != nil || !w {
		t.Fatalf("want waive, got waive=%v err=%v", w, err)
	}
}

func TestResolveIntegrationBlockedDecision_nonInteractiveDefaultAbort(t *testing.T) {
	cfg := Config{Project: ProjectConfig{}}
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader(""), nil, nil, "", false)
	if err != nil || w {
		t.Fatalf("want default abort, got waive=%v err=%v", w, err)
	}
}

func TestResolveIntegrationBlockedDecision_invalidCLI(t *testing.T) {
	cfg := Config{IntegrationBlocked: "maybe"}
	_, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader(""), nil, nil, "", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveIntegrationBlockedDecision_promptWaive(t *testing.T) {
	cfg := Config{}
	var out strings.Builder
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader("m\n"), &out, nil, "tests", true)
	if err != nil || !w {
		t.Fatalf("want waive, got waive=%v err=%v", w, err)
	}
}

func TestResolveIntegrationBlockedDecision_promptAbort(t *testing.T) {
	cfg := Config{}
	var out strings.Builder
	w, err := ResolveIntegrationBlockedDecision(cfg, strings.NewReader("\n"), &out, nil, "", true)
	if err != nil || w {
		t.Fatalf("want abort, got waive=%v err=%v", w, err)
	}
}

func TestMetaSaysWaiveIntegration_invalidIgnored(t *testing.T) {
	_, ok := metaSaysWaiveIntegration(map[string]any{MetaIntegrationBlockedAction: "nope"})
	if ok {
		t.Fatal("invalid meta value should be ignored")
	}
}
