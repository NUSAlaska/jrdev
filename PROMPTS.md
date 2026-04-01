# jrdev prompt templates — substitution reference

Templates are `text/template` files embedded in the binary. Placeholders must match `internal/jrdev/render.go` field names.

## Shared (`prompt_*.md`)

Plain text; only the fields listed per file are substituted.

## Agent phase completion (implement, review, merge)

For **implement**, **review**, and **merge**, jrdev runs the Cursor agent in a loop. On **each** attempt it re-reads git state and re-renders the prompt (fresh `{{.CommitHistory}}` and `{{.GitDiff}}`). It advances to the next pipeline step only when the agent’s stdout contains the substring **`COMPLETE`** (same check as `AgentPhaseCompleteToken` in `internal/jrdev/types.go`), up to **25** attempts per phase. **Plan** is not subject to this loop.

## `prompt_plan.md` — `PlanPromptData`

| Placeholder     | Type   | Meaning                                                |
|----------------|--------|--------------------------------------------------------|
| `{{.QueueLabel}}` | string | GitHub label for the queue (e.g. `agent-queue`) |
| `{{.IssuesJSON}}` | string | Pretty-printed JSON array of open queued issues (`gh` preloaded). |
| `{{.CommitHistory}}` | string | Last 10 commits from the **integration** worktree (`git log -n 10 --format="%H%n%ad%n%B---" --date=short`). |
| `{{.GitDiff}}` | string | `git diff main..HEAD` in the **integration** worktree. |

## `prompt_implement.md` — `ImplementPromptData`

| Placeholder            | Type   | Meaning                          |
|------------------------|--------|----------------------------------|
| `{{.IssueNumber}}`     | int    | GitHub issue number               |
| `{{.IssueTitle}}`      | string | Issue title                       |
| `{{.IssueBody}}`       | string | Issue body text (GitHub shorthands expanded to `https` URLs; agents on **private** repos should use **`gh issue view`** to load linked issues, not unauthenticated fetches) |
| `{{.IssueBranch}}`     | string | Branch from planner (`agent-queue/issue-…`) |
| `{{.IntegrationBranch}}` | string | Current integration branch name |
| `{{.QueueLabel}}`      | string | Queue label                       |
| `{{.CommitHistory}}`   | string | Last 10 commits from the **issue** worktree (same `git log` format as plan). |
| `{{.GitDiff}}`         | string | `git diff main..HEAD` in the **issue** worktree (refreshed before **each** implement attempt). |
| `{{.LintTests}}`       | string | Markdown: **lint** commands from the repo config (or empty-category / no-checks messaging). |
| `{{.UnitTests}}`       | string | Markdown: **unit** commands from the repo config (same). |

## `prompt_review.md`

Same fields as **implement** (`ImplementPromptData` / `ReviewPromptData`), including `{{.LintTests}}` and `{{.UnitTests}}`. **`{{.CommitHistory}}` and `{{.GitDiff}}` are refreshed before each review attempt** (after implement commits exist).

## `prompt_merge.md` — `MergePromptData`

| Placeholder             | Type   | Meaning                    |
|-------------------------|--------|----------------------------|
| `{{.IssueNumber}}`      | int    | GitHub issue number         |
| `{{.IssueTitle}}`       | string | Issue title                 |
| `{{.IssueBranch}}`      | string | Issue branch to merge       |
| `{{.IntegrationBranch}}`| string | Integration branch (merge target) |
| `{{.QueueLabel}}`       | string | Label `gh issue edit --remove-label` refers to |
| `{{.CommitHistory}}`    | string | Last 10 commits from the **integration** worktree (same `git log` format as plan). |
| `{{.GitDiff}}`          | string | `git diff main..HEAD` in the **integration** worktree. |
| `{{.LintTests}}`        | string | Markdown: **lint** commands (for parity with other phases; merge template may omit). |
| `{{.UnitTests}}`        | string | Markdown: **unit** commands (same). |
| `{{.IntegrationTests}}` | string | Markdown: **integration** commands from the repo config (primary check list for merge). |

## Not templated

The preflight **agent smoke** prompt is hardcoded in `internal/jrdev/preflight.go` (`AgentSmokePrompt`).

**Cursor agent environment:** Each **`agent`** subprocess can receive **`CURSOR_CONFIG_DIR`** (permissions / `cli-config.json`) from repo defaults, flags, or **`jrdev-agent-permissions.json`**. See **`README.md`** (“Cursor agent CLI permissions”) and the implementation in `internal/jrdev/agent.go` and `internal/jrdev/agent_cursor_config.go`.
