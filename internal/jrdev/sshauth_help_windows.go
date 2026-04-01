//go:build windows

package jrdev

// sshAgentUnreachableHelp is appended when ssh-add cannot reach an agent after jrdev tries to start one.
func sshAgentUnreachableHelp() string {
	return `Fix on Windows (OpenSSH):

  1. Open PowerShell as Administrator (right-click → Run as administrator).

  2. Allow the agent to start, then start it:
       Get-Service ssh-agent | Set-Service -StartupType Manual
       Start-Service ssh-agent

     (If Start-Service fails with "Access denied", you must use an elevated shell.)

  3. In a normal terminal, load your key once:
       ssh-add ~\.ssh\id_ed25519
     (Use your real key path if different.)

  4. Run jrdev again.

  Optional: in Services (services.msc), find "OpenSSH Authentication Agent",
  set Startup type to Manual or Automatic, and start it.
`
}
