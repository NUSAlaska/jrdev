# jrdev — Pre-PR review — Pass 5 (fix checks from jrdev-captured logs only)

You are in the **integration git worktree** on branch **`{{.IntegrationBranch}}`**.

jrdev ran **Pass 4** (`lint` → `unit` → `integration` from `.jrdev/config.yaml`) and stopped at the **first failing command**. You receive **only the captured stdout/stderr below** — **do not** assume you must re-run the full suite inside this session (GM-010).

## Failure summary

- **Failing step kind**: `{{.FailingKind}}`
- **Command**: `{{.FailingCommand}}`
- **Stable fingerprint (for your awareness)**: `{{.Fingerprint}}`
- **Pass 5 round**: {{.Pass5Round}} of up to 4 (3 regular + 1 bonus on the same fingerprint)

### Captured output (stdout/stderr)

```
{{.CapturedLogs}}
```

## GM-010 — test and fix rules

- You **may** change **product code** and **lint** issues.
- **Test edits** are limited to **mechanical** fixes (imports, compile breaks, small helpers) and **snapshot/golden updates** when the **product** change is already **PRD-correct**.
- **Forbidden**: weakening assertions, mocks, skips, or tolerances **only** to make checks green.
- If the failure is a **bad test by design** (the test is wrong relative to the PRD, and “fixing” it would mean watering it down), **stop** test surgery. Do **not** force green by mutilating the test. Instead, record that in the handoff fence (below) so the human can handle it in the PR.

{{if .NonInteractive}}
## Non-interactive mode

Standard input is **not** a TTY. **Do not** wait for user input. Use best judgment; proceed deterministically.
{{end}}

{{if .BonusSteeringNote}}
## Bonus round steering note (from operator or jrdev default)

{{.BonusSteeringNote}}

{{end}}

## Task

1. Diagnose the failure using **only** the logs and by **reading relevant source/tests** in the repo.
2. Apply minimal, PRD-aligned fixes. Prefer product fixes over test hacks.
3. **`go vet` / `go test` / configured commands**: you **may** run narrow commands locally if needed to validate edits — jrdev does not require you to re-run the full pipeline.
4. Use commit messages prefixed with **`JRDEV: Pre-PR review pass 5 -`** for any commits.
5. Leave a **clean working tree** (commit meaningful work, or **no** changes). If you make **no** commits, end with **`COMPLETE NO COMMIT`** on its own line; otherwise end with **`COMPLETE`**.

### Optional handoff fence (JSON)

If you stopped because of a **bad test by design**, or you have **operator-facing notes** for the PR body, emit **at most one** markdown fence tagged **`jrdev-pre-pr-review-pass5-handoff`** containing a JSON object:

```jrdev-pre-pr-review-pass5-handoff
{
  "badTestByDesign": "",
  "operatorNotes": ""
}
```

Use **empty strings** when a field does not apply. jrdev merges non-empty values into `handoff.json`.

## Linked issues (scope)

{{range $i, $n := .IssueNumbers}}{{if $i}}, {{end}}#{{$n}}{{end}}

**Integration base**: `{{.IntegrationBase}}`
