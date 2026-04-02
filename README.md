# jrdev — agent-queue orchestrator

`jrdev` drives a **plan → implement → review → merge** loop for GitHub issues labeled **`agent-queue`** (configurable), using the **Cursor CLI `agent`** for intelligence and Go for **git** / **`gh`** orchestration.

## Recommended workflow: three bundled skills (in order)

Before you run `jrdev`, this repository includes **three Cursor agent skills** under [`Skills/`](Skills/). Use them **one after the other** so work is scoped, written down, and split into slices that `jrdev` can execute reliably.

1. **[`grill-me`](Skills/grill-me/SKILL.md)** — Use the chat **depth-first**: settle the big decisions for a new feature or refactor (one question per turn until each branch is clear). This is where most of the product and engineering detail should be decided verbally.

2. **[`write-a-prd`](Skills/write-a-prd/SKILL.md)** — Takes that settled discussion (or another decision artifact) and turns it into a **PRD**, then **opens it as a GitHub issue**. The PRD is optimized for the next step—stable **`GM-xx`** decision IDs and structure—not a raw transcript.

3. **[`prd-to-issues`](Skills/prd-to-issues/SKILL.md)** — Reads that PRD issue and **breaks it into vertical slices** (tracer bullets): independently grabbable GitHub issues that match how `jrdev` processes the queue, instead of vague horizontal layers.

After those issues exist (and carry the **`agent-queue`** label, or whatever you set with **`--label`**), run **`jrdev`** from the target application repository.

## Basic use case

**Target repository:** run `jrdev` with your **current working directory inside the project you want to automate** (any subfolder is fine), or pass **`--repo`** with the absolute path to that project’s root. The tool finds the git root by walking up for **`.git`**.

Each cycle: the agent **plans** in an integration worktree, picks **one** queued issue, **implements** and **reviews** in an issue worktree, then **merges** back with **integration** checks from your config. Successful issues are closed and unlabeled; when the plan is empty or limits hit, `jrdev` can open a PR to **`main`**.

**Network / auth:** `gh` uses your GitHub session (`gh auth login`). For **`git fetch`** / **`git push`**, SSH or HTTPS must work. If **`git fetch`** / **`git push`** fails with SSH key errors, `jrdev` may try interactive **`ssh-add`** once (needs a real TTY); otherwise fix credentials before running.

---

## Configure the Cursor agent (`cli-config.json`)

