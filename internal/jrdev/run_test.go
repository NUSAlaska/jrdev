package jrdev

import (
	"strings"
	"testing"
)

type sliceAgent struct {
	outs []string
	i    int
}

func (s *sliceAgent) Run(Config, string, string, AgentRunOptions) (string, error) {
	if s.i >= len(s.outs) {
		return "", nil
	}
	o := s.outs[s.i]
	s.i++
	return o, nil
}

func TestRunAgentUntilComplete(t *testing.T) {
	var renders int
	a := &sliceAgent{outs: []string{"no", "still no", "ok COMPLETE"}}
	cfg := Config{}
	_, err := runAgentUntilComplete(cfg, a, nil, "test", "/tmp", func() (string, error) {
		renders++
		return "prompt", nil
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if renders != 3 {
		t.Fatalf("renders=%d want 3", renders)
	}
	if a.i != 3 {
		t.Fatalf("agent calls=%d want 3", a.i)
	}
}

func TestRunAgentUntilComplete_exhausted(t *testing.T) {
	outs := make([]string, maxAgentStepAttempts)
	for i := range outs {
		outs[i] = "no marker"
	}
	a := &sliceAgent{outs: outs}
	cfg := Config{}
	_, err := runAgentUntilComplete(cfg, a, nil, "test", "/tmp", func() (string, error) {
		return "p", nil
	}, true)
	if err == nil || !strings.Contains(err.Error(), "never contained") {
		t.Fatalf("expected exhaustion error, got %v", err)
	}
	if a.i != maxAgentStepAttempts {
		t.Fatalf("agent calls=%d want %d", a.i, maxAgentStepAttempts)
	}
}
