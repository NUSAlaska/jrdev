//go:build windows

package jrdev

import (
	"os/exec"
)

// startSSHAgentPlatform starts the Windows OpenSSH Authentication Agent service if it is not running.
func startSSHAgentPlatform() error {
	// Non-interactive: user may need to enable the service once (Set-Service -StartupType Manual).
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`$ErrorActionPreference='Stop'; Start-Service -Name ssh-agent`)
	_ = cmd.Run() // may fail if already running or access denied; ssh-add -l is the real check
	return nil
}
