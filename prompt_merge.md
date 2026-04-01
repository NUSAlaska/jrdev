# jrdev — Merge phase

You are in the **integration git worktree** on **`{{.IntegrationBranch}}`**. **`main` is not modified** here; merges target this integration branch only.

## Task

Merge issue branch **`{{.IssueBranch}}`** into the current integration branch (`HEAD`):

1. `git merge {{.IssueBranch}} --no-edit`
2. If there are conflicts, resolve them carefully and intelligently by reading both sides and choosing the correct resolution.
3. After the merge is clean, run the **integration** checks from the repository config below (each shell command in order). If a command fails, fix the issue before proceeding.
4. Do **not** push to **`main`**.

## Integration checks (`.jrdev/config.yaml`)

{{.IntegrationTests}}

## GitHub (jrdev completes after you merge)

**jrdev** will run `gh issue close` and remove label `{{.QueueLabel}}` after a successful merge when you finish with **COMPLETE** below. You may still mention the integration branch in comments if helpful.

## Completion marker

When the merge is complete and every listed **integration** command has succeeded (or there are no integration commands / only the no-checks notice applies), end your output with this exact line on its own:

COMPLETE
