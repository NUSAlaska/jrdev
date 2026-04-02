package jrdev

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func execGit(workDir string, args ...string) *exec.Cmd {
	c := exec.Command("git", append([]string{"-C", workDir}, args...)...)
	return c
}

var issueCloseKeywordsRe = regexp.MustCompile(`(?i)\b(?:fixes|closes|resolves)\s*:?\s*#\s*(\d+)\b`)

// ParseIssueRefsFromGitLogMessage extracts GitHub issue numbers from a single commit message
// (subject + body) using Fixes|Closes|Resolves #<n> patterns (case-insensitive).
func ParseIssueRefsFromGitLogMessage(msg string) []int {
	seen := map[int]struct{}{}
	for _, m := range issueCloseKeywordsRe.FindAllStringSubmatch(msg, -1) {
		if len(m) < 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 {
			continue
		}
		seen[n] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// GitLogMessagesInRange returns commit messages (format %B) for base..HEAD in workDir.
func GitLogMessagesInRange(workDir, base string) (string, error) {
	if strings.TrimSpace(base) == "" {
		base = "origin/main"
	}
	rng := fmt.Sprintf("%s..HEAD", base)
	c := execGit(workDir, "log", rng, "--format=%B", "--no-merges")
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log %s: %w\n%s", rng, err, out)
	}
	return string(out), nil
}

// ParseIssueRefsFromGitLogRange parses all issue references from git log base..HEAD in workDir.
func ParseIssueRefsFromGitLogRange(workDir, base string) ([]int, error) {
	text, err := GitLogMessagesInRange(workDir, base)
	if err != nil {
		return nil, err
	}
	return ParseIssueRefsFromGitLogMessage(text), nil
}

// GitCurrentBranch returns the current branch short name in workDir (or "HEAD" if detached).
func GitCurrentBranch(workDir string) (string, error) {
	c := execGit(workDir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// PRClosingIssueNumbersFromGH returns issue numbers linked as closing references from
// gh pr view for the given head branch. Only call when the PR exists — on typical
// "no PR" errors this returns (nil, nil).
func PRClosingIssueNumbersFromGH(cfg Config, headBranch string) ([]int, error) {
	headBranch = strings.TrimSpace(headBranch)
	if headBranch == "" || headBranch == "HEAD" {
		return nil, nil
	}
	out, err := ghCmd(cfg, "pr", "view", headBranch, "--json", "closingIssuesReferences").CombinedOutput()
	if err != nil {
		s := strings.ToLower(string(out))
		if strings.Contains(s, "no pull requests found") ||
			strings.Contains(s, "could not find pull request") ||
			strings.Contains(s, "not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("gh pr view %q: %w\n%s", headBranch, err, out)
	}
	var v struct {
		ClosingIssuesReferences *struct {
			Nodes []struct {
				Number int `json:"number"`
			} `json:"nodes"`
		} `json:"closingIssuesReferences"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("gh pr view json: %w", err)
	}
	if v.ClosingIssuesReferences == nil {
		return nil, nil
	}
	seen := map[int]struct{}{}
	for _, node := range v.ClosingIssuesReferences.Nodes {
		if node.Number > 0 {
			seen[node.Number] = struct{}{}
		}
	}
	outN := make([]int, 0, len(seen))
	for n := range seen {
		outN = append(outN, n)
	}
	sort.Ints(outN)
	return outN, nil
}

// UnionSortedInts returns the sorted unique union of all slices.
func UnionSortedInts(slices ...[]int) []int {
	seen := map[int]struct{}{}
	for _, s := range slices {
		for _, n := range s {
			if n > 0 {
				seen[n] = struct{}{}
			}
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}
