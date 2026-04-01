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

# Queue sizing + preflight only (no agent smoke; stops before worktrees)
go run . --dry-run

# Full run (preflight includes a minimal agent smoke unless --dry-run)
go run .

# Cap the outer loop (default is 2N+3 for N = open labeled issues at start)
go run . --max-iterations 20
```

After `go install .`, replace `go run .` with **`jrdev`** (or **`jrdev.exe`**) on `PATH`.

## Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--repo` | (walk up) | Git repository root |
| `--worktrees` | `.worktrees` | Directory under repo for worktrees (should be gitignored) |
| `--label` | `agent-queue` | Queue label |
| `--dry-run` | off | Skip agent smoke in preflight; exit before creating integration/issue worktrees |
| `--skip-pr` | off | Do not `gh pr create` when the loop finishes |
| `--max-iterations` | `2N+3` | Outer loop cap |
| `--integration-base` | `origin/main` | Base ref for new `agent-queue/run-…` branch |
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
3. Creates **`agent-queue/run-<timestamp>`** and a worktree under **`--worktrees`** from **`--integration-base`**.
4. Each **cycle**: plan (in integration worktree) → parse `<plan>…</plan>` JSON → **one** issue (first row) → issue worktree from integration tip → **implement**, **review** (if there are commits), and **merge** agent phases each loop until stdout contains **`COMPLETE`**, re-rendering the prompt with fresh git history/diff on every attempt (cap: 25 tries per phase); if implement produces zero commits, that phase runs again once the same way → **`go vet ./...`** and **`go test ./...`** on integration → **`gh issue close`** and remove label → push integration branch.
5. Stops when the plan returns **`issues: []`**, or **max iterations** is reached, then **`gh pr create`** to **`main`** unless **`--skip-pr`**.

`main` is never merged by the tool directly; landing on `main` is via PR only.

## Failure and recovery

- **Zero commits after implement retry**: run aborts; issue is not closed; integration branch and worktrees remain under `.worktrees/` for inspection.
- **Agent phase never prints `COMPLETE`** (after 25 attempts on implement, review, or merge): run aborts with an error; prompts should instruct the model to include `COMPLETE` when the phase is finished.
- **Merge / `go vet` / `go test` failure**: fix locally in the integration or issue worktree, or remove worktrees/branches manually.
- **SSH auth to `origin`**: ensure `ssh-add` / agent (or HTTPS) works; see **Network / auth** above. **Non-interactive** environments cannot complete interactive `ssh-add` recovery.
- **Agent smoke or plan/implement errors about blocked tools**: configure **[Cursor agent CLI permissions](#cursor-agent-cli-permissions--p--headless)** (`repo/.cursor/cli-config.json`, `jrdev-agent-permissions.json`, or flags).

See **`PROMPTS.md`** for template placeholders in the embedded prompts.
