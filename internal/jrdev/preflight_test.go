package jrdev

import "testing"

func TestShouldRunAgentSmoke(t *testing.T) {
	if !ShouldRunAgentSmoke(Config{DryRun: false}) {
		t.Fatal("full run should run agent smoke")
	}
	if ShouldRunAgentSmoke(Config{DryRun: true}) {
		t.Fatal("dry-run should skip agent smoke")
	}
}
