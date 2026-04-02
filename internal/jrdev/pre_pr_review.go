package jrdev

import (
	"bufio"
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

const (
	prePRReviewMaxCycles = 3
	pass2HandoffFenceTag = "jrdev-pre-pr-review-handoff"
)

// PrePRPass2Handoff is the JSON inside the pass-2 handoff fence (GM-012 / session handoff).
type PrePRPass2Handoff struct {
	Round         int    `json:"round"`
	GapSummary    string `json:"gapSummary"`
	MatrixDelta   string `json:"matrixDelta"`
	DraftPRTitle  string `json:"draftPRTitle"`
	DraftPRBody   string `json:"draftPRBody"`
	GapNotes      string `json:"gapNotes"`
	ConflictNotes string `json:"conflictNotes"`
}

// PrePRSessionHandoff is written to handoff.json after Pass 1↔2 orchestration.
type PrePRSessionHandoff struct {
	DraftPRTitle  string `json:"draftPRTitle"`
	DraftPRBody   string `json:"draftPRBody"`
	GapNotes      string `json:"gapNotes"`
	ConflictNotes string `json:"conflictNotes"`
	FinalRound    int    `json:"finalRound"`
	MatrixHadGaps bool   `json:"matrixHadGapsAtEnd"`
}

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

func roundPass1Path(artDir string, round int) string {
	return filepath.Join(artDir, fmt.Sprintf("round-%02d-pass-1.json", round))
}

func roundPass2Path(artDir string, round int) string {
	return filepath.Join(artDir, fmt.Sprintf("round-%02d-pass-2.json", round))
}

func formatPriorPassArtifacts(artDir string, maxRound int) (string, error) {
	if maxRound < 1 {
		return "(none — first round)", nil
	}
	var sb strings.Builder
	for r := 1; r <= maxRound; r++ {
		p1 := roundPass1Path(artDir, r)
		b1, err := os.ReadFile(p1)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		fmt.Fprintf(&sb, "### Round %d — Pass 1 matrix (`round-%02d-pass-1.json`)\n\n```json\n%s\n```\n\n", r, r, strings.TrimSpace(string(b1)))
		p2 := roundPass2Path(artDir, r)
		b2, err := os.ReadFile(p2)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		fmt.Fprintf(&sb, "### Round %d — Pass 2 artifact (`round-%02d-pass-2.json`)\n\n```json\n%s\n```\n\n", r, r, strings.TrimSpace(string(b2)))
	}
	s := strings.TrimSpace(sb.String())
	if s == "" {
		return "(prior artifact files not found)", nil
	}
	return s, nil
}

func parsePass2Handoff(agentOut string) (h PrePRPass2Handoff, rawInner string, parseErr string) {
	inner, err := ExtractSingleMarkdownFence(pass2HandoffFenceTag, agentOut)
	if err != nil {
		return PrePRPass2Handoff{}, "", err.Error()
	}
	rawInner = inner
	var hh PrePRPass2Handoff
	if err := json.Unmarshal([]byte(inner), &hh); err != nil {
		return PrePRPass2Handoff{}, rawInner, err.Error()
	}
	return hh, rawInner, ""
}

func readBonusSteeringNote() (string, error) {
	if StdinIsInteractive() {
		fmt.Fprintf(os.Stderr, "jrdev: pre-pr-review: matrix still has gaps after %d cycles — enter a short steering note for one bonus Pass 1→2 cycle: ", prePRReviewMaxCycles)
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	return PrePRReviewBonusCycleSteeringNonInteractive, nil
}

func writePass2RoundArtifact(artDir string, round int, handoff PrePRPass2Handoff, rawFenceInner string, parseErr string) error {
	type wrap struct {
		Round         int             `json:"round"`
		Handoff       PrePRPass2Handoff `json:"handoff"`
		HandoffRaw    string          `json:"handoffFenceInner,omitempty"`
		HandoffError  string          `json:"handoffParseError,omitempty"`
	}
	w := wrap{Round: round, Handoff: handoff, HandoffRaw: rawFenceInner, HandoffError: parseErr}
	if parseErr == "" {
		w.HandoffRaw = ""
	}
	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(roundPass2Path(artDir, round), b, 0o644)
}

func mergeSessionHandoff(last PrePRPass2Handoff, finalRound int, matrixHadGaps bool) PrePRSessionHandoff {
	return PrePRSessionHandoff{
		DraftPRTitle:  strings.TrimSpace(last.DraftPRTitle),
		DraftPRBody:   strings.TrimSpace(last.DraftPRBody),
		GapNotes:      strings.TrimSpace(last.GapNotes),
		ConflictNotes: strings.TrimSpace(last.ConflictNotes),
		FinalRound:    finalRound,
		MatrixHadGaps: matrixHadGaps,
	}
}

// RunPrePrReview executes discovery, Pass 1 ↔ Pass 2 loops, artifacts, and session handoff (GM-007, GM-012, GM-014).
func RunPrePrReview(cfg Config, prompts PromptBundle, agent AgentRunner, workDir string, log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
	}
	if strings.TrimSpace(prompts.PrePRReviewPass1) == "" {
		return fmt.Errorf("jrdev: embedded pre-pr-review pass 1 prompt is empty")
	}
	if strings.TrimSpace(prompts.PrePRReviewPass2) == "" {
		return fmt.Errorf("jrdev: embedded pre-pr-review pass 2 prompt is empty")
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
	proj := cfg.Project

	runID, err := newPrePRRunID()
	if err != nil {
		return err
	}
	artDir := filepath.Join(workDir, AgentArtifactsDir, PrePrReviewArtifactsRoot, runID)
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		return err
	}
	latestPath := filepath.Join(workDir, AgentArtifactsDir, PrePrReviewArtifactsRoot, "latest")
	if err := os.WriteFile(latestPath, []byte(runID+"\n"), 0o644); err != nil {
		return err
	}

	agentPrint := !StdinIsInteractive()
	var lastHandoff PrePRPass2Handoff
	finalRound := 0

	execRound := func(round int, bonusSteering string) (PRDMatrixDoc, error) {
		priorMD, err := formatPriorPassArtifacts(artDir, round-1)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		commitHist, err := CommitHistoryForPrompt(workDir)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		diff, err := GitDiffForPromptFromBase(workDir, integrationBase)
		if err != nil {
			return PRDMatrixDoc{}, err
		}

		pass1Data := PrePRReviewPass1PromptData{
			IntegrationBranch:          branch,
			IntegrationBase:            integrationBase,
			IssueNumbers:               issueNums,
			IssuesMarkdown:             issuesMD,
			CommitHistory:              commitHist,
			GitDiff:                    diff,
			LintTests:                    PromptLintTests(proj),
			UnitTests:                    PromptUnitTests(proj),
			NonInteractive:             !StdinIsInteractive(),
			PriorPassArtifactsMarkdown: priorMD,
			BonusSteeringNote:          strings.TrimSpace(bonusSteering),
			Round:                      round,
		}
		pass1Body, err := Render("pre-pr-review pass1", prompts.PrePRReviewPass1, pass1Data)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		pass1Out, err := runAgentUntilComplete(cfg, agent, log, "pre-pr-review pass1", workDir, func() (string, error) {
			return pass1Body, nil
		}, agentPrint)
		if err != nil {
			return PRDMatrixDoc{}, err
		}

		inner, err := ExtractSinglePRDMatrixFence(pass1Out)
		if err != nil {
			return PRDMatrixDoc{}, fmt.Errorf("pre-pr-review pass1: %w", err)
		}
		doc, err := ValidatePRDMatrixJSON(inner)
		if err != nil {
			repairPrompt := PrePRMatrixRepairPrompt(inner, err.Error())
			repairOut, rerr := runAgentUntilComplete(cfg, agent, log, "pre-pr-review matrix repair", workDir, func() (string, error) {
				return repairPrompt, nil
			}, agentPrint)
			if rerr != nil {
				return PRDMatrixDoc{}, fmt.Errorf("pre-pr-review matrix repair: %w", rerr)
			}
			inner, err = ExtractSinglePRDMatrixFence(repairOut)
			if err != nil {
				return PRDMatrixDoc{}, fmt.Errorf("pre-pr-review matrix repair output: %w", err)
			}
			doc, err = ValidatePRDMatrixJSON(inner)
			if err != nil {
				return PRDMatrixDoc{}, fmt.Errorf("pre-pr-review matrix invalid after repair: %w", err)
			}
		}
		rawJSON := inner
		p1payload, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		if err := os.WriteFile(roundPass1Path(artDir, round), p1payload, 0o644); err != nil {
			return PRDMatrixDoc{}, err
		}
		if err := os.WriteFile(filepath.Join(artDir, "pass-1.json"), p1payload, 0o644); err != nil {
			return PRDMatrixDoc{}, err
		}
		if err := os.WriteFile(filepath.Join(artDir, "raw-matrix.json"), []byte(rawJSON), 0o644); err != nil {
			return PRDMatrixDoc{}, err
		}
		log("jrdev: pre-pr-review: round %d Pass 1 matrix written (runId=%s)\n", round, runID)

		matrixPretty := string(p1payload)
		p2prior, err := formatPriorPassArtifacts(artDir, round-1)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		// Pass 2 prompt includes current round's Pass 1 output in "prior" via CurrentPass1MatrixJSON only;
		// still pass prior completed rounds for full history (without duplicating current pass-1 file which is not written yet in prior formatter for this round - actually round-XX-pass-1 was just written; formatPriorPassArtifacts(artDir, round-1) excludes current. Good.)
		pass2Data := PrePRReviewPass2PromptData{
			IntegrationBranch:          branch,
			IntegrationBase:            integrationBase,
			IssueNumbers:               issueNums,
			IssuesMarkdown:             issuesMD,
			CommitHistory:              commitHist,
			GitDiff:                    diff,
			LintTests:                    PromptLintTests(proj),
			UnitTests:                    PromptUnitTests(proj),
			NonInteractive:             !StdinIsInteractive(),
			PriorPassArtifactsMarkdown: p2prior,
			BonusSteeringNote:          strings.TrimSpace(bonusSteering),
			Round:                      round,
			CurrentPass1MatrixJSON:     matrixPretty,
		}
		pass2Body, err := Render("pre-pr-review pass2", prompts.PrePRReviewPass2, pass2Data)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		pass2Out, err := runAgentUntilComplete(cfg, agent, log, "pre-pr-review pass2", workDir, func() (string, error) {
			return pass2Body, nil
		}, agentPrint)
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		h, rawInner, parseErr := parsePass2Handoff(pass2Out)
		if parseErr != "" {
			log("jrdev: pre-pr-review: round %d Pass 2 handoff: %s\n", round, parseErr)
		}
		if err := writePass2RoundArtifact(artDir, round, h, rawInner, parseErr); err != nil {
			return PRDMatrixDoc{}, err
		}
		p2bytes, err := os.ReadFile(roundPass2Path(artDir, round))
		if err != nil {
			return PRDMatrixDoc{}, err
		}
		if err := os.WriteFile(filepath.Join(artDir, "pass-2.json"), p2bytes, 0o644); err != nil {
			return PRDMatrixDoc{}, err
		}
		log("jrdev: pre-pr-review: round %d Pass 2 artifact written\n", round)

		lastHandoff = h
		finalRound = round

		if err := RequireGitWorkingTreeClean(workDir); err != nil {
			return PRDMatrixDoc{}, err
		}
		return doc, nil
	}

	var lastDoc PRDMatrixDoc
	for round := 1; round <= prePRReviewMaxCycles; round++ {
		var err error
		lastDoc, err = execRound(round, "")
		if err != nil {
			return err
		}
		if !PRDMatrixDocHasGaps(lastDoc) {
			goto finalize
		}
		log("jrdev: pre-pr-review: round %d — matrix still has gaps; continuing Pass 1→2 loop\n", round)
	}

	if PRDMatrixDocHasGaps(lastDoc) {
		note, err := readBonusSteeringNote()
		if err != nil {
			return err
		}
		log("jrdev: pre-pr-review: bonus Pass 1→2 cycle (round %d)\n", prePRReviewMaxCycles+1)
		lastDoc, err = execRound(prePRReviewMaxCycles+1, note)
		if err != nil {
			return err
		}
	}

finalize:
	session := mergeSessionHandoff(lastHandoff, finalRound, PRDMatrixDocHasGaps(lastDoc))
	sb, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(artDir, "handoff.json"), sb, 0o644); err != nil {
		return err
	}
	log("jrdev: pre-pr-review: session complete (runId=%s); wrote handoff.json and pass 1↔2 artifacts under %s\n", runID, artDir)
	return nil
}