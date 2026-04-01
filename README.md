# jrdev — agent-queue orchestrator

`jrdev` drives a **plan → implement → review → merge** loop for GitHub issues labeled **`agent-queue`** (configurable), using the **Cursor CLI `agent`** for intelligence and Go for git / `gh` orchestration.

## Prerequisites

- Git, **`gh`** authenticated for the repo, **Cursor `agent`** on `PATH` (or `--agent`).
- Run from the repo (or pass **`--repo`**). Git root is discovered by walking up for `.git`.

## Quick start

```text
# Count queue + preflight only (no agent smoke; stops before worktrees)
go run ./cmd/jrdev --dry-run

# Full run (preflight includes minimal `agent -p` smoke)
go run ./cmd/jrdev

# Override iteration cap (default is 2N+3 for N = open labeled issues at start)
go run ./cmd/jrdev --max-iterations 20
```

## Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--repo` | (walk up) | Git repository root |
| `--worktrees` | `.worktrees` | Directory under repo for worktrees (should be gitignored) |
| `--label` | `agent-queue` | Queue label |
| `--dry-run` | off | Skip agent smoke in preflight; exit before creating integration/issue worktrees |
| `--skip-pr` | off | Do not `gh pr create` when the loop finishes |
| `--max-iterations` | `2N+3` | Outer loop cap |
| `--integration-base` | `origin/main` | Base ref for new `agent-queue/run-…` branch |
| `--agent` | `agent` | Cursor agent binary |
| `--gh` | `gh` | GitHub CLI binary |
| `-v` | off | Verbose agent/subprocess logging |

## Behavior (summary)

1. **N** = count of open issues with the queue label; if **N == 0**, exit cleanly.
2. **Preflight** (once): `git`, `gh auth status`, `agent` present; unless `--dry-run`, a minimal non-destructive **`agent -p`** smoke.
3. Creates **`agent-queue/run-<timestamp>`** and a worktree under **`--worktrees`** from **`--integration-base`**.
4. Each **cycle**: plan (in integration worktree) → parse `<plan>…</plan>` JSON → **one** issue (first row) → issue worktree from integration tip → implement (retry once on zero commits) → review if commits → merge phase → **`go vet ./...`** and **`go test ./...`** on integration → **`gh issue close`** and remove label → push integration branch.
5. Stops when the plan returns **`issues: []`**, or **max iterations** is reached, then **`gh pr create`** to **`main`** unless **`--skip-pr`**.

`main` is never merged by the tool directly; landing on `main` is via PR only.

## Failure and recovery

- **Zero commits after implement retry**: run aborts; issue is not closed; integration branch and worktrees remain under `.worktrees/` for inspection.
- **Merge / `go vet` / `go test` failure**: fix locally in the integration or issue worktree, or remove worktrees/branches manually.

See `PROMPTS.md` for template placeholders. Design details: `plans/jrdev_agent_queue_orchestrator.plan.md`.
