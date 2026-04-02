package jrdev

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PrePrReviewArtifactsRoot is the path segment under .jrdev for this workflow.
const PrePrReviewArtifactsRoot = "pre-pr-review"

func newPrePRRunID() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// FormatIssuesMarkdownForPass1 fetches bodies via gh and builds markdown sections.
func FormatIssuesMarkdownForPass1(cfg Config, issueNums []int) (string, error) {
	var sb strings.Builder
	for _, n := range issueNums {
		body, err := IssueBody(cfg, n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, "#### Issue #%d\n\n%s\n\n", n, strings.TrimSpace(body))
	}
	return strings.TrimSpace(sb.String()), nil
}

// DiscoverPrePrReviewIssueNumbers returns the union (GM-002) of issue IDs from git log
// integrationBase..HEAD and from gh pr view closing references when a PR exists.
func DiscoverPrePrReviewIssueNumbers(cfg Config, workDir string) (nums []int, integrationBase string, err error) {
	workDir, err = filepath.Abs(workDir)
	if err != nil {
		return nil, "", err
	}
	base := strings.TrimSpace(cfg.Integration)
	if base == "" {
		base = "origin/main"
	}
	fromLog, err := ParseIssueRefsFromGitLogRange(workDir, base)
	if err != nil {
		return nil, base, err
	}
	headBranch, err := GitCurrentBranch(workDir)
	if err != nil {
		return nil, base, err
	}
	fromPR, err := PRClosingIssueNumbersFromGH(cfg, headBranch)
	if err != nil {
		return nil, base, err
	}
	return UnionSortedInts(fromLog, fromPR), base, nil
}

// RunPrePrReview executes discovery, Pass 1 agent, matrix validation/repair, and artifacts (GM-013–GM-014).
func RunPrePrReview(cfg Config, prompts PromptBundle, agent AgentRunner, workDir string, log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
	}
	if strings.TrimSpace(prompts.PrePRReviewPass1) == "" {
		return fmt.Errorf("jrdev: embedded pre-pr-review pass 1 prompt is empty")
	}
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return err
	}

	git := GitOps{RepoRoot: cfg.RepoRoot}
	if cfg.Verbose {
		git.Log = log
	}
	vlog(cfg, log, "jrdev: verbose: pre-pr-review — git fetch origin\n")
	if err := git.FetchOrigin(); err != nil {
		return err
	}

	issueNums, integrationBase, err := DiscoverPrePrReviewIssueNumbers(cfg, workDir)
	if err != nil {
		return err
	}
	if len(issueNums) == 0 {
		return fmt.Errorf("jrdev pre-pr-review: no linked GitHub issues found — add Fixes/Closes/Resolves #<n> to commits in %s..HEAD and/or ensure the branch has a GitHub PR with closing issue references", integrationBase)
	}
	log("jrdev: pre-pr-review: discovered issues %v (integration base %q)\n", issueNums, integrationBase)

	vlog(cfg, log, "jrdev: verbose: pre-pr-review — preflight\n")
	if err := RunPreflight(cfg, agent, log); err != nil {
		return err
	}

	branch, err := GitCurrentBranch(workDir)
	if err != nil {
		return err
	}
	issuesMD, err := FormatIssuesMarkdownForPass1(cfg, issueNums)
	if err != nil {
		return err
	}
	commitHist, err := CommitHistoryForPrompt(workDir)
	if err != nil {
		return err
	}
	diff, err := GitDiffForPromptFromBase(workDir, integrationBase)
	if err != nil {
		return err
	}
	proj := cfg.Project
	pass1Data := PrePRReviewPass1PromptData{
		IntegrationBranch: branch,
		IntegrationBase:   integrationBase,
		IssueNumbers:      issueNums,
		IssuesMarkdown:    issuesMD,
		CommitHistory:     commitHist,
		GitDiff:           diff,
		LintTests:         PromptLintTests(proj),
		UnitTests:         PromptUnitTests(proj),
		NonInteractive:    !StdinIsInteractive(),
	}
	pass1Body, err := Render("pre-pr-review pass1", prompts.PrePRReviewPass1, pass1Data)
	if err != nil {
		return err
	}

	agentPrint := !StdinIsInteractive() // GM-011: TTY uses interactive agent argv; headless uses -p
	pass1Out, err := runAgentUntilComplete(cfg, agent, log, "pre-pr-review pass1", workDir, func() (string, error) {
		return pass1Body, nil
	}, agentPrint)
	if err != nil {
		return err
	}

	inner, err := ExtractSinglePRDMatrixFence(pass1Out)
	if err != nil {
		return fmt.Errorf("pre-pr-review pass1: %w", err)
	}
	doc, err := ValidatePRDMatrixJSON(inner)
	if err != nil {
		repairPrompt := PrePRMatrixRepairPrompt(inner, err.Error())
		repairOut, rerr := runAgentUntilComplete(cfg, agent, log, "pre-pr-review matrix repair", workDir, func() (string, error) {
			return repairPrompt, nil
		}, agentPrint)
		if rerr != nil {
			return fmt.Errorf("pre-pr-review matrix repair: %w", rerr)
		}
		inner, err = ExtractSinglePRDMatrixFence(repairOut)
		if err != nil {
			return fmt.Errorf("pre-pr-review matrix repair output: %w", err)
		}
		doc, err = ValidatePRDMatrixJSON(inner)
		if err != nil {
			return fmt.Errorf("pre-pr-review matrix invalid after repair: %w", err)
		}
	}
	rawJSON := inner

	runID, err := newPrePRRunID()
	if err != nil {
		return err
	}
	artDir := filepath.Join(workDir, AgentArtifactsDir, PrePrReviewArtifactsRoot, runID)
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	p1path := filepath.Join(artDir, "pass-1.json")
	if err := os.WriteFile(p1path, payload, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artDir, "raw-matrix.json"), []byte(rawJSON), 0o644); err != nil {
		return err
	}
	latestPath := filepath.Join(workDir, AgentArtifactsDir, PrePrReviewArtifactsRoot, "latest")
	if err := os.WriteFile(latestPath, []byte(runID+"\n"), 0o644); err != nil {
		return err
	}
	log("jrdev: pre-pr-review: wrote %s Pass 1 artifacts (runId=%s)\n", p1path, runID)
	return nil
}
