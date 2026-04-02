package jrdev

import (
	"encoding/json"
	"fmt"
	"strings"
)

const prdMatrixFenceTag = "jrdev-prd-matrix"

var allowedMatrixStatuses = map[string]struct{}{
	"satisfied":      {},
	"not_satisfied":  {},
	"unknown":        {},
	"conflict":       {},
}

// PRDMatrixEvidence models per-row evidence (GM-005).
type PRDMatrixEvidence struct {
	Paths []string `json:"paths"`
	Tests []string `json:"tests"`
}

// PRDMatrixRow is one requirements matrix row.
type PRDMatrixRow struct {
	ID            string            `json:"id"`
	VerbatimQuote string            `json:"verbatimQuote"`
	Status        string            `json:"status"`
	Evidence      PRDMatrixEvidence `json:"evidence"`
	Notes         string            `json:"notes"`
}

// PRDMatrixDoc is the validated JSON object inside the jrdev-prd-matrix fence.
type PRDMatrixDoc struct {
	SchemaVersion string          `json:"schemaVersion"`
	Requirements  []PRDMatrixRow  `json:"requirements"`
}

// ExtractSinglePRDMatrixFence returns the inner text of exactly one ```jrdev-prd-matrix fenced block.
func ExtractSinglePRDMatrixFence(agentOutput string) (inner string, err error) {
	return ExtractSingleMarkdownFence(prdMatrixFenceTag, agentOutput)
}

// PRDMatrixDocHasGaps reports whether any requirement row still needs Pass 2 attention (GM-007).
func PRDMatrixDocHasGaps(doc PRDMatrixDoc) bool {
	for _, row := range doc.Requirements {
		switch strings.TrimSpace(row.Status) {
		case "not_satisfied", "unknown", "conflict":
			return true
		}
	}
	return false
}

// ValidatePRDMatrixJSON unmarshals and checks GM-005 field constraints.
func ValidatePRDMatrixJSON(jsonText string) (PRDMatrixDoc, error) {
	var doc PRDMatrixDoc
	if err := json.Unmarshal([]byte(jsonText), &doc); err != nil {
		return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: %w", err)
	}
	if strings.TrimSpace(doc.SchemaVersion) == "" {
		return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: missing schemaVersion")
	}
	if doc.Requirements == nil {
		return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements must be present")
	}
	for i, row := range doc.Requirements {
		if strings.TrimSpace(row.ID) == "" {
			return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d].id empty", i)
		}
		if strings.TrimSpace(row.VerbatimQuote) == "" {
			return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d].verbatimQuote empty", i)
		}
		st := strings.TrimSpace(row.Status)
		if _, ok := allowedMatrixStatuses[st]; !ok {
			return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d].status invalid %q", i, row.Status)
		}
		if row.Evidence.Paths == nil {
			return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d].evidence.paths must be present (array)", i)
		}
		if row.Evidence.Tests == nil {
			return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d].evidence.tests must be present (array)", i)
		}
		if st == "satisfied" {
			ok := false
			for _, t := range row.Evidence.Tests {
				if strings.TrimSpace(t) != "" {
					ok = true
					break
				}
			}
			if !ok {
				return PRDMatrixDoc{}, fmt.Errorf("jrdev: matrix json: requirements[%d] status satisfied requires concrete evidence.tests", i)
			}
		}
	}
	return doc, nil
}

// PrePRMatrixRepairPrompt builds the minimal one-shot repair prompt (failing JSON + error only).
func PrePRMatrixRepairPrompt(fencedInnerJSON, parseErr string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You have invalid JSON that must appear inside a single markdown fenced block tagged exactly %q.\n\n", prdMatrixFenceTag)
	fmt.Fprintf(&b, "Parse/validation error:\n%s\n\n", strings.TrimSpace(parseErr))
	fmt.Fprintf(&b, "Failing JSON text (fix it; output only the corrected fenced block and nothing else before it):\n```%s\n%s\n```\n\n", prdMatrixFenceTag, strings.TrimSpace(fencedInnerJSON))
	fmt.Fprintf(&b, "Output requirements:\n")
	fmt.Fprintf(&b, "- Emit exactly one ```%s fenced block containing a single JSON object.\n", prdMatrixFenceTag)
	fmt.Fprintf(&b, "- The JSON must satisfy the PRD requirements matrix schema (schemaVersion, requirements[] with id, verbatimQuote, status, evidence.paths, evidence.tests, notes).\n")
	fmt.Fprintf(&b, "- End your response with the line COMPLETE on its own line.\n")
	return b.String()
}
