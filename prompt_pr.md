# jrdev — Pull request title and body

You help write a **GitHub pull request** for an automated integration branch produced by **jrdev**.

## Context

- **Integration branch:** `{{.IntegrationBranch}}`
- **Queue label:** `{{.QueueLabel}}`
- **PR base:** `{{.PRBase}}` (compare commits and diff against this branch)

{{if .PrePRReviewHandoffPresent}}
## Pre-pr-review handoff (latest run)

The branch has **pre-pr-review** artifacts recorded under **`.jrdev/pre-pr-review/`**. Treat the handoff below as authoritative for **what the review passes concluded** (matrix gaps, testing notes, check status) and for **draft PR text** from earlier passes when present. **Still reconcile** with the commit list and diff below; if they disagree, prefer facts visible in git and call out the mismatch in the PR body.

### Summaries

{{.PrePRReviewHandoffSummary}}

### Artifact paths (JSON)

{{.PrePRReviewArtifactPaths}}
{{end}}

## Recent commits on the integration branch

```
{{.CommitHistory}}
```

## Diff vs `{{.PRBase}}`

```diff
{{.GitDiff}}
```

## Your task

1. Propose a **clear, conventional PR title** (roughly 50–72 characters when possible) that summarizes the *substance* of the changes for human reviewers — not generic text like "jrdev integration".
2. Write a **PR body** in Markdown that:
   - Explains **what** changed and **why** (for reviewers),
   - Calls out **risk areas** or **follow-ups** if obvious from the diff,
   - Uses a short bullet list when it helps readability.
3. When the **pre-pr-review handoff** section is present above, **use it**: surface draft title/body and gap/testing/check-status notes where they help reviewers; merge them with the git story instead of ignoring them.
4. Ground claims in **git** (commits + diff). Do not invent files or behavior not supported by the diff/commits and handoff text; if the handoff references detail that is not in the diff, phrase it cautiously or point readers to the listed artifact JSON files.

## Required output format (exact tags)

Your reply must contain these two blocks **verbatim** (XML-style tags, nothing else wrapping the inner text). Put the PR title on **one line** inside `<pr_title>`.

<pr_title>
Example: concise summary of the change
</pr_title>
<pr_body>
## Summary

- …

## Notes

…
</pr_body>

## Completion

After those blocks, end your output with this exact line on its own:

COMPLETE
