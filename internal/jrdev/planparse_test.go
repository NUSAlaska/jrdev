package jrdev

import (
	"errors"
	"testing"
)

func TestParsePlan_Golden(t *testing.T) {
	raw := `Some chatter
<plan>
{ "issues": [ { "number": 42, "title": "Fix thing", "branch": "agent-queue/issue-42-fix-thing" } ] }
</plan>
tail`
	doc, err := ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Issues) != 1 {
		t.Fatalf("len %d", len(doc.Issues))
	}
	if doc.Issues[0].Number != 42 || doc.Issues[0].Title != "Fix thing" {
		t.Fatalf("%+v", doc.Issues[0])
	}
}

func TestParsePlan_EmptyIssues(t *testing.T) {
	raw := `<plan>{"issues":[]}</plan>`
	doc, err := ParsePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Issues) != 0 {
		t.Fatal("expected empty")
	}
}

func TestParsePlan_Errors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want error
	}{
		{"no wrapper", `{"issues":[]}`, ErrPlanNotFound},
		{"unclosed", `<plan>{"issues":[]}`, ErrPlanNotFound},
		{"bad json", `<plan>not json</plan>`, ErrPlanInvalidJSON},
		{"unknown field", `<plan>{"issues":[],"foo":1}</plan>`, ErrPlanInvalidJSON},
		{"bad number", `<plan>{"issues":[{"number":0,"title":"t","branch":"b"}]}</plan>`, ErrPlanInvalidJSON},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePlan(tt.in)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("err %v, want %v", err, tt.want)
			}
		})
	}
}
