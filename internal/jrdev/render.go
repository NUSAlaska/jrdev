package jrdev

import (
	"bytes"
	"strings"
	"text/template"
)

// PlanPromptData is passed to the plan markdown template.
type PlanPromptData struct {
	QueueLabel    string
	IssuesJSON    string // preloaded gh issue list JSON
	CommitHistory string // git log snippet for agent cwd (integration worktree)
	GitDiff       string // git diff main..HEAD in agent cwd
}

// ImplementPromptData is passed to the implement template.
type ImplementPromptData struct {
	IssueNumber       int
	IssueTitle        string
	IssueBody         string
	IssueBranch       string
	IntegrationBranch string
	QueueLabel        string
	CommitHistory     string // git log snippet for agent cwd (issue worktree)
	GitDiff           string // git diff main..HEAD in agent cwd
}

// ReviewPromptData is passed to the review template.
type ReviewPromptData = ImplementPromptData

// MergePromptData is passed to the merge template.
type MergePromptData struct {
	IssueNumber       int
	IssueTitle        string
	IssueBranch       string
	IntegrationBranch string
	QueueLabel        string
	CommitHistory     string // git log snippet for agent cwd (integration worktree)
	GitDiff           string // git diff main..HEAD in agent cwd
}

// Render treats body as a Go text/template with the given name for errors.
func Render(name, body string, data any) (string, error) {
	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
