# jrdev — Pre-PR review — Pass 3 (test design, no check execution)

You are in the **integration git worktree** on branch **`{{.IntegrationBranch}}`**. This is **Pass 3 only**: review **test design**, **coverage**, and **alignment** with **PRD testing expectations** and **project-wide test strictness** inferred from **existing tests** (read source; **GM-008**).

## GM-008 — no lint/unit/integration execution

**Do not** run the repository’s configured **lint**, **unit**, or **integration** commands, and **do not** ask the operator to run them for you during this pass. jrdev does **not** trigger those commands in Pass 3. Base judgments on **static review** of code and tests (read files, compare patterns, trace requirements from issues/PRD text).

You **may** add or improve tests as code changes; after edits, your **commit message prefix** for any commit must be **`JRDEV: Pre-PR review pass 3 -`**. End with a **clean working tree** (commit everything meaningful, or **no changes**). If you make **no** commits, you may finish with **`COMPLETE NO COMMIT`** on its own line; otherwise end with **`COMPLETE`**.

## Linked issues (scope)

{{range $i, $n := .IssueNumbers}}{{if $i}}, {{end}}#{{$n}}{{end}}

### Issue bodies (requirements and testing expectations)

{{.IssuesMarkdown}}

### Linked issues and PRDs (private GitHub)

If a body links to other issues (for example a parent PRD) by URL or `#n`, read them with **`gh issue view`** from the repository root — not unauthenticated HTTP or **WebFetch**.

{{if .NonInteractive}}
## Non-interactive mode

Standard input is **not** a TTY. **Do not** wait for user input or ask the operator questions. Use best judgment; proceed deterministically.
{{end}}

{{if .PriorPassArtifactsMarkdown}}
## Prior pass artifacts (Pass 1↔2 — full history)

{{.PriorPassArtifactsMarkdown}}

{{end}}

## Session handoff (Pass 1↔2 — `handoff.json`)

Summary JSON written by jrdev after the matrix loop (draft PR text, gap notes, final round):

```json
{{.SessionHandoffJSON}}
```

**Final Pass 1↔2 round**: {{.FinalRound}}

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

## Task

1. From issue/PRD text, infer **what testing the branch is expected to demonstrate** (explicit or implied).
2. From **representative existing tests** (same packages, similar commands in `*_test.go`), infer **local conventions**: table-driven style, helper patterns, assertion style, error handling in tests, etc.
3. Assess whether **current tests** on this branch **adequately** cover the linked requirements and edge cases **by design** (not by running the suite).
4. If gaps are clear and **small**, **add or refine tests** (still **no** lint/unit/integration **execution** in this pass). Prefer minimal, consistent changes.
5. Emit **exactly one** markdown fenced block tagged **`jrdev-pre-pr-review-pass3`** containing a single JSON object with:
   - **`summary`**: one short paragraph on overall test posture for this branch
   - **`testDesign`**: key design choices, strengths, or weaknesses you observed
   - **`coverage`**: what seems covered vs. missing or risky (reasoning only; no executed coverage reports)
   - **`prdTestingAlignment`**: how tests match (or miss) PRD/issue testing expectations
   - **`strictnessInference`**: how you inferred project-wide strictness from existing tests (cite a few **file paths** only)
   - **`followUps`**: optional remaining work for the human author (empty string if none)

Example structure (replace with real analysis):

```jrdev-pre-pr-review-pass3
{
  "summary": "",
  "testDesign": "",
  "coverage": "",
  "prdTestingAlignment": "",
  "strictnessInference": "",
  "followUps": ""
}
```

## Completion

After the artifact fence (and any commits), end with **`COMPLETE`** or **`COMPLETE NO COMMIT`** on its own line, consistent with working-tree state.
