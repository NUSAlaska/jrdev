//go:build !windows

package jrdev

// sshAgentUnreachableHelp is appended when ssh-add cannot reach an agent after jrdev tries to start one.
func sshAgentUnreachableHelp() string {
	return `Fix on this system:

  1. Start an agent and load your key in the same terminal before jrdev, for example:
       eval "$(ssh-agent -s)"
       ssh-add ~/.ssh/id_ed25519

  2. Or ensure your desktop session already runs ssh-agent (many distros do) and run:
       ssh-add ~/.ssh/id_ed25519

  3. Run jrdev again from that terminal so SSH_AUTH_SOCK is inherited.
`
}
