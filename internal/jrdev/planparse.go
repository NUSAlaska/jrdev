package jrdev

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrPlanNotFound is returned when <plan>...</plan> is missing.
	ErrPlanNotFound = errors.New("jrdev: missing <plan>...</plan> wrapper")
	// ErrPlanInvalidJSON is returned when inner JSON does not decode.
	ErrPlanInvalidJSON = errors.New("jrdev: plan JSON invalid")
)

// ParsePlan extracts the first <plan>...</plan> block and decodes issues.
func ParsePlan(agentOutput string) (PlanDocument, error) {
	start := strings.Index(agentOutput, "<plan>")
	end := strings.Index(agentOutput, "</plan>")
	if start < 0 || end < 0 || end <= start {
		return PlanDocument{}, ErrPlanNotFound
	}
	inner := agentOutput[start+len("<plan>") : end]
	inner = strings.TrimSpace(inner)

	var doc PlanDocument
	dec := json.NewDecoder(bytes.NewReader([]byte(inner)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return PlanDocument{}, fmt.Errorf("%w: %v", ErrPlanInvalidJSON, err)
	}
	for i, iss := range doc.Issues {
		if iss.Number <= 0 {
			return PlanDocument{}, fmt.Errorf("%w: issues[%d].number must be positive", ErrPlanInvalidJSON, i)
		}
		if strings.TrimSpace(iss.Title) == "" {
			return PlanDocument{}, fmt.Errorf("%w: issues[%d].title empty", ErrPlanInvalidJSON, i)
		}
		if strings.TrimSpace(iss.Branch) == "" {
			return PlanDocument{}, fmt.Errorf("%w: issues[%d].branch empty", ErrPlanInvalidJSON, i)
		}
	}
	return doc, nil
}
