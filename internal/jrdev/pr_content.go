package jrdev

import (
	"fmt"
	"strings"
)

// PullRequestTitleAndBody runs the PR-description agent in the integration worktree, or returns
// fallbacks if the prompt is empty, the agent fails, or output cannot be parsed.
// If omitOptionalPrePRHandoffWarn is true, the informational warning about a missing `.jrdev/pre-pr-review/latest`
// is not logged (e.g. pipeline skipped pre-pr-review because nothing closed an issue via commit refs).
func PullRequestTitleAndBody(cfg Config, agent AgentRunner, log func(string, ...any), prompts PromptBundle, intPath, integrationBranch string, omitOptionalPrePRHandoffWarn bool) (title, body string) {
	fallbackTitle := fmt.Sprintf("jrdev: agent-queue integration %s", integrationBranch)
	fallbackBody := fmt.Sprintf("Automated integration branch %q.\n\nLabel %q was processed by jrdev.", integrationBranch, cfg.Label)
	if log == nil {
		log = func(string, ...any) {}
	}
	prBase := strings.TrimSpace(cfg.PRBase)
	if prBase == "" {
		prBase = DefaultPRBaseBranch
	}
	if strings.TrimSpace(prompts.PR) == "" {
		return fallbackTitle, fallbackBody
	}

	render := func() (string, error) {
		hist, err := CommitHistoryForPrompt(intPath)
		if err != nil {
			return "", err
		}
		diff, err := GitDiffForPromptFromBase(intPath, prBase)
		if err != nil {
			return "", err
		}
		handoff, handoffWarn := LoadPRPromptPrePRReviewHandoff(intPath)
		if handoffWarn != "" && !(omitOptionalPrePRHandoffWarn && strings.Contains(handoffWarn, "no .jrdev/pre-pr-review/latest")) {
			log("jrdev: warning: %s\n", handoffWarn)
		}
		return Render("pr", prompts.PR, PRPromptData{
			IntegrationBranch:         integrationBranch,
			QueueLabel:                cfg.Label,
			PRBase:                    prBase,
			CommitHistory:             hist,
			GitDiff:                   diff,
			PrePRReviewHandoffPresent: handoff.Present,
			PrePRReviewHandoffSummary: handoff.Summary,
			PrePRReviewArtifactPaths:  handoff.ArtifactPaths,
		})
	}

	vlog(cfg, log, "jrdev: verbose: pr description — agent run in %s\n", intPath)
	out, err := runAgentUntilComplete(cfg, agent, log, "pr", intPath, render, true)
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
