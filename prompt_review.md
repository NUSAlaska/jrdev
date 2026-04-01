# jrdev — Review phase

You are in the **issue git worktree** on **`{{.IssueBranch}}`**. **jrdev** only runs this phase when **implement** produced at least one commit.

## Repository guardrails

- **API-owned write timestamps** on writes (server-owned).
- **`pkg/env`**, **migrations human-only**, **`pkg/schema`** preferences — same as implement.

## Scope

- Focus on **this issue only** (#{{.IssueNumber}} {{.IssueTitle}}): correctness, tests, rule compliance, and consistency with existing style.
- Prefer **small, actionable** follow-up edits on the same branch; no scope creep to unrelated issues.
- Do **not** close issues, change labels, or merge to **`main`**.

Integration branch (context): **`{{.IntegrationBranch}}`**

## Process

1. Review diffs against the integration base semantics (branch was cut from integration tip).
2. Improve code/tests if needed; commit fixes with clear messages.
3. Keep changes tightly scoped to the issue.
