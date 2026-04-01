# jrdev — agent-queue orchestrator

`jrdev` drives a **plan → implement → review → merge** loop for GitHub issues labeled **`agent-queue`** (configurable), using the **Cursor CLI `agent`** for intelligence and Go for **git** / **`gh`** orchestration.

## What you need installed

| Requirement | Notes |
|-------------|--------|
| **Go** | Version **1.26** or newer (see `go.mod`). [Install Go](https://go.dev/doc/install). |
| **Git** | Used for repository detection, worktrees, and branches. |
| **GitHub CLI (`gh`)** | Must be installed and **logged in** with access to the repo you automate. [Install gh](https://cli.github.com/). |
| **Cursor `agent`** | The Cursor agent CLI must be on your **`PATH`** (or pass **`--agent`** with the full path to the binary). This is what runs the plan / implement / review steps. |

**Target repository:** Run `jrdev` with your **current working directory inside the project you want to automate** (any subfolder is fine), or pass **`--repo`** with the absolute path to that project’s root. The tool finds the git root by walking up for `.git`.

**Network / auth:** `gh` uses your normal GitHub authentication (`gh auth login`). For git operations, SSH or HTTPS credentials for `git push` / `git fetch` must work for that repo. On **`git fetch`** / **`git push`**, if the error looks like an SSH public-key failure, `jrdev` attempts a one-time interactive **`ssh-add`** recovery (requires a real TTY). If that still fails, follow the hints in the error or load your key before running `jrdev`.

## Cursor agent CLI permissions (`-p` / headless)

The Cursor **`agent`** CLI enforces **permissions** from [`cli-config.json`](https://cursor.com/docs/cli/reference/configuration) (global, project, or via **`CURSOR_CONFIG_DIR`**). In non-interactive mode, **shell** tools such as **`git`** or **`go`** only run if your config **allows** them.

If you pass **`--agent-permissions`** or **`--agent-cursor-config-dir`**, that value is used and repo / executable discovery is skipped (**`--agent-cursor-config-dir`** and **`--agent-permissions`** cannot be combined).

When **neither** flag is set: **`repo/.cursor/cli-config.json`** (if present), otherwise **`jrdev-agent-permissions.json`** next to the **`jrdev`** binary (if present). If neither exists, `jrdev` does not set **`CURSOR_CONFIG_DIR`** (the agent uses your normal Cursor global / project config only).

Mechanisms:

| Source | When |
|--------|------|
| **`--agent-cursor-config-dir`** | Directory that contains **`cli-config.json`** (passed through as **`CURSOR_CONFIG_DIR`**). Mutually exclusive with **`--agent-permissions`**. |
| **`--agent-permissions`** | Path to a small JSON file: **`{"allow":["git","go"],"deny":[]}`**. Each bare name becomes a Cursor **`Shell(name)`** entry; strings that already look like permission tokens (they contain **`(`**) are left as-is. `jrdev` writes a temporary **`cli-config.json`** (and always includes **`Read(**)`** / **`Write(**)`** so file tools still work while overriding the global config). |
| **`<repo>/.cursor/cli-config.json`** | If neither flag is set and this file exists, **`CURSOR_CONFIG_DIR`** is **`<repo>/.cursor`**. |
| **`jrdev-agent-permissions.json`** next to the **`jrdev` executable** | If nothing above applies and this file exists, it is used like **`--agent-permissions`**. |

For the full permission vocabulary (**`Read`**, **`Write`**, **`WebFetch`**, **`Mcp`**, **`Shell`** patterns), see Cursor’s [permissions reference](https://cursor.com/docs/cli/reference/permissions).

**Example** `jrdev-agent-permissions.json` (or **`--agent-permissions`** file):

```json
{
  "allow": ["git", "go", "gh"],
  "deny": []
}
```

With **`-v` / `--verbose`**, startup logs show whether a repo **`.cursor`** directory, a permissions file path, or **`--agent-cursor-config-dir`** is in effect.

Some Cursor docs describe project permissions only in **`.cursor/cli.json`**. `jrdev` auto-discovery specifically looks for **`.cursor/cli-config.json`** next to the git root so the directory can serve as **`CURSOR_CONFIG_DIR`** (full `cli-config.json` layout). If you only have **`cli.json`**, use **`--agent-cursor-config-dir`** pointing at a folder that contains the `cli-config.json` name your install expects, or maintain **`cli-config.json`** in **`.cursor`**.

## Linux — getting set up

1. **Install Go** (distribution packages are often old; prefer the official tarball or your distro’s backported `go` 1.26+).
2. **Install Git** (e.g. `sudo apt install git` on Debian/Ubuntu).
3. **Install `gh`** and sign in:
   ```bash
   gh auth login
   gh auth status
   ```
4. **Install Cursor and ensure `agent` is on `PATH`.** After install, verify:
   ```bash
   command -v agent
   agent --help   # or your installed CLI’s help flag
   ```
   If the binary lives outside `PATH`, use **`jrdev --agent /full/path/to/agent`**.
5. **Get `jrdev`:**
   - **From a local clone** (common for a private org repo):
     ```bash
     cd /path/to/jrdev
     go build -o jrdev .
     # optional: install into Go’s bin directory
     go install .
     ```
     Ensure **`$GOBIN`** or **`$GOPATH/bin`** is on your **`PATH`** if you use `go install`.
   - **From GitHub** (module path matches the repo):
     ```bash
     go install github.com/NUSAlaska/jrdev@latest
     ```
     For a **private** module, configure module privacy and git access, for example:
     ```bash
     go env -w GOPRIVATE=github.com/NUSAlaska/*
     ```
     Use SSH or a configured credential helper so `go` can fetch the module.

6. **Run from your application repo** (the repo that has the queued issues), not from the `jrdev` tree unless you only use **`--repo`**:
   ```bash
   cd /path/to/your-app
   jrdev --dry-run
   ```
   Use the **`jrdev`** binary from **`go install .`** / **`go build`**, or invoke the full path to the built binary.

## Windows — getting set up

1. **Install Go** from [go.dev/dl](https://go.dev/dl/) and confirm:
   ```powershell
   go version
   ```
2. **Install Git for Windows** if you do not already have it; use Git Bash or PowerShell with `git` on `PATH`.
3. **Install `gh`** ([installer](https://cli.github.com/)), then in PowerShell:
   ```powershell
   gh auth login
   gh auth status
   ```
4. **Cursor `agent`:** Install Cursor and add the directory containing **`agent.exe`** to your **user or system `PATH`**, or pass **`--agent`** with a full path (e.g. `C:\Users\You\AppData\Local\Programs\Cursor\...`). Verify in a **new** terminal:
   ```powershell
   Get-Command agent
   ```
5. **Build or install `jrdev` from a clone:**
   ```powershell
   cd C:\path\to\jrdev
   go build -o jrdev.exe .
   ```
   Or install Go’s bin folder onto `PATH` and run:
   ```powershell
   go install .
   ```
   Binaries go to **`%USERPROFILE%\go\bin`** unless **`GOBIN`** is set—add that folder to **PATH** in *Settings → Environment variables* so `jrdev` runs from anywhere.

6. **Private `go install` from GitHub:** Same as Linux: set `GOPRIVATE` for your org and ensure Git can authenticate to GitHub. **To avoid remote fetch entirely**, use **`go install .`** or **`go build`** from a local clone.

7. **Run against your app repo:**
   ```powershell
   cd C:\path\to\your-app
   jrdev --dry-run
   ```

## Quick start (from `jrdev` source tree)

```text
# Show all flags and examples
go run . help

# Create or finish .jrdev/config.yaml (TTY only; embedded language presets)
go run . init

# Queue sizing + preflight only (no agent smoke; stops before worktrees)
go run . --dry-run

# Full run (preflight includes a minimal agent smoke unless --dry-run)
go run .

# Start a new integration run even if a prior agent-queue/run-… worktree exists
go run . --fresh

# Cap the outer loop (default is 2N+3 for N = open labeled issues at start)
go run . --max-iterations 20
```

On a **terminal** (interactive stdin), a **successful** full run ends with a prompt: remove **`--worktrees`** and local **`agent-queue/*`** branches, or leave them to inspect prompts and diffs first.

After `go install .`, replace `go run .` with **`jrdev`** (or **`jrdev.exe`**) on `PATH`.

## Repository configuration (`.jrdev/config.yaml`)

Before the **plan → implement → review → merge** loop, jrdev loads a small YAML file. Use **`--config path`** to point at a file; the default is **`<repo>/.jrdev/config.yaml`**.

- **`config_ready`**: must be **`true`** for the pipeline to run. While it is **`false`**, an interactive terminal can finish setup via **`jrdev init`** or the auto-started wizard; **non-interactive** runs exit with an error and tell you to run **`jrdev init`** in a real terminal.
- **`lint`**, **`unit`**, **`integration`**: optional YAML lists of shell commands. They are rendered into agent prompts (**implement** / **review** use **lint** + **unit**; **merge** uses **integration**). **jrdev does not run a separate post-merge `go vet` / `go test` gate** on your target repo—the lists you configure are what the agents are asked to execute.
- If all three lists are empty but **`config_ready`** is **`true`**, prompts include one explicit **no checks configured** paragraph (same text in **lint** / **unit** / **integration** sections of the template) so it is obvious the repo chose empty checks.
- **`meta`**: optional key/value map (for example **`source_preset`** after **`jrdev init`**, or **`integration_blocked_action`** for **`JRDEV_INTEGRATION_BLOCKED:`** handling). Configuration is read only from this file (and **`--config`**); v1 does not load repo config from environment variables.

Background and roadmap (without duplicating the full spec): [jrdev PRD — issue #1](https://github.com/NUSAlaska/jrdev/issues/1).

## `jrdev init` and presets (TTY vs non-interactive)

**Language presets** live under **`internal/jrdev/presets/`** in source and are **embedded in the jrdev binary**. The wizard discovers them at runtime—**`go install`** / **`go build`** carry the same catalog, no extra files required beside **`config.yaml`**.

| Situation | Behavior |
|-----------|----------|
| **`jrdev init`** | Requires an **interactive terminal** (stdin is a TTY). If **`config_ready`** is already **`true`**, prints a short message and exits **0**. Otherwise: pick a preset (number or id), jrdev writes **`.jrdev/config.yaml`** with **`meta.source_preset`**, preset **lint** / **unit** / **integration** commands, and **`config_ready: false`**; you can edit the file, then confirm in the terminal so jrdev sets **`config_ready: true`**. |
| **`jrdev` with missing config** | **TTY**: runs the same wizard as **`jrdev init`**. **Non-TTY**: writes a minimal **stub** (`config_ready: false`, empty lists, **no** `meta`) and exits **1** with a hint to run **`jrdev init`**. |
| **jrdev run, config on disk but not ready** | **TTY**: wizard path again. **Non-TTY**: exits **1** (finish setup in a terminal). |

## Integration blocked (`JRDEV_INTEGRATION_BLOCKED:`)

During **merge**, the agent may print **one line** starting with **`JRDEV_INTEGRATION_BLOCKED:`** when integration checks from the config cannot pass after a reasonable attempt (details in **`prompt_merge.md`**). The line may include a short reason after the colon. The merge phase must still end with **`COMPLETE`** as usual.

Resolution order:

1. **`--integration-blocked abort`** or **`merge`** — forces the outcome; overrides **`meta.integration_blocked_action`**.
2. Else **`meta.integration_blocked_action`** in **`.jrdev/config.yaml`** (**`abort`** or **`merge`**). Invalid or missing values fall through.
3. Else **interactive stdin (TTY)**: prompt **Abort** vs **Merge (waive)** — default is abort.
4. Else **non-interactive**: default is **abort** if meta does not supply a valid **`abort`** / **`merge`**.

**Abort** leaves the issue open and, if the agent had already merged into the integration worktree, resets that worktree using **`ORIG_HEAD`**. **Merge (waive)** continues the usual close-label-push path without jrdev re-running integration for that merge attempt.

## Trust model (v1)

**Preflight** still validates **`git`**, **`gh`**, and (unless **`--dry-run`**) a token-only **agent** smoke—it does **not** execute your **lint** / **unit** / **integration** lists. During **implement**, **review**, and **merge**, those commands appear in the rendered prompts; the **Cursor agent** is expected to run them in the relevant worktree. There is **no** separate jrdev-side “verify pass” after the agent finishes in v1—checks are in the agent loop only.

## Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--repo` | (walk up) | Git repository root |
| `--config` | `<repo>/.jrdev/config.yaml` | Repository jrdev YAML (`config_ready`, `lint` / `unit` / `integration` command lists); must exist and have `config_ready: true` before the pipeline runs |
| `--integration-blocked` | — | When merge output contains `JRDEV_INTEGRATION_BLOCKED:` — force **`abort`** or **`merge`** (waive); overrides **`meta.integration_blocked_action`**; non-interactive default is **abort** unless meta sets a valid action |
| `--worktrees` | `.worktrees` | Directory under repo for worktrees (should be gitignored) |
| `--label` | `agent-queue` | Queue label |
| `--dry-run` | off | Skip agent smoke in preflight; exit before creating integration/issue worktrees |
| `--skip-pr` | off | Do not `gh pr create` when the loop finishes (end-of-run cleanup prompt still runs on a TTY) |
| `--max-iterations` | `2N+3` | Outer loop cap |
| `--integration-base` | `origin/main` | Base ref for new `agent-queue/run-…` branch |
| `--fresh` | off | Discard prior jrdev state: remove worktrees under `--worktrees` and local `agent-queue/run-*` / `agent-queue/issue-*` branches; skip the resume prompt (always start a new integration run) |
| `--agent` | `agent` on PATH | Cursor agent binary |
| `--agent-model` | `composer-2-fast` | Value passed to Cursor agent as `--model` |
| `--agent-permissions` | (see [permissions](#cursor-agent-cli-permissions--p--headless)) | JSON allow/deny file; `jrdev` materializes `cli-config.json` and sets `CURSOR_CONFIG_DIR` |
| `--agent-cursor-config-dir` | — | Use this directory as `CURSOR_CONFIG_DIR` (must contain `cli-config.json`); mutually exclusive with `--agent-permissions` |
| `--gh` | `gh` | GitHub CLI binary |
| `-v` / `--verbose` | off | Verbose preflight, per-cycle phases, agent invocation summary, git subprocess output |
| `-help`, `-h` | — | Print usage and exit |

## Behavior (summary)

1. **N** = count of open issues with the queue label; if **N == 0**, exit cleanly.
2. **Preflight** (once): `git` on PATH and `git version`, `gh auth status`, **`agent`** resolved and able to run `-h` / `--help`; unless `--dry-run`, a minimal **`agent -p`** smoke that must print a fixed token (that prompt forbids shell commands and file edits). Agent invocations use the [permission / `CURSOR_CONFIG_DIR`](#cursor-agent-cli-permissions--p--headless) rules above.
3. **Integration worktree**: After `git fetch origin`, if a prior run left a resumable **`agent-queue/run-…`** worktree under **`--worktrees`**, an **interactive** terminal prompts: **continue** with that branch and worktree, or **clean** and start fresh (same cleanup as **`--fresh`**). With **non-interactive** stdin, jrdev **resumes** automatically when possible and logs a hint to use **`--fresh`** if you want a clean run. **`--fresh`** skips the prompt and always clears that jrdev state, then creates a new **`agent-queue/run-<timestamp>`** and worktree from **`--integration-base`**.
4. Each **cycle**: plan (in integration worktree) → parse `<plan>…</plan>` JSON → **one** issue (first row) → issue worktree from integration tip → **implement**, **review** (if there are commits), and **merge** agent phases each loop until stdout contains **`COMPLETE`**, re-rendering the prompt with fresh git history/diff on every attempt (cap: 25 tries per phase); if implement produces zero commits, that phase runs again once the same way → the **merge** prompt directs the agent to run the **integration** commands from **`.jrdev/config.yaml`** after merging → **`git merge`** into the integration worktree (by the agent and/or `jrdev` if needed). If the agent prints **`JRDEV_INTEGRATION_BLOCKED:`**, jrdev applies **`--integration-blocked`**, **`meta.integration_blocked_action`**, or an interactive prompt, then either aborts (issue stays open; integration worktree may reset) or waives and continues. On success: **`gh issue close`** and remove label → push integration branch. **There is no subprocess `go vet` / `go test` quality gate after merge**; preflight is unchanged. **Issue worktrees and `agent-queue/issue-…` branches are not removed here**—they stay under **`--worktrees`** so you can inspect transcripts and diffs across every issue processed in the run.
5. Stops when the plan returns **`issues: []`**, or **max iterations** is reached, then **`gh pr create`** to **`main`** unless **`--skip-pr`**.
6. **End of a successful run**: On an **interactive** terminal (real TTY on stdin), `jrdev` asks whether to remove **all** jrdev-linked worktrees under **`--worktrees`** and delete local **`agent-queue/run-*`** / **`agent-queue/issue-*`** branches—the same scope as **`--fresh`**. **Y** / **yes** performs that cleanup; anything else (including Enter) **leaves** trees and branches for inspection. With **non-interactive** stdin (piped or CI), there is **no** prompt and **no** automatic cleanup; run **`jrdev --fresh`** before the next run if you want a clean slate, or delete worktrees/branches manually.

`main` is never merged by the tool directly; landing on `main` is via PR only.

## Local agent transcripts (`.jrdev/`)

Every Cursor **`agent`** invocation **`jrdev`** launches (preflight smoke, plan, implement, review, merge) stores the **full prompt** and the **combined stdout/stderr** of that run under the **process working directory** for that step—the **git repo root** during preflight, or the **integration / issue worktree** during the main loop:

| Path | Contents |
|------|----------|
| **`.jrdev/agent-runs/<timestamp>-<pid>/prompt.md`** | Exact prompt text passed to the agent (via `-p` pointing at this path, relative to that cwd). |
| **`.jrdev/agent-runs/<timestamp>-<pid>/output.md`** | Everything the agent process printed (success or failure). |

The first time artifacts are written in a given worktree, **`jrdev`** creates **`.jrdev/.gitignore`** so **`agent-runs/`** is ignored by Git in that tree. If your **`--worktrees`** directory is already gitignored (recommended), those paths usually stay hidden from **`git status`** entirely.

If you run preflight from the **repo root** and **`--worktrees`** is not under an ignored path, consider adding **`.jrdev/`** to your **repository root** `.gitignore` so local transcripts (and the nested `.gitignore`) never clutter **`git status`**.

With **`-v` / `--verbose`**, logs include the artifact directory for each agent run—useful to open the matching **`prompt.md`** and **`output.md`** after a failure.

## Failure and recovery

- **Successful run, worktrees still on disk**: Until you answer **Y** at the end-of-run prompt (TTY) or run **`--fresh`**, integration and issue worktrees remain under **`--worktrees`** with **`.jrdev/agent-runs/`** and branches **`agent-queue/run-…`** / **`agent-queue/issue-…`** so you can review everything from one session.
- **Interrupted run (Ctrl+C, crash, merge failure, etc.)**: The integration branch and worktrees under **`--worktrees`** are usually left in place (there is **no** end-of-run cleanup prompt because the run did not finish cleanly). On the **next** full run, if a valid **`agent-queue/run-…`** worktree still exists, you get a **resume vs fresh** prompt (TTY) or an automatic **resume** (non-interactive). Choose **fresh** in the prompt, or run **`jrdev --fresh`**, to remove jrdev worktrees under **`--worktrees`** and local **`agent-queue/run-*`** / **`agent-queue/issue-*`** branches before starting a new run.
- **Zero commits after implement retry**: run aborts; issue is not closed; integration branch and worktrees remain under `.worktrees/` for inspection.
- **Agent phase never prints `COMPLETE`** (after 25 attempts on implement, review, or merge): run aborts with an error; prompts should instruct the model to include `COMPLETE` when the phase is finished.
- **Merge or integration-check failure** (commands from **`.jrdev/config.yaml`**): fix locally in the integration or issue worktree, or remove worktrees/branches manually (or **`--fresh`** / **fresh** at the resume prompt).
- **SSH auth to `origin`**: ensure `ssh-add` / agent (or HTTPS) works; see **Network / auth** above. **Non-interactive** environments cannot complete interactive `ssh-add` recovery.
- **Agent smoke or plan/implement errors about blocked tools**: configure **[Cursor agent CLI permissions](#cursor-agent-cli-permissions--p--headless)** (`repo/.cursor/cli-config.json`, `jrdev-agent-permissions.json`, or flags).
- **Debugging agent failures**: inspect the latest directories under **`.jrdev/agent-runs/`** in the relevant cwd (repo root for smoke, integration or issue worktree for orchestration); see **[Local agent transcripts (`.jrdev/`)](#local-agent-transcripts-jrdev)**.

See **`PROMPTS.md`** for template placeholders in the embedded prompts.
