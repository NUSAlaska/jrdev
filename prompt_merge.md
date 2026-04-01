# jrdev — Merge phase

You are in the **integration git worktree** on **`{{.IntegrationBranch}}`**. **`main` is not modified** here; merges target this integration branch only.

## Repository guardrails

Same as other phases: **API-owned timestamps**, **`pkg/env`**, **migrations human-only**, **`/pkg/schema`**.

## Task

Merge issue branch **`{{.IssueBranch}}`** into the current integration branch (`HEAD`):

1. `git merge {{.IssueBranch}} --no-edit`
2. If there are conflicts, resolve them carefully, then complete the merge.
3. Run **`go vet ./...`** then **`go test ./...`** from the repo root of this worktree.
4. Do **not** push to **`main`**.

## GitHub (jrdev completes after you merge)

**jrdev** will run `gh issue close` and remove label `{{.QueueLabel}}` after a successful merge and quality gate. You may still mention the integration branch in comments if helpful.

## Completion marker

When merge, **`go vet ./...`**, and **`go test ./...`** have all **succeeded**, end your output with this exact line on its own:

<promise>COMPLETE</promise>

**jrdev** also verifies merge state and runs **`go vet` / `go test`** again after this phase.
