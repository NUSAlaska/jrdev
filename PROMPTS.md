# jrdev prompt templates — substitution reference

Templates are `text/template` files embedded in the binary. Placeholders must match `internal/jrdev/render.go` field names.

## Shared (`prompt_*.md`)

Plain text; only the fields listed per file are substituted.

## `prompt_plan.md` — `PlanPromptData`

| Placeholder     | Type   | Meaning                                                |
|----------------|--------|--------------------------------------------------------|
| `{{.QueueLabel}}` | string | GitHub label for the queue (e.g. `agent-queue`) |
| `{{.IssuesJSON}}` | string | Pretty-printed JSON array of open queued issues (`gh` preloaded). |

## `prompt_implement.md` — `ImplementPromptData`

| Placeholder            | Type   | Meaning                          |
|------------------------|--------|----------------------------------|
| `{{.IssueNumber}}`     | int    | GitHub issue number               |
| `{{.IssueTitle}}`      | string | Issue title                       |
| `{{.IssueBody}}`       | string | Issue body text                   |
| `{{.IssueBranch}}`     | string | Branch from planner (`agent-queue/issue-…`) |
| `{{.IntegrationBranch}}` | string | Current integration branch name |
| `{{.QueueLabel}}`      | string | Queue label                       |

## `prompt_review.md`

Same fields as **implement** (`ImplementPromptData` / `ReviewPromptData`).

## `prompt_merge.md` — `MergePromptData`

| Placeholder             | Type   | Meaning                    |
|-------------------------|--------|----------------------------|
| `{{.IssueNumber}}`      | int    | GitHub issue number         |
| `{{.IssueTitle}}`       | string | Issue title                 |
| `{{.IssueBranch}}`      | string | Issue branch to merge       |
| `{{.IntegrationBranch}}`| string | Integration branch (merge target) |
| `{{.QueueLabel}}`       | string | Label `gh issue edit --remove-label` refers to |

## Not templated

The preflight **agent smoke** prompt is hardcoded in `internal/jrdev/preflight.go` (`AgentSmokePrompt`).

**Cursor agent environment:** Each **`agent`** subprocess can receive **`CURSOR_CONFIG_DIR`** (permissions / `cli-config.json`) from repo defaults, flags, or **`jrdev-agent-permissions.json`**. See **`README.md`** (“Cursor agent CLI permissions”) and the implementation in `internal/jrdev/agent.go` and `internal/jrdev/agent_cursor_config.go`.
