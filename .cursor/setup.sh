#!/usr/bin/env bash
# Cloud Agent install hook for the jrdev development environment.
#
# Responsibilities (idempotent, non-interactive):
#   1. Install the Go toolchain pinned by go.mod (go 1.26) — required.
#   2. Warm the module cache and build the jrdev binary — required.
#   3. Install the Cursor `agent` CLI that jrdev drives — best effort.
#   4. Copy agent skills/rules from the agent-config manifest — best effort.
#
# Steps 3 and 4 reach the public network; when they fail the environment is
# still fully usable for building, vetting, and testing jrdev, so they only warn.
set -euo pipefail

GO_VERSION="1.26.7"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

log() { echo "jrdev-setup: $*"; }
warn() { echo "jrdev-setup: WARNING: $*" >&2; }

# Run a command as root, using sudo only when we are not already root.
run_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    sudo "$@"
  fi
}

install_go() {
  if command -v go >/dev/null 2>&1 && go version 2>/dev/null | grep -q "go${GO_VERSION} "; then
    log "Go ${GO_VERSION} already present ($(go version))"
    return 0
  fi
  log "Installing Go ${GO_VERSION}..."
  local tmp
  tmp="$(mktemp -d)"
  curl -fsSL -o "${tmp}/${GO_TARBALL}" "${GO_URL}"
  run_root rm -rf /usr/local/go
  run_root tar -C /usr/local -xzf "${tmp}/${GO_TARBALL}"
  rm -rf "${tmp}"
  # /usr/local/bin precedes /usr/bin on PATH, so these symlinks shadow any
  # older distro Go for every shell without editing profile files.
  run_root ln -sf /usr/local/go/bin/go /usr/local/bin/go
  run_root ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  hash -r
  log "Installed $(go version)"
}

build_jrdev() {
  log "Downloading Go modules..."
  go mod download
  log "Building jrdev (warms the build cache)..."
  go build ./...
  log "Build complete."
}

install_cursor_agent() {
  if command -v agent >/dev/null 2>&1; then
    log "Cursor agent CLI already on PATH ($(command -v agent))"
    return 0
  fi
  log "Installing Cursor agent CLI..."
  if ! curl -fsS https://cursor.com/install | bash; then
    warn "Cursor agent CLI install failed; 'jrdev' runs need it but build/test do not."
    return 0
  fi
  # The installer drops binaries in ~/.local/bin, which is not on PATH by
  # default; expose `agent` via /usr/local/bin so jrdev can resolve it.
  if [ -e "${HOME}/.local/bin/agent" ]; then
    run_root ln -sf "${HOME}/.local/bin/agent" /usr/local/bin/agent
  fi
  if command -v agent >/dev/null 2>&1; then
    log "Cursor agent CLI installed ($(command -v agent))"
  else
    warn "Cursor agent CLI installed under ~/.local/bin but not resolvable on PATH."
  fi
}

bootstrap_agent_config() {
  local manifest=".cursor/agent-manifest.json"
  if [ ! -f "${manifest}" ]; then
    log "No ${manifest}; skipping agent-config bootstrap."
    return 0
  fi
  if ! command -v jq >/dev/null 2>&1; then
    warn "jq not found; skipping agent-config bootstrap."
    return 0
  fi
  log "Bootstrapping agent skills/rules from ${manifest}..."
  if ! curl -fsSL https://raw.githubusercontent.com/KroniK907/agent-config/v1.0.1/scripts/bootstrap-agent.sh | bash; then
    warn "agent-config bootstrap failed; agent skills/rules may be missing."
  fi
}

install_go
build_jrdev
install_cursor_agent
bootstrap_agent_config

log "Environment ready."
