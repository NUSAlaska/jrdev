package jrdev

import (
	"strings"
	"testing"
)

func TestExtractSinglePRDMatrixFence(t *testing.T) {
	s := "intro\n\n```JrDev-PRD-Matrix\n{\"schemaVersion\":\"1\",\"requirements\":[]}\n```\n\nCOMPLETE\n"
	inner, err := ExtractSinglePRDMatrixFence(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inner, "schemaVersion") {
		t.Fatalf("inner=%q", inner)
	}

	_, err = ExtractSinglePRDMatrixFence("no fence")
	if err == nil {
		t.Fatal("expected error")
	}

	two := "```jrdev-prd-matrix\n{}\n```\n```jrdev-prd-matrix\n{}\n```"
	_, err = ExtractSinglePRDMatrixFence(two)
	if err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("got %v", err)
	}

	_, err = ExtractSinglePRDMatrixFence("```jrdev-prd-matrix\n{\"x\":1}\n")
	if err == nil || !strings.Contains(err.Error(), "unclosed") {
		t.Fatalf("expected unclosed fence error, got %v", err)
	}
}

func TestValidatePRDMatrixJSON(t *testing.T) {
	valid := `{
  "schemaVersion": "1",
  "requirements": [
    {
      "id": "REQ-1",
      "verbatimQuote": "Ship widget",
      "status": "satisfied",
      "evidence": { "paths": ["a.go"], "tests": ["a_test.go TestFoo"] },
      "notes": ""
    }
  ]
}`
	doc, err := ValidatePRDMatrixJSON(valid)
	if err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != "1" || len(doc.Requirements) != 1 {
		t.Fatalf("%+v", doc)
	}

	_, err = ValidatePRDMatrixJSON("{")
	if err == nil {
		t.Fatal("expected parse error")
	}

	badStatus := `{"schemaVersion":"1","requirements":[{"id":"a","verbatimQuote":"q","status":"maybe","evidence":{"paths":[],"tests":[]},"notes":""}]}`
	_, err = ValidatePRDMatrixJSON(badStatus)
	if err == nil {
		t.Fatal("expected status error")
	}

	satisfiedNoTests := `{"schemaVersion":"1","requirements":[{"id":"a","verbatimQuote":"q","status":"satisfied","evidence":{"paths":[],"tests":[]},"notes":""}]}`
	_, err = ValidatePRDMatrixJSON(satisfiedNoTests)
	if err == nil {
		t.Fatal("expected satisfied/tests error")
	}

	missingSchema := `{"requirements":[]}`
	_, err = ValidatePRDMatrixJSON(missingSchema)
	if err == nil {
		t.Fatal("expected schemaVersion error")
	}

	noRequirementsKey := `{"schemaVersion":"1"}`
	_, err = ValidatePRDMatrixJSON(noRequirementsKey)
	if err == nil || !strings.Contains(err.Error(), "requirements") {
		t.Fatalf("expected requirements error, got %v", err)
	}

	evidencePathsNull := `{"schemaVersion":"1","requirements":[{"id":"a","verbatimQuote":"q","status":"unknown","evidence":{"paths":null,"tests":[]},"notes":""}]}`
	_, err = ValidatePRDMatrixJSON(evidencePathsNull)
	if err == nil || !strings.Contains(err.Error(), "paths") {
		t.Fatalf("expected paths error, got %v", err)
	}
}

func TestPrePRMatrixRepairPrompt_containsFenceAndError(t *testing.T) {
	p := PrePRMatrixRepairPrompt(`{`, "broken")
	if !strings.Contains(p, prdMatrixFenceTag) || !strings.Contains(p, "broken") {
		t.Fatalf("%s", p)
	}
}
