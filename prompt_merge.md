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

## When integration checks fail (machine-readable signal)

If you cannot get the listed **integration** commands to pass after a honest attempt (and the merge itself is already done in this worktree), **jrdev** needs a deterministic signal:

- Print **exactly one line** that starts with `JRDEV_INTEGRATION_BLOCKED:` (prefix must be at the beginning of the line after normal trimming). You may add a short human-readable reason after the colon.
- You must still finish with **`COMPLETE`** on its own line as below so the merge phase is recognized as finished.

**Operator / automation behavior (for awareness):**

- **jrdev** will then either **abort** (issue stays open; integration worktree may be reset) or **continue the merge path** (waive re-running integration for this attempt — no extra jrdev subprocess runs integration tests). That choice comes from `--integration-blocked`, or `meta.integration_blocked_action` in `.jrdev/config.yaml` (`abort` or `merge`), or an interactive prompt when stdin is a TTY.
- Prefer fixing failures or adjusting the branch when possible; use the blocked line only when integration is still red after reasonable effort.

## GitHub (jrdev completes after you merge)

**jrdev** will run `gh issue close` and remove label `{{.QueueLabel}}` after a successful merge when you finish with **COMPLETE** below and **jrdev** does not abort for integration blocked. You may still mention the integration branch in comments if helpful.

## Completion marker

When the merge is complete and every listed **integration** command has succeeded (or there are no integration commands / only the no-checks notice applies), or when you emitted `JRDEV_INTEGRATION_BLOCKED:` as above, end your output with this exact line on its own:

COMPLETE
