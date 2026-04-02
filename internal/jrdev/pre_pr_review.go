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
	prePRReviewMaxCycles  = 3
	pass2HandoffFenceTag  = "jrdev-pre-pr-review-handoff"
	pass3ArtifactFenceTag = "jrdev-pre-pr-review-pass3"
	pass5HandoffFenceTag  = "jrdev-pre-pr-review-pass5-handoff"
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
	// Pass5BadTestByDesign and Pass5OperatorNotes may be merged after Pass 5 (optional fence in agent output).
	Pass5BadTestByDesign string `json:"pass5BadTestByDesign,omitempty"`
	Pass5OperatorNotes   string `json:"pass5OperatorNotes,omitempty"`
}

// PrePRPass5Handoff is optional JSON inside the pass-5 handoff fence (GM-010).
type PrePRPass5Handoff struct {
	BadTestByDesign string `json:"badTestByDesign"`
	OperatorNotes   string `json:"operatorNotes"`
}

type prePRPass4ArtifactAttempt struct {
	Attempt     int               `json:"attempt"`
	Success     bool              `json:"success"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Steps       []Pass4StepRecord `json:"steps"`
	FailedStep  *Pass4StepRecord  `json:"failedStep,omitempty"`
}

type prePRPass4ArtifactFile struct {
	Attempts             []prePRPass4ArtifactAttempt `json:"attempts"`
	FinalPass4Success    bool                        `json:"finalPass4Success"`
	Pass5BudgetExhausted bool                        `json:"pass5BudgetExhausted,omitempty"`
}

type prePRPass5RoundRecord struct {
	Round             int    `json:"round"`
	Fingerprint       string `json:"fingerprint"`
	FailingKind       string `json:"failingKind"`
	FailingCommand    string `json:"failingCommand"`
	HandoffFenceInner string `json:"pass5HandoffFenceInner,omitempty"`
	HandoffParseError string `json:"pass5HandoffParseError,omitempty"`
}

type prePRPass5ArtifactFile struct {
	Rounds []prePRPass5RoundRecord `json:"rounds"`
}

// PrePRPass3Artifact is the JSON inside the pass-3 artifact fence (GM-008).
type PrePRPass3Artifact struct {
	Summary             string `json:"summary"`
	TestDesign          string `json:"testDesign"`
	Coverage            string `json:"coverage"`
	PRDTestingAlignment string `json:"prdTestingAlignment"`
	StrictnessInference string `json:"strictnessInference"`
	FollowUps           string `json:"followUps"`
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

func parsePass3Artifact(agentOut string) (a PrePRPass3Artifact, rawInner string, parseErr string) {
	inner, err := ExtractSingleMarkdownFence(pass3ArtifactFenceTag, agentOut)
	if err != nil {
		return PrePRPass3Artifact{}, "", err.Error()
	}
	rawInner = inner
	var aa PrePRPass3Artifact
	if err := json.Unmarshal([]byte(inner), &aa); err != nil {
		return PrePRPass3Artifact{}, rawInner, err.Error()
	}
	return aa, rawInner, ""
}

func writePass3Artifact(artDir string, finalRound int, artifact PrePRPass3Artifact, rawFenceInner string, parseErr string) error {
	type wrap struct {
		FinalRound         int                `json:"finalRound"`
		Artifact           PrePRPass3Artifact `json:"artifact"`
		ArtifactRaw        string             `json:"artifactFenceInner,omitempty"`
		ArtifactParseError string             `json:"artifactParseError,omitempty"`
	}
	w := wrap{FinalRound: finalRound, Artifact: artifact, ArtifactRaw: rawFenceInner, ArtifactParseError: parseErr}
	if parseErr == "" {
		w.ArtifactRaw = ""
	}
	b, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artDir, "pass-3.json"), b, 0o644)
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

func readPass5BonusSteeringNote() (string, error) {
	if StdinIsInteractive() {
		fmt.Fprintf(os.Stderr, "jrdev: pre-pr-review: checks still failing after 3 Pass 5 rounds on the same fingerprint — enter a short steering note for one bonus Pass 5 round: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	return PrePRReviewPass5BonusCycleSteeringNonInteractive, nil
}

func parsePass5HandoffOptional(agentOut string) (h PrePRPass5Handoff, rawInner string, parseErr string) {
	if !strings.Contains(agentOut, pass5HandoffFenceTag) {
		return PrePRPass5Handoff{}, "", ""
	}
	inner, err := ExtractSingleMarkdownFence(pass5HandoffFenceTag, agentOut)
	if err != nil {
		return PrePRPass5Handoff{}, "", err.Error()
	}
	rawInner = inner
	var hh PrePRPass5Handoff
	if err := json.Unmarshal([]byte(inner), &hh); err != nil {
		return PrePRPass5Handoff{}, rawInner, err.Error()
	}
	return hh, rawInner, ""
}

func mergePass5HandoffIntoSessionFile(path string, h PrePRPass5Handoff) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sess PrePRSessionHandoff
	if err := json.Unmarshal(raw, &sess); err != nil {
		return fmt.Errorf("handoff.json: %w", err)
	}
	if t := strings.TrimSpace(h.BadTestByDesign); t != "" {
		sess.Pass5BadTestByDesign = t
	}
	if t := strings.TrimSpace(h.OperatorNotes); t != "" {
		sess.Pass5OperatorNotes = t
	}
	out, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func writePrePRPass4Artifact(artDir string, attempts []prePRPass4ArtifactAttempt, finalSuccess, budgetExhausted bool) error {
	wrap := prePRPass4ArtifactFile{
		Attempts:             attempts,
		FinalPass4Success:    finalSuccess,
		Pass5BudgetExhausted: budgetExhausted,
	}
	b, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artDir, "pass-4.json"), b, 0o644)
}

func writePrePRPass5Artifact(artDir string, rounds []prePRPass5RoundRecord) error {
	if rounds == nil {
		rounds = []prePRPass5RoundRecord{}
	}
	wrap := prePRPass5ArtifactFile{Rounds: rounds}
	b, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artDir, "pass-5.json"), b, 0o644)
}

func runPrePRPass4And5(cfg Config, prompts PromptBundle, agent AgentRunner, workDir, artDir, branch, integrationBase string, issueNums []int, agentPrint bool, log func(string, ...any)) error {
	if strings.TrimSpace(prompts.PrePRReviewPass5) == "" {
		return fmt.Errorf("jrdev: embedded pre-pr-review pass 5 prompt is empty")
	}
	handoffPath := filepath.Join(artDir, "handoff.json")
	plan := BuildPass4CheckPlan(cfg.Project)

	var pass4Attempts []prePRPass4ArtifactAttempt
	var pass5Rounds []prePRPass5RoundRecord

	activeFP := ""
	pass5RoundForFP := 0
	bonusNote := ""

	for attempt := 1; ; attempt++ {
		outcome := RunPass4Checks(workDir, plan)
		fp := ""
		if !outcome.Success && outcome.FailedStep != nil {
			fp = Pass4FailureFingerprint(outcome.FailedStep.Kind, outcome.FailedStep.Output)
		}

		arec := prePRPass4ArtifactAttempt{Attempt: attempt, Success: outcome.Success, Steps: outcome.Steps}
		if outcome.FailedStep != nil {
			fs := *outcome.FailedStep
			arec.FailedStep = &fs
			arec.Fingerprint = fp
		}
		pass4Attempts = append(pass4Attempts, arec)

		if outcome.Success {
			if err := writePrePRPass4Artifact(artDir, pass4Attempts, true, false); err != nil {
				return err
			}
			if err := writePrePRPass5Artifact(artDir, pass5Rounds); err != nil {
				return err
			}
			log("jrdev: pre-pr-review: Pass 4 checks passed\n")
			return nil
		}

		if outcome.FailedStep == nil {
			return fmt.Errorf("jrdev: pre-pr-review Pass 4: failure with no failed step recorded")
		}

		if fp != activeFP {
			activeFP = fp
			pass5RoundForFP = 0
			bonusNote = ""
		}
		pass5RoundForFP++
		if pass5RoundForFP > prePRPass5MaxRounds {
			log("jrdev: pre-pr-review: Pass 5 budget exhausted for this failure fingerprint (GM-010); leaving checks red (GM-014)\n")
			if err := writePrePRPass4Artifact(artDir, pass4Attempts, false, true); err != nil {
				return err
			}
			if err := writePrePRPass5Artifact(artDir, pass5Rounds); err != nil {
				return err
			}
			return nil
		}
		if pass5RoundForFP == prePRPass5MaxRounds {
			n, err := readPass5BonusSteeringNote()
			if err != nil {
				return err
			}
			bonusNote = n
		} else {
			bonusNote = ""
		}

		pass5Body, err := Render("pre-pr-review pass5", prompts.PrePRReviewPass5, PrePRReviewPass5PromptData{
			IntegrationBranch:   branch,
			IntegrationBase:     integrationBase,
			IssueNumbers:        issueNums,
			FailingKind:         string(outcome.FailedStep.Kind),
			FailingCommand:      outcome.FailedStep.Command,
			CapturedLogs:        outcome.FailedStep.Output,
			NonInteractive:      !StdinIsInteractive(),
			BonusSteeringNote:   bonusNote,
			Pass5Round:          pass5RoundForFP,
			Fingerprint:         fp,
		})
		if err != nil {
			return err
		}
		pass5Out, err := runAgentUntilComplete(cfg, agent, log, "pre-pr-review pass5", workDir, func() (string, error) {
			return pass5Body, nil
		}, agentPrint)
		if err != nil {
			return err
		}
		h, rawH, hErr := parsePass5HandoffOptional(pass5Out)
		p5rec := prePRPass5RoundRecord{
			Round:             pass5RoundForFP,
			Fingerprint:       fp,
			FailingKind:       string(outcome.FailedStep.Kind),
			FailingCommand:    outcome.FailedStep.Command,
			HandoffParseError: hErr,
		}
		if hErr != "" && rawH != "" {
			p5rec.HandoffFenceInner = rawH
		}
		pass5Rounds = append(pass5Rounds, p5rec)

		if hErr == "" && (strings.TrimSpace(h.BadTestByDesign) != "" || strings.TrimSpace(h.OperatorNotes) != "") {
			if err := mergePass5HandoffIntoSessionFile(handoffPath, h); err != nil {
				return err
			}
		}

		if err := writePrePRPass4Artifact(artDir, pass4Attempts, false, false); err != nil {
			return err
		}
		if err := writePrePRPass5Artifact(artDir, pass5Rounds); err != nil {
			return err
		}

		if err := RequireGitWorkingTreeClean(workDir); err != nil {
			return err
		}
		log("jrdev: pre-pr-review: Pass 5 round %d complete; re-running Pass 4\n", pass5RoundForFP)
	}
}

func writePass2RoundArtifact(artDir string, round int, handoff PrePRPass2Handoff, rawFenceInner string, parseErr string) error {
	type wrap struct {
		Round        int               `json:"round"`
		Handoff      PrePRPass2Handoff `json:"handoff"`
		HandoffRaw   string            `json:"handoffFenceInner,omitempty"`
		HandoffError string            `json:"handoffParseError,omitempty"`
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

// RunPrePrReview executes discovery, Pass 1 ↔ Pass 2 loops, Pass 3 test-design review, Pass 4 configured checks, and Pass 5 fix loop, with artifacts and session handoff (GM-007–GM-010, GM-012, GM-014).
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
	if strings.TrimSpace(prompts.PrePRReviewPass3) == "" {
		return fmt.Errorf("jrdev: embedded pre-pr-review pass 3 prompt is empty")
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
			LintTests:                  PromptLintTests(proj),
			UnitTests:                  PromptUnitTests(proj),
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
		// Prior rounds only; this round's matrix is CurrentPass1MatrixJSON (avoids duplicating round-N-pass-1.json).
		pass2Data := PrePRReviewPass2PromptData{
			IntegrationBranch:          branch,
			IntegrationBase:            integrationBase,
			IssueNumbers:               issueNums,
			IssuesMarkdown:             issuesMD,
			CommitHistory:              commitHist,
			GitDiff:                    diff,
			LintTests:                  PromptLintTests(proj),
			UnitTests:                  PromptUnitTests(proj),
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

	priorAll, err := formatPriorPassArtifacts(artDir, finalRound)
	if err != nil {
		return err
	}
	commitHist3, err := CommitHistoryForPrompt(workDir)
	if err != nil {
		return err
	}
	diff3, err := GitDiffForPromptFromBase(workDir, integrationBase)
	if err != nil {
		return err
	}
	pass3Data := PrePRReviewPass3PromptData{
		IntegrationBranch:          branch,
		IntegrationBase:            integrationBase,
		IssueNumbers:               issueNums,
		IssuesMarkdown:             issuesMD,
		CommitHistory:              commitHist3,
		GitDiff:                    diff3,
		NonInteractive:             !StdinIsInteractive(),
		PriorPassArtifactsMarkdown: priorAll,
		SessionHandoffJSON:         strings.TrimSpace(string(sb)),
		FinalRound:                 finalRound,
	}
	pass3Body, err := Render("pre-pr-review pass3", prompts.PrePRReviewPass3, pass3Data)
	if err != nil {
		return err
	}
	pass3Out, err := runAgentUntilComplete(cfg, agent, log, "pre-pr-review pass3", workDir, func() (string, error) {
		return pass3Body, nil
	}, agentPrint)
	if err != nil {
		return err
	}
	p3, raw3, p3err := parsePass3Artifact(pass3Out)
	if p3err != "" {
		log("jrdev: pre-pr-review: Pass 3 artifact: %s\n", p3err)
	}
	if err := writePass3Artifact(artDir, finalRound, p3, raw3, p3err); err != nil {
		return err
	}
	log("jrdev: pre-pr-review: Pass 3 artifact written (pass-3.json)\n")
	if err := RequireGitWorkingTreeClean(workDir); err != nil {
		return err
	}
	if err := runPrePRPass4And5(cfg, prompts, agent, workDir, artDir, branch, integrationBase, issueNums, agentPrint, log); err != nil {
		return err
	}
	return nil
}
