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
	LintTests         string // markdown: configured lint commands (from repo .jrdev/config.yaml)
	UnitTests         string // markdown: configured unit commands
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
	LintTests         string
	UnitTests         string
	IntegrationTests  string
}

// PRPromptData is passed to the pull-request description template.
type PRPromptData struct {
	IntegrationBranch string
	QueueLabel        string
	PRBase            string // compare branch for the PR (e.g. main)
	CommitHistory     string
	GitDiff           string
}

// PrePRReviewPass1PromptData is passed to prompt_pre_pr_review_pass1.md.
type PrePRReviewPass1PromptData struct {
	IntegrationBranch string
	IntegrationBase   string
	IssueNumbers      []int
	IssuesMarkdown    string // gh-fetched bodies + headings
	CommitHistory     string
	GitDiff           string
	LintTests         string
	UnitTests         string
	NonInteractive    bool
	PriorPassArtifactsMarkdown string // full serialized prior pass outputs (GM-007)
	BonusSteeringNote          string // 4th cycle; interactive note or non-interactive best-judgment line
	Round                      int    // 1-based pass number for this Pass 1 invocation
}

// PrePRReviewPass2PromptData is passed to prompt_pre_pr_review_pass2.md.
type PrePRReviewPass2PromptData struct {
	IntegrationBranch string
	IntegrationBase   string
	IssueNumbers      []int
	IssuesMarkdown    string
	CommitHistory     string
	GitDiff           string
	LintTests         string
	UnitTests         string
	NonInteractive    bool
	PriorPassArtifactsMarkdown string
	BonusSteeringNote          string
	Round                      int
	CurrentPass1MatrixJSON     string // indented JSON from validated matrix this round
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
