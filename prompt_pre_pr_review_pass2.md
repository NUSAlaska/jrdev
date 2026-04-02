# jrdev — Pre-PR review — Pass 2 (gap fixes)

You are in the **integration git worktree** on branch **`{{.IntegrationBranch}}`**. This is **Pass 2 only**: fix gaps called out by the **current** Pass 1 matrix for **round {{.Round}}**. Do **not** re-run a full Pass 1 resweep yourself; jrdev will run Pass 1 again on the next cycle if needed.

## Linked issues (scope)

{{range $i, $n := .IssueNumbers}}{{if $i}}, {{end}}#{{$n}}{{end}}

### Issue bodies (context)

{{.IssuesMarkdown}}

### Linked issues and PRDs (private GitHub)

If a body links to other issues (for example a parent PRD) by URL or `#n`, read them with **`gh issue view`** from the repository root — not unauthenticated HTTP or **WebFetch**.

{{if .NonInteractive}}
## Non-interactive mode

Standard input is **not** a TTY. **Do not** wait for user input or ask the operator questions. Use best judgment; proceed deterministically (same spirit as other jrdev non-interactive agent phases).
{{end}}

{{if .BonusSteeringNote}}
## Operator steering (bonus Pass 1→2 cycle)

{{.BonusSteeringNote}}

{{end}}
{{if .PriorPassArtifactsMarkdown}}
## Prior pass artifacts (full history so far)

{{.PriorPassArtifactsMarkdown}}

{{end}}

## Current Pass 1 matrix (this round)

Valid JSON requirements matrix from Pass 1 (address every row with status `not_satisfied`, `unknown`, or `conflict`; improve evidence where weak):

```json
{{.CurrentPass1MatrixJSON}}
```

## Repository context

Last commits in this worktree:

```
{{.CommitHistory}}
```

Diff ({{.IntegrationBase}}..HEAD):

```
{{.GitDiff}}
```

**Integration base**: `{{.IntegrationBase}}`

## Configured checks

**Lint** (run when you change code; fix new violations you introduce):

{{.LintTests}}

**Unit** (run when you change code):

{{.UnitTests}}

## Task (gap fixes)

1. Prioritize rows with `not_satisfied`, then `conflict`, then `unknown`.
2. Make **minimal** code or test changes that move requirements toward **satisfied** with real evidence. **Do not** broaden scope beyond the linked issues / matrix.
3. **Every commit** on this pass must use a subject line starting with **`JRDEV: Pre-PR review pass 2 -`** (prefix exactly; you may add a short description after the dash).
4. When you touch code, run the repo’s configured **lint** and **unit** commands above and keep them green for your edits.
5. Leave the working tree **clean** when you finish (all changes committed — jrdev will verify before the next cycle).
6. **Do not** automatically produce a new full requirements matrix (no Pass 1 resweep); focus on fixes.
7. Emit **exactly one** markdown fenced block tagged **`jrdev-pre-pr-review-handoff`** containing a single JSON object with:
   - **`round`**: integer, must match **{{.Round}}**
   - **`gapSummary`**: short description of what you fixed or punted
   - **`matrixDelta`**: free text describing how the branch state moved relative to the matrix (tests, files, decisions)
   - **`draftPRTitle`**: suggested PR title for the integration branch (for the human PR author; merged into session handoff)
   - **`draftPRBody`**: suggested PR body (markdown plain text inside the JSON string)
   - **`gapNotes`**: remaining gaps or follow-ups for the author
   - **`conflictNotes`**: notable conflicts or ambiguities still open

Example structure (fill with real values):

```jrdev-pre-pr-review-handoff
{
  "round": {{.Round}},
  "gapSummary": "",
  "matrixDelta": "",
  "draftPRTitle": "",
  "draftPRBody": "",
  "gapNotes": "",
  "conflictNotes": ""
}
```

## Completion

When fixes and the handoff fence are emitted, end with **`COMPLETE`** on its own line. If **no** commits or working-tree edits are needed, you may use **`COMPLETE NO COMMIT`** on its own line instead (then still emit the handoff fence with honest `gapSummary` / notes). If you use **`COMPLETE NO COMMIT`**, the working tree must still be clean.
