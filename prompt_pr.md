# jrdev — Pull request title and body

You help write a **GitHub pull request** for an automated integration branch produced by **jrdev**.

## Context

- **Integration branch:** `{{.IntegrationBranch}}`
- **Queue label:** `{{.QueueLabel}}`
- **PR base:** `{{.PRBase}}` (compare commits and diff against this branch)

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
3. Base your summary only on the commit messages and diff above; do not invent files or behavior not shown there.

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
