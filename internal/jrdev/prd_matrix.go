package jrdev

import (
	"encoding/json"
	"fmt"
	"regexp"
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

var fenceStartRe = regexp.MustCompile("(?i)^```\\s*" + regexp.QuoteMeta(prdMatrixFenceTag) + "\\s*$")

// ExtractSinglePRDMatrixFence returns the inner text of exactly one ```jrdev-prd-matrix fenced block.
func ExtractSinglePRDMatrixFence(agentOutput string) (inner string, err error) {
	lines := strings.Split(agentOutput, "\n")
	var starts []int
	for i, line := range lines {
		if fenceStartRe.MatchString(strings.TrimRight(line, "\r")) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return "", fmt.Errorf("jrdev: no %q fenced code block in agent output", prdMatrixFenceTag)
	}
	if len(starts) > 1 {
		return "", fmt.Errorf("jrdev: expected exactly one %q fenced block, found %d", prdMatrixFenceTag, len(starts))
	}
	startIdx := starts[0]
	var endIdx int
	foundEnd := false
	for j := startIdx + 1; j < len(lines); j++ {
		if strings.HasPrefix(strings.TrimRight(lines[j], "\r"), "```") {
			endIdx = j
			foundEnd = true
			break
		}
	}
	if !foundEnd {
		return "", fmt.Errorf("jrdev: unclosed %q fenced code block", prdMatrixFenceTag)
	}
	var b strings.Builder
	for j := startIdx + 1; j < endIdx; j++ {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.TrimRight(lines[j], "\r"))
	}
	return strings.TrimSpace(b.String()), nil
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