The Cursor **`agent`** CLI reads **[`cli-config.json`](https://cursor.com/docs/cli/reference/configuration)** (global, project, or via **`CURSOR_CONFIG_DIR`**). In headless mode (**`-p`**), **shell** tools such as **`git`**, **`go`**, or **`gh`** only run if your config **allows** them.

**Project layout:** at the git root, put **`cli-config.json`** in **`.cursor/`**. If neither **`--agent-permissions`** nor **`--agent-cursor-config-dir`** is set and **`<repo>/.cursor/cli-config.json`** exists, `jrdev` sets **`CURSOR_CONFIG_DIR`** to **`<repo>/.cursor`**.

**Example** **`<repo>/.cursor/cli-config.json`** (trim and extend `permissions.allow` to match what your agents need; see Cursor’s [permissions reference](https://cursor.com/docs/cli/reference/permissions)):

```json
{
  "permissions": {
    "allow": [
      "Shell(git)",
      "Shell(gh)",
      "Shell(go)",
      "Read(**)",
      "Write(**)"
    ],
    "deny": []
  },
  "version": 1
}
```

**Overrides:**

- **`--agent-cursor-config-dir`** — directory that contains **`cli-config.json`** (passed as **`CURSOR_CONFIG_DIR`**). Mutually exclusive with **`--agent-permissions`**.
- **`--agent-permissions`** — path to a small JSON file like **`{"allow":["git","go","gh"],"deny":[]}`**; `jrdev` writes a temporary **`cli-config.json`** (and includes **`Read(**)`** / **`Write(**)`**).

If neither flag is set: **`<repo>/.cursor/cli-config.json`** (if present), else **`jrdev-agent-permissions.json`** next to the **`jrdev`** binary (if present). If neither exists, `jrdev` does not set **`CURSOR_CONFIG_DIR`**.

With **`-v` / `--verbose`**, startup logs show which discovery path is in effect.

Some Cursor installs document project config as **`.cursor/cli.json`**. `jrdev` discovery expects **`.cursor/cli-config.json`** so the folder can serve as **`CURSOR_CONFIG_DIR`**.

---

## Configure the repository (`.jrdev/config.yaml`)

Default path: **`<repo>/.jrdev/config.yaml`**. Override with **`--config`**.

- **`config_ready`**: must be **`true`** before the pipeline runs. **`jrdev init`** (TTY) creates or finishes the file; non-interactive runs with **`config_ready: false`** exit with an error pointing you to **`jrdev init`**.
- **`lint`**, **`unit`**, **`integration`**: optional YAML lists of shell commands. They are embedded in agent prompts (**implement** / **review** use **lint** + **unit**; **merge** uses **integration**). **jrdev does not run a separate post-merge gate**—the lists you configure are what agents are asked to run.
- If all three lists are empty but **`config_ready`** is **`true`**, prompts include an explicit **no checks configured** note.
- **`meta`**: optional map (e.g. **`source_preset`** after **`jrdev init`**, **`integration_blocked_action`** for **`JRDEV_INTEGRATION_BLOCKED:`**). v1 does not load repo config from environment variables.

**Example:**

```yaml
config_ready: true

lint:
  - go vet ./...

unit:
  - go test ./...

integration:
  - go test -count=1 ./...
```

Background and roadmap: [jrdev PRD — issue #1](https://github.com/NUSAlaska/jrdev/issues/1).

---

## Log in: Git, GitHub CLI, and Cursor agent

| Piece | What to verify |
|--------|----------------|
| **Git** | On **`PATH`**; **`git version`** works. Credentials for **`origin`** (HTTPS or SSH) must allow **fetch** / **push** for the repo you automate. |
| **`gh`** | Installed; **`gh auth login`**, then **`gh auth status`** shows the right account and can access the repo. |
| **Cursor `agent`** | CLI on **`PATH`** (or pass **`--agent`** with the full path). **`agent --help`** (or your build’s help flag) works. Sign in through Cursor as you normally would so headless runs can use your account. |

**Linux — quick checklist**

1. **Go** 1.26+ (see **`go.mod`**).
2. **Git** — e.g. `sudo apt install git` where needed.
3. **`gh auth login`** and **`gh auth status`**.
4. **`command -v agent`** and **`agent --help`**. If the binary is not on **`PATH`**, use **`jrdev --agent /full/path/to/agent`**.

**Windows — quick checklist**

1. **Go** from [go.dev/dl](https://go.dev/dl/); **`go version`**.
2. **Git for Windows**; **`git`** on **`PATH`**.
3. **`gh auth login`** / **`gh auth status`** in PowerShell.
4. **`Get-Command agent`** in a **new** terminal after install; or **`--agent`** with a full path to **`agent.exe`**.

---

## Common ways to run

Install **`jrdev`** first (see **[Install and build](#install-and-build)**) so the binary is on your **`PATH`**—on Windows the default install dir is **`%USERPROFILE%\go\bin`** unless **`GOBIN`** is set.

```text
# Show all flags and examples
jrdev help

# Create or finish .jrdev/config.yaml (TTY only; embedded language presets)
jrdev init

# Queue sizing + preflight only (no agent smoke; stops before worktrees)
jrdev --dry-run

# Full run (preflight includes a minimal agent smoke unless --dry-run)
jrdev

# Start a new integration run even if a prior agent-queue/run-… worktree exists
jrdev --fresh

# Cap the outer loop (default is 2N+3 for N = open labeled issues at start)
jrdev --max-iterations 20
```

On Windows, use **`jrdev.exe`** if your shell resolves that more reliably than **`jrdev`**.

On a **TTY**, a **successful** full run ends with a prompt: remove **`--worktrees`** and local **`agent-queue/*`** branches, or leave them to inspect prompts and diffs first.

**From your application repo** (the repo that has the queued issues):

```bash
cd /path/to/your-app
jrdev --dry-run   # then drop --dry-run for a full run
```

**Developing `jrdev` from a clone:** you can run **`go run . help`**, **`go run . init`**, and so on from the repository root instead of **`jrdev`**—behavior matches the same CLI.

---

## Install and build

### Requirements

| Requirement | Notes |
|-------------|--------|
| **Go** | Version **1.26** or newer (see `go.mod`). [Install Go](https://go.dev/doc/install). |
| **Git** | Repository detection, worktrees, branches. |
| **GitHub CLI (`gh`)** | [Install gh](https://cli.github.com/); logged in with access to the repo you automate. |
| **Cursor `agent`** | On **`PATH`** or **`--agent`**. |

### Install `jrdev` (recommended)

Put Go’s install directory on your **`PATH`** (**`$GOBIN`**, **`$GOPATH/bin`**, or **`%USERPROFILE%\go\bin`** on Windows when using defaults).

**Public module:**

```bash
go install github.com/NUSAlaska/jrdev@latest
```

**Private module:** configure fetch and auth, then install the same way:

```bash
go env -w GOPRIVATE=github.com/NUSAlaska/*
go install github.com/NUSAlaska/jrdev@latest
```

Use SSH or a credential helper so **`go`** can clone or download the module.

### From a local clone (optional)

If you prefer not to install from the network, or you are hacking on **`jrdev`**:

**Linux / macOS:**

```bash
cd /path/to/jrdev
go install .
```

**Windows:**

```powershell
cd C:\path\to\jrdev
go install .
```

To produce a binary in the current directory without installing into **`GOBIN`**, use **`go build -o jrdev`** (or **`jrdev.exe`** on Windows) from the repo root.

---

## `jrdev init` and presets (TTY vs non-interactive)

**Language presets** live under **`internal/jrdev/presets/`** and are **embedded in the binary**.

| Situation | Behavior |
|-----------|----------|
| **`jrdev init`** | **TTY** only. If **`config_ready`** is already **`true`**, exits **0**. Otherwise: preset → writes **`.jrdev/config.yaml`**; you edit and confirm → **`config_ready: true`**. |
| **`jrdev` with missing config** | **TTY**: same wizard as **`jrdev init`**. **Non-TTY**: stub file, exit **1**, hint **`jrdev init`**. |
| **Run with config not ready** | **TTY**: wizard. **Non-TTY**: exit **1**. |

---

## Integration blocked (`JRDEV_INTEGRATION_BLOCKED:`)

During **merge**, the agent may print a line starting with **`JRDEV_INTEGRATION_BLOCKED:`** when integration checks cannot pass after a reasonable attempt (see **`prompt_merge.md`**). The merge phase must still end with **`COMPLETE`**.

Resolution order:

1. **`--integration-blocked abort`** or **`merge`** — overrides **`meta.integration_blocked_action`**.
2. Else **`meta.integration_blocked_action`** in **`config.yaml`** (**`abort`** or **`merge`**).
3. Else **TTY**: prompt **Abort** vs **Merge (waive)** — default abort.
4. Else **non-interactive**: default **abort** unless meta sets a valid action.

**Abort** leaves the issue open and may reset the integration worktree from **`ORIG_HEAD`**. **Merge (waive)** continues close-label-push without re-running integration for that attempt.

---

## Trust model (v1)

**Preflight** validates **`git`**, **`gh`**, and (unless **`--dry-run`**) a token-only **agent** smoke—it does **not** run your **lint** / **unit** / **integration** lists. During **implement**, **review**, and **merge**, those commands appear in prompts; the agent is expected to run them. There is **no** separate jrdev-side verify-after-agent step in v1.

---

## Flags

| Flag | Default | Purpose |
|------|---------|---------|
| `--repo` | (walk up) | Git repository root |
| `--config` | `<repo>/.jrdev/config.yaml` | `config_ready`, command lists; must be ready before the pipeline runs |
| `--integration-blocked` | — | On `JRDEV_INTEGRATION_BLOCKED:` — force **`abort`** or **`merge`** |
| `--worktrees` | `.worktrees` | Worktree directory (should be gitignored) |
| `--label` | `agent-queue` | Queue label |
| `--dry-run` | off | Skip agent smoke in preflight; exit before issue worktrees |
| `--skip-pr` | off | Do not `gh pr create` at end |
| `--max-iterations` | `2N+3` | Outer loop cap |
| `--integration-base` | `origin/main` | Base for new `agent-queue/run-…` branch |
| `--fresh` | off | Remove jrdev worktrees and local `agent-queue/run-*` / `agent-queue/issue-*` branches; start clean |
| `--agent` | `agent` on PATH | Cursor agent binary |
| `--agent-model` | `composer-2-fast` | Passed to agent as `--model` |
| `--agent-permissions` | (see [Configure the Cursor agent](#configure-the-cursor-agent-cliconfigjson)) | JSON allow/deny; materializes `cli-config.json` |
| `--agent-cursor-config-dir` | — | `CURSOR_CONFIG_DIR`; mutually exclusive with `--agent-permissions` |
| `--gh` | `gh` | GitHub CLI binary |
| `-v` / `--verbose` | off | Verbose logging |
| `-help`, `-h` | — | Usage |

---

## Behavior (summary)

1. **N** = open issues with the queue label; if **N == 0**, exit cleanly.
2. **Preflight**: **`git`**, **`gh auth status`**, **`agent`** help; unless **`--dry-run`**, minimal **`agent -p`** smoke (fixed token; no shell/file edits). Uses **[permission / `CURSOR_CONFIG_DIR`](#configure-the-cursor-agent-cliconfigjson)** rules.
3. **Integration worktree**: After **`git fetch origin`**, if a resumable **`agent-queue/run-…`** exists under **`--worktrees`**, **TTY** prompts continue vs clean (**`--fresh`** skips prompt and cleans). **Non-TTY** resumes when possible.
4. Each **cycle**: plan → parse **`<plan>…</plan>`** → first queued issue → issue worktree → **implement**, **review** (if commits), **merge** (agent loops until **`COMPLETE`**, up to 25 tries per phase); **merge** includes **integration** commands from config. On **`JRDEV_INTEGRATION_BLOCKED:`**, apply flags/meta/prompt as above. On success: close issue, remove label, push integration branch. Issue worktrees and **`agent-queue/issue-…`** branches remain under **`--worktrees`** for inspection.
5. Stops when plan returns **`issues: []`** or **max iterations**, then **`gh pr create`** unless **`--skip-pr`**.
6. **Successful TTY end**: optional cleanup of all jrdev worktrees under **`--worktrees`** and local **`agent-queue/run-*`** / **`agent-queue/issue-*`** branches. **Non-interactive**: no cleanup; use **`--fresh`** next time if desired.

`main` is never merged by the tool directly; landing is via PR only.

---

## Local agent transcripts (`.jrdev/`)

Every **`agent`** invocation stores **prompt** and **combined stdout/stderr** under the cwd for that step (repo root or worktree):

| Path | Contents |
|------|----------|
| **`.jrdev/agent-runs/<timestamp>-<pid>/prompt.md`** | Prompt passed to the agent |
| **`.jrdev/agent-runs/<timestamp>-<pid>/output.md`** | Agent process output |

The first write creates **`.jrdev/.gitignore`** with **`agent-runs/`** so transcripts stay ignored while **`config.yaml`** may stay tracked.

If **`--worktrees`** is not ignored, add **`.jrdev/agent-runs/`** at the repo root **`.gitignore`** (not necessarily all of **`.jrdev/`**).

With **`-v`**, logs include artifact paths for debugging.

---

## Failure and recovery

- **Successful run, worktrees left**: answer **Y** at end prompt (TTY) or run **`--fresh`** to clean.
- **Interrupted run**: integration branch/worktrees usually remain; next run may **resume** (TTY prompt or auto). Use **`--fresh`** for a clean start.
- **Zero commits after implement retry**: run aborts; issue stays open.
- **Phase never prints `COMPLETE`** (after 25 attempts): run aborts.
- **Merge / integration failures**: fix locally or **`--fresh`** / **fresh** at resume prompt.
- **SSH to `origin`**: fix **`ssh-add`** / keys; non-interactive cannot complete interactive **`ssh-add`** recovery.
- **Blocked tools in agent**: fix **[`cli-config.json` / permissions](#configure-the-cursor-agent-cliconfigjson)**.
- **Debugging**: latest **`.jrdev/agent-runs/`** under the relevant cwd.

See **`PROMPTS.md`** for template placeholders in embedded prompts.
