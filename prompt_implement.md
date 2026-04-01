# jrdev — Implement phase

You are in the **issue git worktree** on the issue branch named below. **jrdev** created this worktree and branch from the **current integration tip**; do not recreate them unless fixing a documented failure.

## Repository guardrails

- **API-owned write timestamps** — do not send client-authored `created_at` / `updated_at` on writes.
- **`pkg/env`** for environment reads where applicable (not raw `os.Getenv`).
- **Migrations**: human-only; do not run or generate migrations.
- Prefer **`pkg/schema`** types over ad-hoc duplicates.

## Sandbox

- **`main` is read-only** for this loop; do not merge or push to `main`.
- Do **not** close GitHub issues or remove the queue label here.
- Use **`gh` / `git`** only when necessary for implementation; avoid extra fetches if jrdev already updated refs.

## Context

- Issue **#{{.IssueNumber}}** — {{.IssueTitle}}
- Issue branch (must match plan): **`{{.IssueBranch}}`**
- Integration branch name (for awareness): **`{{.IntegrationBranch}}`**
- Queue label: `{{.QueueLabel}}`

### Issue body

{{.IssueBody}}

### Linked issues and PRDs (private GitHub)

The body may include **full URLs** to other issues (for example a PRD). In **private** repositories, those pages are **not** readable via generic HTTP, **WebFetch**, or a browser without your GitHub session.

- Use the **GitHub CLI** from the **repository root** (same context **jrdev** uses), e.g. **`gh issue view <number> --json title,body`** or **`gh issue view <paste-the-issue-url>`**.
- Ensure Cursor **agent permissions** allow **`gh`** (e.g. **`"gh"`** in **`jrdev-agent-permissions.json`** / **`Shell(gh)`** in **`cli-config.json`**). Otherwise **`gh`** will be blocked in headless mode.

## Process

1. Understand the issue and inspect the codebase in this worktree.
2. Implement the change with tests as appropriate for the repo.
3. Run **`go test ./...`** (and focused tests you touch) before committing.
4. Commit with **clear messages**. If work is only partial but safe, commit WIP with an explicit message rather than leaving **zero commits**.

## Retry policy

**jrdev** retries implement once if there are **zero commits**. Aim for at least one commit whenever any substantive progress is possible.
