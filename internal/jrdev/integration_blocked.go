package jrdev

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// IntegrationBlockedLinePrefix is a machine-readable prefix the merge agent may print when
// integration checks fail after merging; see parent PRD (issue #1) and prompt_merge.md.
const IntegrationBlockedLinePrefix = "JRDEV_INTEGRATION_BLOCKED:"

// MetaIntegrationBlockedAction is the optional `.jrdev/config.yaml` meta key for non-interactive
// default when IntegrationBlockedLinePrefix appears: "abort" or "merge" (waive and continue).
const MetaIntegrationBlockedAction = "integration_blocked_action"

// IntegrationBlockedFromStdout reports whether any line (after trimming spaces and CR) starts with
// IntegrationBlockedLinePrefix, and returns the trimmed suffix as reason.
func IntegrationBlockedFromStdout(stdout string) (blocked bool, reason string) {
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, IntegrationBlockedLinePrefix) {
			return true, strings.TrimSpace(strings.TrimPrefix(line, IntegrationBlockedLinePrefix))
		}
	}
	return false, ""
}

func metaSaysWaiveIntegration(meta map[string]any) (waive bool, ok bool) {
	if meta == nil {
		return false, false
	}
	raw, found := meta[MetaIntegrationBlockedAction]
	if !found {
		return false, false
	}
	s, okStr := raw.(string)
	if !okStr {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "abort":
		return false, true
	case "merge":
		return true, true
	default:
		return false, false
	}
}

// ResolveIntegrationBlockedDecision picks abort (return waive=false) or merge path (waive=true).
// Precedence: cfg.IntegrationBlocked ("abort"|"merge"), then meta.integration_blocked_action,
// then interactive prompt when canPrompt, then default abort.
func ResolveIntegrationBlockedDecision(cfg Config, stdin io.Reader, stdout io.Writer, log func(string, ...any), reason string, canPrompt bool) (waive bool, err error) {
	if log == nil {
		log = func(string, ...any) {}
	}
	switch strings.ToLower(strings.TrimSpace(cfg.IntegrationBlocked)) {
	case "abort":
		log("jrdev: integration blocked — proceeding per --integration-blocked=abort\n")
		return false, nil
	case "merge":
		log("jrdev: integration blocked — proceeding per --integration-blocked=merge (waiving integration for this attempt)\n")
		return true, nil
	}
	if cfg.IntegrationBlocked != "" {
		return false, fmt.Errorf("jrdev: --integration-blocked must be abort or merge, got %q", cfg.IntegrationBlocked)
	}
	if w, ok := metaSaysWaiveIntegration(cfg.Project.Meta); ok {
		if w {
			log("jrdev: integration blocked — meta.%s=merge; continuing (waive)\n", MetaIntegrationBlockedAction)
		} else {
			log("jrdev: integration blocked — meta.%s=abort; stopping\n", MetaIntegrationBlockedAction)
		}
		return w, nil
	}
	if canPrompt {
		return promptIntegrationBlockedWaiveOrAbort(stdin, stdout, reason)
	}
	log("jrdev: integration blocked — non-interactive stdin and no meta.%s; default abort\n", MetaIntegrationBlockedAction)
	return false, nil
}

func promptIntegrationBlockedWaiveOrAbort(stdin io.Reader, stdout io.Writer, reason string) (waive bool, err error) {
	fmt.Fprintf(stdout, "\njrdev: merge agent reported integration checks did not pass")
	if reason != "" {
		fmt.Fprintf(stdout, " (%s)", reason)
	}
	fmt.Fprintf(stdout, ".\n\n")
	fmt.Fprintf(stdout, "  [A]bort — stop without closing the issue; undo the agent's merge in the integration worktree if one was recorded\n")
	fmt.Fprintf(stdout, "  [M]erge — waive integration for this run and continue the merge path (close issue as usual)\n\n")
	fmt.Fprintf(stdout, "Enter A or M (default A): ")
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	s := strings.ToLower(strings.TrimSpace(line))
	switch s {
	case "m", "merge", "w", "waive":
		return true, nil
	default:
		return false, nil
	}
}
