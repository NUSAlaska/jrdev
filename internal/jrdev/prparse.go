package jrdev

import (
	"fmt"
	"strings"
)

const (
	prTitleOpen  = "<pr_title>"
	prTitleClose = "</pr_title>"
	prBodyOpen   = "<pr_body>"
	prBodyClose  = "</pr_body>"
)

// PRMetadata is the agent-produced title and body for gh pr create.
type PRMetadata struct {
	Title string
	Body  string
}

// ParsePRMetadata extracts <pr_title> and <pr_body> from agent stdout.
func ParsePRMetadata(agentOutput string) (PRMetadata, error) {
	title, err := extractTagged(agentOutput, prTitleOpen, prTitleClose)
	if err != nil {
		return PRMetadata{}, fmt.Errorf("jrdev: pr_title: %w", err)
	}
	title = strings.TrimSpace(title)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return PRMetadata{}, fmt.Errorf("jrdev: pr_title empty")
	}
	body, err := extractTagged(agentOutput, prBodyOpen, prBodyClose)
	if err != nil {
		return PRMetadata{}, fmt.Errorf("jrdev: pr_body: %w", err)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return PRMetadata{}, fmt.Errorf("jrdev: pr_body empty")
	}
	return PRMetadata{Title: title, Body: body}, nil
}

func extractTagged(s, open, close string) (string, error) {
	i := strings.Index(s, open)
	if i < 0 {
		return "", fmt.Errorf("missing %s", open)
	}
	start := i + len(open)
	j := strings.Index(s[start:], close)
	if j < 0 {
		return "", fmt.Errorf("missing %s", close)
	}
	return s[start : start+j], nil
}
