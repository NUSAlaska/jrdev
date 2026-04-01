package jrdev

import (
	"fmt"
	"path/filepath"
)

// Run executes the full agent-queue loop after preflight.
func Run(cfg Config, prompts PromptBundle, agent AgentRunner, log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
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

	if err := RunPreflight(cfg, agent); err != nil {
		return err
	}

	if cfg.DryRun {
		log("jrdev: --dry-run stops after preflight (no agent smoke, no git/orchestration).\n")
		return nil
	}

	workRoot := filepath.Join(cfg.RepoRoot, cfg.Worktrees)
	if err := EnsureWorktreesRoot(workRoot); err != nil {
		return err
	}

	git := GitOps{RepoRoot: cfg.RepoRoot}
	if cfg.Verbose && log != nil {
		git.Log = log
	}

	if err := git.FetchOrigin(); err != nil {
		return err
	}
	base := cfg.Integration
	if base == "" {
		base = "origin/main"
	}

	integrationBranch, intPath, err := git.CreateIntegrationBranchAndWorktree(workRoot, base)
	if err != nil {
		return err
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
		planBody, err := Render("plan", prompts.Plan, PlanPromptData{
			QueueLabel: cfg.Label,
			IssuesJSON: issuesJSON,
		})
		if err != nil {
			return err
		}
		planOut, err := agent.Run(cfg, intPath, planBody, AgentRunOptions{Print: true})
		if err != nil {
			return fmt.Errorf("plan phase: %w", err)
		}
		doc, err := ParsePlan(planOut)
		if err != nil {
			return fmt.Errorf("plan parse: %w", err)
		}
		if len(doc.Issues) == 0 {
			log("jrdev: planner returned no issues — stopping.\n")
			break
		}
		job := doc.Issues[0]

		issueSlug := IssueSlug(job.Title)
		expBranch := fmt.Sprintf("agent-queue/issue-%d-%s", job.Number, issueSlug)
		if job.Branch != expBranch {
			log("jrdev: warning: plan branch %q differs from conventional %q (using plan branch).\n", job.Branch, expBranch)
		}

		body, err := IssueBody(cfg, job.Number)
		if err != nil {
			return err
		}

		issuePath := IssueWorkPath(workRoot, job.Number, issueSlug)
		baseSHA, err := git.CreateIssueWorktree(issuePath, integrationBranch, job.Branch)
		if err != nil {
			return err
		}

		implData := ImplementPromptData{
			IssueNumber:       job.Number,
			IssueTitle:        job.Title,
			IssueBody:         body,
			IssueBranch:       job.Branch,
			IntegrationBranch: integrationBranch,
			QueueLabel:        cfg.Label,
		}
		implBody, err := Render("implement", prompts.Implement, implData)
		if err != nil {
			return err
		}
		if _, err := agent.Run(cfg, issuePath, implBody, AgentRunOptions{Print: true}); err != nil {
			return fmt.Errorf("implement: %w", err)
		}
		commits, err := CommitCountAhead(issuePath, baseSHA)
		if err != nil {
			return err
		}
		if commits == 0 {
			log("jrdev: zero commits after implement — retrying once.\n")
			if _, err := agent.Run(cfg, issuePath, implBody, AgentRunOptions{Print: true}); err != nil {
				return fmt.Errorf("implement retry: %w", err)
			}
			commits, err = CommitCountAhead(issuePath, baseSHA)
			if err != nil {
				return err
			}
		}
		if commits == 0 {
			return fmt.Errorf("jrdev: issue #%d produced zero commits after retry — aborting (no merge, no gh close)", job.Number)
		}

		if commits > 0 {
			revBody, err := Render("review", prompts.Review, implData)
			if err != nil {
				return err
			}
			if _, err := agent.Run(cfg, issuePath, revBody, AgentRunOptions{Print: true}); err != nil {
				return fmt.Errorf("review: %w", err)
			}
		}

		mergeBody, err := Render("merge", prompts.Merge, MergePromptData{
			IssueNumber:       job.Number,
			IssueTitle:        job.Title,
			IssueBranch:       job.Branch,
			IntegrationBranch: integrationBranch,
			QueueLabel:        cfg.Label,
		})
		if err != nil {
			return err
		}
		if _, err := agent.Run(cfg, intPath, mergeBody, AgentRunOptions{Print: true}); err != nil {
			return fmt.Errorf("merge phase: %w", err)
		}
		merged, err := BranchMergedIntoHead(intPath, job.Branch)
		if err != nil {
			return err
		}
		if !merged {
			if err := MergeBranchInDir(intPath, job.Branch); err != nil {
				return fmt.Errorf("merge %s into integration: %w", job.Branch, err)
			}
		}
		if err := GoVetTest(intPath); err != nil {
			return fmt.Errorf("post-merge quality gate: %w", err)
		}

		comment := fmt.Sprintf("Merged into integration branch %s via jrdev.", integrationBranch)
		if err := CloseIssue(cfg, job.Number, comment); err != nil {
			return err
		}
		if err := RemoveQueueLabel(cfg, job.Number); err != nil {
			return err
		}

		// Push integration branch for PR
		if err := git.git("push", "-u", "origin", integrationBranch); err != nil {
			log("jrdev: warning: git push integration: %v\n", err)
		}

		_ = git.RemoveWorktree(issuePath)
	}

	if iter >= maxIter {
		log("jrdev: stopped: hit maxIterations=%d\n", maxIter)
	}

	if !cfg.SkipPR {
		title := fmt.Sprintf("jrdev: agent-queue integration %s", integrationBranch)
		body := fmt.Sprintf("Automated integration branch %q.\n\nLabel %q was processed by jrdev.", integrationBranch, cfg.Label)
		if err := CreatePullRequest(cfg, title, body, integrationBranch); err != nil {
			return err
		}
	}

	return nil
}
