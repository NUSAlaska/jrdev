# jrdev — Merge phase

You are in the **integration git worktree** on **`{{.IntegrationBranch}}`**. **`main` is not modified** here; merges target this integration branch only.

## Task

Merge issue branch **`{{.IssueBranch}}`** into the current integration branch (`HEAD`):

1. `git merge {{.IssueBranch}} --no-edit`
2. If there are conflicts, resolve them carefully and inteligently by reading both sides and choosing the correct resolution.
3. After resolving conflicts run **`go vet ./...`** then **`go test ./...`** to verify everything works
4. If tests fail, fix the issues before proceeding.
5. Do **not** push to **`main`**.

## GitHub (jrdev completes after you merge)

**jrdev** will run `gh issue close` and remove label `{{.QueueLabel}}` after a successful merge and quality gate. You may still mention the integration branch in comments if helpful.

## Completion marker

When merge, **`go vet ./...`**, and **`go test ./...`** have all **succeeded**, end your output with this exact line on its own:

COMPLETE
