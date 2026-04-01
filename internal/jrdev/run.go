package jrdev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxAgentStepAttempts = 25

// Run executes the full agent-queue loop after preflight.
func Run(cfg Config, prompts PromptBundle, agent AgentRunner, log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
	}
	vlog(cfg, log, "jrdev: verbose: repo root=%q label=%q dryRun=%v skipPR=%v integration=%q\n",
		cfg.RepoRoot, cfg.Label, cfg.DryRun, cfg.SkipPR, cfg.Integration)
	agentBin := cfg.AgentBin
	if agentBin == "" {
		agentBin = "agent"
	}
	vlog(cfg, log, "jrdev: verbose: agent=%q model=%q gh=%q worktrees=%q\n",
		agentBin, cfg.AgentModel, cfg.GhBin, cfg.Worktrees)
	switch {
	case cfg.AgentCursorConfigDir != "":
		vlog(cfg, log, "jrdev: verbose: agent CURSOR_CONFIG_DIR=%q (cli-config.json)\n", cfg.AgentCursorConfigDir)
	case cfg.AgentPermissionsFile != "":
		vlog(cfg, log, "jrdev: verbose: agent permissions file=%q (temp cli-config.json per agent run)\n", cfg.AgentPermissionsFile)
	}

	n, err := QueueOpenCount(cfg)
	if err != nil {
		return err
	}
	if n == 0 {
		log("jrdev: no open issues labeled %q — nothing to do.\n", cfg.Label)
		return nil
	}
	maxIter := EffectiveMaxIterations(n, cfg.MaxIters)
	log("jrdev: N=%d open queued issues; maxIterations=%d\n", n, maxIter)
	vlog(cfg, log, "jrdev: verbose: maxIterations raw config=%d effective=%d (default 2N+3 when 0)\n", cfg.MaxIters, maxIter)

	if err := RunPreflight(cfg, agent, log); err != nil {
		return err
	}
	vlog(cfg, log, "jrdev: verbose: preflight finished\n")

	if cfg.DryRun {
		log("jrdev: --dry-run stops after preflight (no agent smoke, no git/orchestration).\n")
		return nil
	}

	workRoot := filepath.Join(cfg.RepoRoot, cfg.Worktrees)
	vlog(cfg, log, "jrdev: verbose: worktrees root=%q\n", workRoot)
	if err := EnsureWorktreesRoot(workRoot); err != nil {
		return err
	}

	git := GitOps{RepoRoot: cfg.RepoRoot}
	if cfg.Verbose && log != nil {
		git.Log = log
	}

	vlog(cfg, log, "jrdev: verbose: git fetch origin (repo=%s)\n", cfg.RepoRoot)
	if err := git.FetchOrigin(); err != nil {
		return err
	}

	workRootAbs, err := filepath.Abs(workRoot)
	if err != nil {
		return err
	}

	if cfg.FreshStart {
		vlog(cfg, log, "jrdev: verbose: --fresh: clearing jrdev worktrees under %q and agent-queue branches\n", workRootAbs)
		if err := git.CleanupJrdevWorkstate(workRootAbs); err != nil {
			return err
		}
	}

	var integrationBranch, intPath string
	if !cfg.FreshStart {
		if b, p, ok, err := git.FindResumableIntegrationWorktree(workRootAbs); err != nil {
			return err
		} else if ok {
			resume := true
			if StdinIsInteractive() {
				resume, err = PromptResumeOrCleanIntegration(os.Stdin, os.Stderr, b, p)
				if err != nil {
					return err
				}
			} else {
				log("jrdev: resuming existing integration %q (non-interactive stdin; use --fresh for a clean run)\n", b)
			}
			if resume {
				integrationBranch, intPath = b, p
				vlog(cfg, log, "jrdev: verbose: resuming integration worktree %q\n", intPath)
			} else {
				vlog(cfg, log, "jrdev: verbose: fresh start selected — cleaning jrdev workstate\n")
				if err := git.CleanupJrdevWorkstate(workRootAbs); err != nil {
					return err
				}
			}
		}
	}

	base := cfg.Integration
	if base == "" {
		base = "origin/main"
	}
	if integrationBranch == "" {
		vlog(cfg, log, "jrdev: verbose: creating integration branch from base %q\n", base)
		integrationBranch, intPath, err = git.CreateIntegrationBranchAndWorktree(workRoot, base)
		if err != nil {
			return err
		}
	}
	log("jrdev: integration branch %s → worktree %s\n", integrationBranch, intPath)

	iter := 0
	for iter < maxIter {
		iter++
		log("jrdev: cycle %d/%d\n", iter, maxIter)

		issuesJSON, err := IssuesListJSON(cfg)
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: gh issue list JSON for planner: %d bytes\n", len(issuesJSON))
		planHist, err := CommitHistoryForPrompt(intPath)
		if err != nil {
			return err
		}
		planDiff, err := GitDiffForPrompt(intPath)
		if err != nil {
			return err
		}
		planBody, err := Render("plan", prompts.Plan, PlanPromptData{
			QueueLabel:    cfg.Label,
			IssuesJSON:    issuesJSON,
			CommitHistory: planHist,
			GitDiff:       planDiff,
		})
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: plan phase — rendered prompt %d bytes (worktree=%s)\n", len(planBody), intPath)
		planOut, err := agent.Run(cfg, intPath, planBody, AgentRunOptions{Print: true})
		if err != nil {
			return fmt.Errorf("plan phase: %w", err)
		}
		vlog(cfg, log, "jrdev: verbose: plan phase — agent output %d bytes\n", len(planOut))
		doc, err := ParsePlan(planOut)
		if err != nil {
			vlog(cfg, log, "jrdev: verbose: plan parse failed (%v); retrying once with correction\n", err)
			retryPrompt := AppendAgentOutputRetryInstructions(planBody, "Plan phase", err, planOut)
			planOut, err = agent.Run(cfg, intPath, retryPrompt, AgentRunOptions{Print: true})
			if err != nil {
				return fmt.Errorf("plan phase retry: %w", err)
			}
			vlog(cfg, log, "jrdev: verbose: plan phase — retry agent output %d bytes\n", len(planOut))
			doc, err = ParsePlan(planOut)
			if err != nil {
				return fmt.Errorf("plan parse: %w", err)
			}
		}
		if len(doc.Issues) == 0 {
			log("jrdev: planner returned no issues — stopping.\n")
			break
		}
		job := doc.Issues[0]
		vlog(cfg, log, "jrdev: verbose: plan — first job issue #%d branch=%q title=%q\n", job.Number, job.Branch, job.Title)

		issueSlug := IssueSlug(job.Title)
		expBranch := fmt.Sprintf("agent-queue/issue-%d-%s", job.Number, issueSlug)
		if job.Branch != expBranch {
			log("jrdev: warning: plan branch %q differs from conventional %q (using plan branch).\n", job.Branch, expBranch)
		}

		vlog(cfg, log, "jrdev: verbose: gh issue view %d (body for implement prompt)\n", job.Number)
		body, err := IssueBody(cfg, job.Number)
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: issue #%d body length %d runes\n", job.Number, len([]rune(body)))

		issuePath := IssueWorkPath(workRoot, job.Number, issueSlug)
		vlog(cfg, log, "jrdev: verbose: create issue worktree path=%q from integration %q branch %q\n", issuePath, integrationBranch, job.Branch)
		baseSHA, err := git.CreateIssueWorktree(issuePath, integrationBranch, job.Branch)
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: issue worktree base SHA %s\n", baseSHA)

		proj := cfg.Project
		implData := ImplementPromptData{
			IssueNumber:       job.Number,
			IssueTitle:        job.Title,
			IssueBody:         body,
			IssueBranch:       job.Branch,
			IntegrationBranch: integrationBranch,
			QueueLabel:        cfg.Label,
			LintTests:         PromptLintTests(proj),
			UnitTests:         PromptUnitTests(proj),
		}
		renderImplement := func() (string, error) {
			var err error
			implData.CommitHistory, err = CommitHistoryForPrompt(issuePath)
			if err != nil {
				return "", err
			}
			implData.GitDiff, err = GitDiffForPrompt(issuePath)
			if err != nil {
				return "", err
			}
			return Render("implement", prompts.Implement, implData)
		}
		implOut, err := runAgentUntilComplete(cfg, agent, log, "implement", issuePath, renderImplement)
		if err != nil {
			return err
		}
		commits, err := CommitCountAhead(issuePath, baseSHA)
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: commits ahead of base after implement: %d\n", commits)
		noCommitOK := strings.Contains(implOut, AgentImplementNoCommitToken)
		if commits == 0 && noCommitOK {
			vlog(cfg, log, "jrdev: verbose: implement output contained %q — not requiring commits\n", AgentImplementNoCommitToken)
			log("jrdev: implement reported %q — continuing without commits on issue branch.\n", AgentImplementNoCommitToken)
		}
		if commits == 0 && !noCommitOK {
			log("jrdev: zero commits after implement — retrying implement phase.\n")
			implOut, err = runAgentUntilComplete(cfg, agent, log, "implement", issuePath, renderImplement)
			if err != nil {
				return err
			}
			commits, err = CommitCountAhead(issuePath, baseSHA)
			if err != nil {
				return err
			}
			noCommitOK = strings.Contains(implOut, AgentImplementNoCommitToken)
			vlog(cfg, log, "jrdev: verbose: commits ahead after implement retry: %d\n", commits)
			if commits == 0 && noCommitOK {
				vlog(cfg, log, "jrdev: verbose: implement retry output contained %q — not requiring commits\n", AgentImplementNoCommitToken)
				log("jrdev: implement reported %q — continuing without commits on issue branch.\n", AgentImplementNoCommitToken)
			}
		}
		if commits == 0 && !noCommitOK {
			return fmt.Errorf("jrdev: issue #%d produced zero commits after retry — aborting (no merge, no gh close)", job.Number)
		}

		if commits > 0 {
			renderReview := func() (string, error) {
				var err error
				implData.CommitHistory, err = CommitHistoryForPrompt(issuePath)
				if err != nil {
					return "", err
				}
				implData.GitDiff, err = GitDiffForPrompt(issuePath)
				if err != nil {
					return "", err
				}
				return Render("review", prompts.Review, implData)
			}
			if _, err := runAgentUntilComplete(cfg, agent, log, "review", issuePath, renderReview); err != nil {
				return err
			}
		}

		mergeData := MergePromptData{
			IssueNumber:       job.Number,
			IssueTitle:        job.Title,
			IssueBranch:       job.Branch,
			IntegrationBranch: integrationBranch,
			QueueLabel:        cfg.Label,
			LintTests:         PromptLintTests(proj),
			UnitTests:         PromptUnitTests(proj),
			IntegrationTests:  PromptIntegrationTests(proj),
		}
		renderMerge := func() (string, error) {
			var err error
			mergeData.CommitHistory, err = CommitHistoryForPrompt(intPath)
			if err != nil {
				return "", err
			}
			mergeData.GitDiff, err = GitDiffForPrompt(intPath)
			if err != nil {
				return "", err
			}
			return Render("merge", prompts.Merge, mergeData)
		}
		if _, err := runAgentUntilComplete(cfg, agent, log, "merge", intPath, renderMerge); err != nil {
			return fmt.Errorf("merge phase: %w", err)
		}
		merged, err := BranchMergedIntoHead(intPath, job.Branch)
		if err != nil {
			return err
		}
		vlog(cfg, log, "jrdev: verbose: branch %q already merged into integration HEAD: %v\n", job.Branch, merged)
		if !merged {
			vlog(cfg, log, "jrdev: verbose: git merge %q into integration at %s\n", job.Branch, intPath)
			if err := MergeBranchInDir(intPath, job.Branch); err != nil {
				return fmt.Errorf("merge %s into integration: %w", job.Branch, err)
			}
		}

		comment := fmt.Sprintf("Merged into integration branch %s via jrdev.", integrationBranch)
		vlog(cfg, log, "jrdev: verbose: gh issue close %d + remove label %q\n", job.Number, cfg.Label)
		if err := CloseIssue(cfg, job.Number, comment); err != nil {
			return err
		}
		if err := RemoveQueueLabel(cfg, job.Number); err != nil {
			return err
		}

		// Push integration branch for PR
		vlog(cfg, log, "jrdev: verbose: git push -u origin %q\n", integrationBranch)
		if err := git.PushUpstream(integrationBranch); err != nil {
			log("jrdev: warning: git push integration: %v\n", err)
		}
	}

	if iter >= maxIter {
		log("jrdev: stopped: hit maxIterations=%d\n", maxIter)
	}

	prCreated := false
	if !cfg.SkipPR {
		title, body := PullRequestTitleAndBody(cfg, agent, log, prompts, intPath, integrationBranch)
		vlog(cfg, log, "jrdev: verbose: gh pr create base=main head=%q title=%q\n", integrationBranch, title)
		if err := CreatePullRequest(cfg, title, body, integrationBranch); err != nil {
			return err
		}
		prCreated = true
	} else {
		vlog(cfg, log, "jrdev: verbose: skipping gh pr create (--skip-pr)\n")
	}

	if StdinIsInteractive() {
		doCleanup, err := PromptCleanupJrdevWorkstate(os.Stdin, os.Stderr, prCreated)
		if err != nil {
			return err
		}
		if doCleanup {
			vlog(cfg, log, "jrdev: verbose: cleaning jrdev worktrees under %q and agent-queue/* branches\n", workRootAbs)
			if err := git.CleanupJrdevWorkstate(workRootAbs); err != nil {
				return err
			}
			log("jrdev: removed jrdev worktrees and local agent-queue/* branches.\n")
		} else {
			log("jrdev: leaving worktrees and branches in place under %s (inspect before your next run).\n", workRoot)
		}
	} else {
		log("jrdev: non-interactive stdin — leaving worktrees and branches under %s in place (confirm cleanup when running in a terminal, or use --fresh to discard before a new run).\n", workRoot)
	}

	return nil
}

