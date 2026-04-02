# jrdev — Pre-PR review — Pass 1 (requirements matrix)

You are in the **integration git worktree** on branch **`{{.IntegrationBranch}}`**. This is **Pass 1 only**: build a structured requirements matrix from the linked issues; do **not** run later passes.

## Linked issues (scope)

These issue numbers were discovered from `git log {{.IntegrationBase}}..HEAD` (Fixes/Closes/Resolves #n) and, when applicable, from GitHub **closing references** on the PR for this branch:

{{range $i, $n := .IssueNumbers}}{{if $i}}, {{end}}#{{$n}}{{end}}

### Issue bodies (primary requirements substrate)

{{.IssuesMarkdown}}

### Linked issues and PRDs (private GitHub)

If a body links to other issues (for example a parent PRD) by URL or `#n`, read them with **`gh issue view`** from the repository root — not unauthenticated HTTP or **WebFetch**.

{{if .NonInteractive}}
## Non-interactive mode

Standard input is **not** a TTY. **Do not** wait for user input or ask the operator questions. Use best judgment; proceed deterministically.
{{end}}

{{if .BonusSteeringNote}}
## Operator steering (bonus Pass 1→2 cycle)

{{.BonusSteeringNote}}

{{end}}
{{if .PriorPassArtifactsMarkdown}}
## Prior pass artifacts (full history so far)

{{.PriorPassArtifactsMarkdown}}

{{end}}
## Repository context

Last commits in this worktree:

```
{{.CommitHistory}}
```

Diff ({{.IntegrationBase}}..HEAD):

```
{{.GitDiff}}
```

**Integration base** (same as jrdev `--integration-base`): `{{.IntegrationBase}}`

## Configured checks (repository `.jrdev/config.yaml`)

**Lint** (for awareness; you do not need to green the branch in Pass 1):

{{.LintTests}}

**Unit** (for awareness):

{{.UnitTests}}

## Task (GM-005 / grill-me matrix)

1. Treat each issue body as the **primary requirements substrate**: infer **atomic requirements** from headings, bullets, user stories, tables, checklists, or prose — even when wording is informal.
2. For **each** atomic requirement row:
   - **`id`**: stable identifier (e.g. `REQ-001`).
   - **`verbatimQuote`**: a **short verbatim quote** from the issue (or parent PRD if you followed a link) for traceability.
   - **`status`**: one of `satisfied`, `not_satisfied`, `unknown`, `conflict` — judged against **this integration branch** (code + tests in the repo).
   - **`evidence.paths`**: relevant file paths (repo-relative); use `[]` only when no paths apply.
   - **`evidence.tests`**: concrete test proof: repo path + test name or `go test -run=…` (or equivalent). For **`satisfied`**, every row must list **at least one non-empty** test string unless tests are genuinely not the right proof — in that case **do not** claim `satisfied`.
   - **`notes`**: short rationale, gaps, or follow-ups.
3. Emit **exactly one** markdown fenced block whose info string is **`jrdev-prd-matrix`** containing a **single JSON object** with:
   - **`schemaVersion`** (non-empty string, e.g. `"1"`),
   - **`requirements`**: array of row objects as above.

Example fence (structure only):

```jrdev-prd-matrix
{ "schemaVersion": "1", "requirements": [] }
```

## Completion

When the matrix is emitted and correct, end your output with the word **`COMPLETE`** **on its own line** so jrdev can detect completion (avoid burying `COMPLETE` inside other text).
