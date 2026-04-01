package jrdev

import "testing"

func TestLooksLikeSSHAuthFailure(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"github publickey", "git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.", true},
		{"generic publickey", "Permission denied (publickey).", true},
		{"caps", "Permission Denied (publickey).", true},
		{"no methods", "no supported authentication methods available (publickey).", true},
		{"publickey auth failed", "publickey authentication failed", true},
		{"merge conflict", "Automatic merge failed; fix conflicts and then commit", false},
		{"https 401", "fatal: Authentication failed for 'https://github.com/x/y.git'", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeSSHAuthFailure(tt.in); got != tt.want {
				t.Fatalf("looksLikeSSHAuthFailure(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
