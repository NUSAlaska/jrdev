# jrdev — Plan phase

You run in the **integration git worktree** (branch `agent-queue/run-…`). **jrdev** set `cwd` here; do **not** create new worktrees or integration branches unless recovering from a documented failure.

## Repository guardrails (read-only planning)

- **API-owned write timestamps** — clients must not dictate `created_at` / `updated_at` on writes.
- **`pkg/env`** for env reads; **migrations human-only**; prefer **`pkg/schema`** types.
- **No product code changes** in this phase: reasoning and **`gh`** read-only usage only (if needed). Prefer the issue data below over redundant `gh` calls.

## Queue

- Queue label: **`{{.QueueLabel}}`**
- Consider **only** open issues with this label when building dependencies.

## Preloaded issues (JSON from `gh issue list`)

Use this list as the authoritative set of queued issues and their titles/bodies for dependency reasoning:

```json
{{.IssuesJSON}}
```

**Private GitHub:** Issue bodies above may link to other issues (e.g. PRDs) by URL. Anything **not** in this JSON is still readable with **`gh issue view`** from the repo root (your **`gh auth`** session); generic HTTP or **WebFetch** will not work without auth. Use **`gh`** only when necessary; shell **`gh`** must be permitted in agent config if you run it.

## Process

1. Build a **dependency graph** among **only** these labeled open issues.
2. Select **unblocked** issues first. Order is your judgment; **jrdev** takes the **first** entry in your output list in v1.
3. For each selected issue, set **`branch`** to exactly: `agent-queue/issue-{number}-{kebab-slug-from-title}` (Git branch naming: use lowercase kebab-case from the title; collapse non-alphanumeric runs to single `-`; trim `-`; if empty use `issue`).
4. If **everything** appears blocked, pick the **single** best “weakest blockage” candidate per team policy (still only from this list).

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
