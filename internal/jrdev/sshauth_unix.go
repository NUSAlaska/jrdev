//go:build !windows

package jrdev

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

var (
	sshAuthSockRE = regexp.MustCompile(`(?m)^SSH_AUTH_SOCK=([^;]+);`)
	sshAgentPIDRE = regexp.MustCompile(`(?m)^SSH_AGENT_PID=([^;]+);`)
)

func startSSHAgentPlatform() error {
	out, err := exec.Command("ssh-agent", "-s").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-agent -s: %w\n%s", err, out)
	}
	s := string(out)
	m := sshAuthSockRE.FindStringSubmatch(s)
	if len(m) < 2 {
		return fmt.Errorf("parse ssh-agent output (SSH_AUTH_SOCK missing)\n%s", out)
	}
	if err := os.Setenv("SSH_AUTH_SOCK", m[1]); err != nil {
		return err
	}
	if m := sshAgentPIDRE.FindStringSubmatch(s); len(m) >= 2 {
		_ = os.Setenv("SSH_AGENT_PID", m[1])
	}
	return nil
}