// runAgentUntilComplete invokes agent.Run with a fresh prompt from render on each attempt until stdout
// contains AgentPhaseCompleteToken or maxAgentStepAttempts is reached.
func runAgentUntilComplete(cfg Config, agent AgentRunner, log func(string, ...any), phase, workDir string, render func() (string, error)) (string, error) {
	var lastOut string
	for attempt := 1; attempt <= maxAgentStepAttempts; attempt++ {
		body, err := render()
		if err != nil {
			return "", err
		}
		vlog(cfg, log, "jrdev: verbose: %s — attempt %d/%d, prompt %d bytes\n", phase, attempt, maxAgentStepAttempts, len(body))
		out, err := agent.Run(cfg, workDir, body, AgentRunOptions{Print: true})
		lastOut = out
		if err != nil {
			return out, fmt.Errorf("%s: %w", phase, err)
		}
		if strings.Contains(out, AgentPhaseCompleteToken) {
			vlog(cfg, log, "jrdev: verbose: %s — saw %q in output (%d bytes)\n", phase, AgentPhaseCompleteToken, len(out))
			return out, nil
		}
		vlog(cfg, log, "jrdev: verbose: %s — output missing %q (%d bytes); retrying\n", phase, AgentPhaseCompleteToken, len(out))
	}
	return lastOut, fmt.Errorf("jrdev: %s: output never contained %q after %d attempts (last output %d bytes)",
		phase, AgentPhaseCompleteToken, maxAgentStepAttempts, len(lastOut))
}
