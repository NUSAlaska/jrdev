# jrdev — Plan phase

You run in the **integration git worktree** (branch `agent-queue/run-…`). **jrdev** set `cwd` here; do **not** create new worktrees or integration branches unless recovering from a documented failure.

## Queue

- Queue label: **`{{.QueueLabel}}`**
- Consider **only** open issues with this label when building dependencies.

## Preloaded issues (JSON from `gh issue list`)

Use this list as the authoritative set of queued issues and their titles/bodies for dependency reasoning:

```json
{{.IssuesJSON}}
```

**Private GitHub:** Issue bodies above may link to other issues (e.g. PRDs) by URL. Anything **not** in this JSON is still readable with **`gh issue view`** from the repo root (your **`gh auth`** session); generic HTTP or **WebFetch** will not work without auth. 

## Process

Analyze the open issues and build a dependency graph. For each issue, determine whether it blocks or is blocked by any other open issue.

An issue B is blocked by issue A if:

    B requires code or infrastructure that A introduces
    B and A modify overlapping files or modules, making concurrent work likely to produce merge conflicts
    B's requirements depend on a decision or API shape that A will establish

An issue is unblocked if it has zero blocking dependencies on other open issues.

For each unblocked issue, assign a branch name using the format `agent-queue/issue-{number}-{slug}`

## Output contract (mandatory)

Output **nothing** after the plan except the following wrapper. Inside `<plan>...</plan>` put **only** valid JSON:

```json
{ "issues": [ { "number": <int>, "title": "<string>", "branch": "<string>" } ] }
```

- Use an **empty** `"issues": []` array when there is no work to do this cycle (jrdev will stop the outer loop).
- Do not wrap the JSON in markdown code fences **inside** the `<plan>` tags.

Example (your real output must use real issue numbers from the JSON above):

    <plan>
    { "issues": [ { "number": 12, "title": "Fix flux capacitor", "branch": "agent-queue/issue-12-fix-flux-capacitor" } ] }
    </plan>
