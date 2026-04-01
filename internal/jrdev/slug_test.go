package jrdev

import "testing"

func TestIssueSlug(t *testing.T) {
	if g := IssueSlug("Fix the API  !!"); g != "fix-the-api" {
		t.Fatalf("%q", g)
	}
	if g := IssueSlug("   "); g != "issue" {
		t.Fatalf("%q", g)
	}
}
