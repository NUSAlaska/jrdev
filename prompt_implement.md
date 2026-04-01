# jrdev — Implement phase

You are in the **issue git worktree** on the issue branch named below. **jrdev** created this worktree and branch from the **current integration tip**; do not recreate them unless fixing a documented failure.

## Sandbox

- **`main` is read-only** for this loop; do not merge or push to `main`.
- Do **not** close GitHub issues or remove the queue label here.

## Context

- Issue **#{{.IssueNumber}}** — {{.IssueTitle}}
- Issue branch (must match plan): **`{{.IssueBranch}}`**
- Integration branch name (for awareness): **`{{.IntegrationBranch}}`**
- Queue label: `{{.QueueLabel}}`

Here are the last 10 commits:
{{.CommitHistory}}

### Issue body

{{.IssueBody}}

### Linked issues and PRDs (private GitHub)

The body may include **full URLs** to other issues (for example a PRD). In **private** repositories, those pages are **not** readable via generic HTTP, **WebFetch**, or a browser without your GitHub session.

- Use the **GitHub CLI** from the **repository root** (same context **jrdev** uses), e.g. **`gh issue view <number> --json title,body`** or **`gh issue view <paste-the-issue-url>`**.

## Process

1. Explore the repo and gather relevent information that will allow you to complete the task. Pay extra attention to the test files that touch the relevent parts of the code.
2. If applicable, use RGR to complete the task
  a. RED: write one test
  b. GREEN: write the implementation to pass that test
  c. REPEAT: until done
  d. REFACTOR the code
3. Run **`go vet ./..`** and **`go test ./...`** (and focused tests you touch) before committing.
4. Make a git commit. The commit message must:
  a. Start with `JRDEV:` prefix
  b. Include task completed + PRD reference
  c. Key decisions made
  d. Files changed
  e. Blockers or notes for the next iteration

  keep it consise

## THE ISSUE

If the task is not complete, leave a comment on the GitHub issue with what was done.

Do not close the issue - this will be done later.

Once complete, output COMPLETE.

## FINAL RULES

ONLY WORK ON A SINGLE TASK.