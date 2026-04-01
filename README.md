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

**Network / auth:** `gh` uses your normal GitHub authentication (`gh auth login`). For git operations, SSH or HTTPS credentials for `git push` / fetch must work for that repo.

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
| `--gh` | `gh` | GitHub CLI binary |
| `-v` | off | Verbose agent/subprocess logging |
| `-help`, `-h` | — | Print usage and exit |

## Behavior (summary)

1. **N** = count of open issues with the queue label; if **N == 0**, exit cleanly.
2. **Preflight** (once): `git`, `gh auth status`, `agent` present; unless `--dry-run`, a minimal non-destructive **`agent -p`** smoke.
3. Creates **`agent-queue/run-<timestamp>`** and a worktree under **`--worktrees`** from **`--integration-base`**.
4. Each **cycle**: plan (in integration worktree) → parse `<plan>…</plan>` JSON → **one** issue (first row) → issue worktree from integration tip → implement (retry once on zero commits) → review if commits → merge phase → **`go vet ./...`** and **`go test ./...`** on integration → **`gh issue close`** and remove label → push integration branch.
5. Stops when the plan returns **`issues: []`**, or **max iterations** is reached, then **`gh pr create`** to **`main`** unless **`--skip-pr`**.

`main` is never merged by the tool directly; landing on `main` is via PR only.

## Failure and recovery

- **Zero commits after implement retry**: run aborts; issue is not closed; integration branch and worktrees remain under `.worktrees/` for inspection.
- **Merge / `go vet` / `go test` failure**: fix locally in the integration or issue worktree, or remove worktrees/branches manually.

See **`PROMPTS.md`** for template placeholders in the embedded prompts.
