package jrdev

import (
	"strings"
	"testing"
)

func TestParsePRMetadata(t *testing.T) {
	out := `Some intro

<pr_title>
Add PR description generation to jrdev
</pr_title>
<pr_body>
## Summary

- Runs agent before gh pr create

## Notes

**Label:** agent-queue
</pr_body>

COMPLETE
`
	meta, err := ParsePRMetadata(out)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := meta.Title, "Add PR description generation to jrdev"; got != want {
		t.Fatalf("title=%q want %q", got, want)
	}
	if !strings.Contains(meta.Body, "Runs agent") {
		t.Fatalf("body missing expected text, got %q", meta.Body)
	}
}

func TestParsePRMetadata_errors(t *testing.T) {
	_, err := ParsePRMetadata("no tags")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParsePRMetadata("<pr_title></pr_title><pr_body>x</pr_body>")
	if err == nil || !strings.Contains(err.Error(), "pr_title empty") {
		t.Fatalf("want pr_title empty, got %v", err)
	}
	_, err = ParsePRMetadata("<pr_title>ok</pr_title><pr_body>  \n  </pr_body>")
	if err == nil || !strings.Contains(err.Error(), "pr_body empty") {
		t.Fatalf("want pr_body empty, got %v", err)
	}
}
