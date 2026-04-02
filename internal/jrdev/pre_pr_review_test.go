package jrdev

import "testing"

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
