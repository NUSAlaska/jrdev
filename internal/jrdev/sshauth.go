package jrdev

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// looksLikeSSHAuthFailure reports whether combined git/SSH output suggests SSH public-key auth failed.
func looksLikeSSHAuthFailure(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "permission denied (publickey)") ||
		strings.Contains(s, "no supported authentication methods available (publickey)") ||
		strings.Contains(s, "publickey authentication failed")
}

// sshAgentReachable reports whether ssh-add can talk to the current agent (including when it has no keys).
// OpenSSH: exit 0 = keys listed, exit 1 often = agent empty, exit 2 = cannot connect to agent.
func sshAgentReachable() bool {
	c := exec.Command("ssh-add", "-l")
	_, err := c.CombinedOutput()
	if err == nil {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		switch ee.ExitCode() {
		case 1:
			return true // typical when agent has no identities
		case 2:
			return false
		}
	}
	return false
}

// trySSHInteractiveRecovery ensures an ssh-agent is available and runs ssh-add attached to the terminal.
func trySSHInteractiveRecovery(log func(string, ...any)) error {
	if log == nil {
		log = func(string, ...any) {}
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("stdin is not a terminal; run ssh-add manually or start ssh-agent before jrdev")
	}
	if !sshAgentReachable() {
		if err := startSSHAgentPlatform(); err != nil {
			return err
		}
		if !sshAgentReachable() {
			return fmt.Errorf("ssh-agent is still unreachable after jrdev tried to start it.\n\n%s", sshAgentUnreachableHelp())
		}
	}
	log("jrdev: SSH authentication failed; running ssh-add (enter your key passphrase if prompted).\n")
	cmd := exec.Command("ssh-add")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
