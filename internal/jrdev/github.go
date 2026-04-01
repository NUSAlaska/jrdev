package jrdev

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

func ghCmd(cfg Config, args ...string) *exec.Cmd {
	c := exec.Command(cfg.GhBin, args...)
	if cfg.RepoRoot != "" {
		c.Dir = cfg.RepoRoot
	}
	return c
}

// IssueRow is a minimal gh --json row for templates and planning.
type IssueRow struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// QueueOpenCount returns N = len(open issues with cfg.Label).
func QueueOpenCount(cfg Config) (int, error) {
	rows, err := listOpenIssues(cfg)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// IssuesListJSON returns pretty-printed JSON for plan template substitution.
func IssuesListJSON(cfg Config) (string, error) {
	rows, err := listOpenIssues(cfg)
	if err != nil {
		return "", err
	}
	meta, err := loadRepoWebMeta(cfg)
	if err != nil {
		return "", err
	}
	for i := range rows {
		rows[i].Body = expandGitHubLinksInIssueBody(meta, rows[i].Body)
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func listOpenIssues(cfg Config) ([]IssueRow, error) {
	out, err := ghCmd(cfg, "issue", "list",
		"--label", cfg.Label,
		"--state", "open",
		"--json", "number,title,body",
	).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %w\n%s", err, out)
	}
	var rows []IssueRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("gh issue list json: %w", err)
	}
	return rows, nil
}

// IssueBody fetches body text for one issue (best-effort for implement prompt).
// GitHub issue/PR shorthands and repo-relative markdown links are rewritten to absolute
// https URLs so the agent can open them (same host as gh repo view).
func IssueBody(cfg Config, number int) (string, error) {
	out, err := ghCmd(cfg, "issue", "view", fmt.Sprintf("%d", number), "--json", "body").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh issue view %d: %w\n%s", number, err, out)
	}
	var v struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return "", err
	}
	meta, err := loadRepoWebMeta(cfg)
	if err != nil {
		return "", fmt.Errorf("gh repo view (for issue link expansion): %w", err)
	}
	return expandGitHubLinksInIssueBody(meta, v.Body), nil
}

func loadRepoWebMeta(cfg Config) (*repoWebMeta, error) {
	out, err := ghCmd(cfg, "repo", "view", "--json", "url").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh repo view: %w\n%s", err, out)
	}
	var v struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("gh repo view json: %w", err)
	}
	if v.URL == "" {
		return nil, fmt.Errorf("gh repo view: empty url")
	}
	u, err := url.Parse(v.URL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("gh repo view: invalid url %q", v.URL)
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	return &repoWebMeta{base: u}, nil
}

// CloseIssue removes queue label and closes with comment (PRD: close when merged into integration).
func CloseIssue(cfg Config, number int, comment string) error {
	out, err := ghCmd(cfg, "issue", "close", fmt.Sprintf("%d", number), "--comment", comment).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue close %d: %w\n%s", number, err, out)
	}
	return nil
}

// RemoveQueueLabel removes cfg.Label from the issue without closing.
func RemoveQueueLabel(cfg Config, number int) error {
	out, err := ghCmd(cfg, "issue", "edit", fmt.Sprintf("%d", number), "--remove-label", cfg.Label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh issue edit --remove-label %d: %w\n%s", number, err, out)
	}
	return nil
}

// CreatePullRequest opens a PR from head branch to main.
func CreatePullRequest(cfg Config, title, body, headBranch string) error {
	args := []string{
		"pr", "create",
		"--base", "main",
		"--head", headBranch,
		"--title", title,
		"--body", body,
	}
	out, err := ghCmd(cfg, args...).CombinedOutput()
	if err != nil {
		// If PR already exists, gh often errors — surface output
		if strings.Contains(strings.ToLower(string(out)), "already") {
			return nil
		}
		return fmt.Errorf("gh pr create: %w\n%s", err, out)
	}
	return nil
}
