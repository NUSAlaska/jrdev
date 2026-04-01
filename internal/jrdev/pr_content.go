package jrdev

import (
	"fmt"
	"strings"
)

const prBaseBranch = "main"

// PullRequestTitleAndBody runs the PR-description agent in the integration worktree, or returns
// fallbacks if the prompt is empty, the agent fails, or output cannot be parsed.
func PullRequestTitleAndBody(cfg Config, agent AgentRunner, log func(string, ...any), prompts PromptBundle, intPath, integrationBranch string) (title, body string) {
	fallbackTitle := fmt.Sprintf("jrdev: agent-queue integration %s", integrationBranch)
	fallbackBody := fmt.Sprintf("Automated integration branch %q.\n\nLabel %q was processed by jrdev.", integrationBranch, cfg.Label)
	if log == nil {
		log = func(string, ...any) {}
	}
	if strings.TrimSpace(prompts.PR) == "" {
		return fallbackTitle, fallbackBody
	}

	render := func() (string, error) {
		hist, err := CommitHistoryForPrompt(intPath)
		if err != nil {
			return "", err
		}
		diff, err := GitDiffForPrompt(intPath)
		if err != nil {
			return "", err
		}
		return Render("pr", prompts.PR, PRPromptData{
			IntegrationBranch: integrationBranch,
			QueueLabel:        cfg.Label,
			PRBase:            prBaseBranch,
			CommitHistory:     hist,
			GitDiff:           diff,
		})
	}

	vlog(cfg, log, "jrdev: verbose: pr description — agent run in %s\n", intPath)
	out, err := runAgentUntilComplete(cfg, agent, log, "pr", intPath, render)
	if err != nil {
		log("jrdev: warning: PR description agent failed (%v); using default title/body\n", err)
		return fallbackTitle, fallbackBody
	}
	meta, err := ParsePRMetadata(out)
	if err != nil {
		log("jrdev: warning: could not parse PR title/body (%v); using default\n", err)
		return fallbackTitle, fallbackBody
	}
	return meta.Title, meta.Body
}
