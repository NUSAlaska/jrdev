package jrdev

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

type repoWebMeta struct {
	base *url.URL
}

var (
	crossRepoIssueRef = regexp.MustCompile(`(^|[\s(>])([A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*)#(\d+)\b`)
	sameRepoIssueRef  = regexp.MustCompile(`(^|[\s(>])#(\d+)\b`)
	mdLink            = regexp.MustCompile(`(!?\[[^\]]*\])\(([^)]+)\)`)
	hashDigitsOnly    = regexp.MustCompile(`^#\d+$`)
	mdDigitsOnly      = regexp.MustCompile(`^#(\d+)$`)
)

// expandGitHubLinksInIssueBody turns GitHub shorthand and repo-relative links into absolute
// https URLs so prompts contain fetchable addresses (e.g. #123 → …/issues/123, org/repo#456).
func expandGitHubLinksInIssueBody(meta *repoWebMeta, body string) string {
	if meta == nil || meta.base == nil || body == "" {
		return body
	}
	var b strings.Builder
	inFence := false
	for i, line := range strings.Split(body, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			b.WriteString(line)
			continue
		}
		if inFence {
			b.WriteString(line)
			continue
		}
		b.WriteString(expandLineGitHubLinks(meta, line))
	}
	return b.String()
}

func expandLineGitHubLinks(meta *repoWebMeta, line string) string {
	line = expandMarkdownLinkTargets(meta, line)
	line = crossRepoIssueRef.ReplaceAllStringFunc(line, func(m string) string {
		sub := crossRepoIssueRef.FindStringSubmatch(m)
		if len(sub) != 4 {
			return m
		}
		prefix, ownerRepo, num := sub[1], sub[2], sub[3]
		parts := strings.SplitN(ownerRepo, "/", 2)
		if len(parts) != 2 {
			return m
		}
		return prefix + issueURL(meta, parts[0], parts[1], num)
	})
	line = sameRepoIssueRef.ReplaceAllStringFunc(line, func(m string) string {
		sub := sameRepoIssueRef.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		return sub[1] + issueURLSameRepo(meta, sub[2])
	})
	return line
}

func expandMarkdownLinkTargets(meta *repoWebMeta, line string) string {
	return mdLink.ReplaceAllStringFunc(line, func(full string) string {
		sub := mdLink.FindStringSubmatch(full)
		if len(sub) != 3 {
			return full
		}
		prefix, rawURL := sub[1], sub[2]
		rew := rewriteMarkdownURL(meta, strings.TrimSpace(rawURL))
		if rew == rawURL {
			return full
		}
		return prefix + "(" + rew + ")"
	})
}

func rewriteMarkdownURL(meta *repoWebMeta, raw string) string {
	if raw == "" {
		return raw
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") {
		return raw
	}
	// Same-issue anchors GitHub stores as #issue-… / #issuecomment-… — leave as-is
	if strings.HasPrefix(raw, "#") && !hashDigitsOnly.MatchString(raw) {
		return raw
	}
	if strings.HasPrefix(raw, "#") {
		sub := mdDigitsOnly.FindStringSubmatch(raw)
		if len(sub) == 2 {
			return issueURLSameRepo(meta, sub[1])
		}
	}
	// Absolute path on the same host (e.g. /org/repo/issues/5)
	if strings.HasPrefix(raw, "/") {
		ref, err := url.Parse(raw)
		if err != nil {
			return raw
		}
		return meta.base.ResolveReference(ref).String()
	}
	raw = strings.TrimPrefix(raw, "./")
	if strings.EqualFold(raw, "issues") || strings.EqualFold(raw, "issues/") {
		return raw
	}
	if after, ok := strings.CutPrefix(raw, "issues/"); ok && after != "" && isDigits(after) {
		return issueURLSameRepo(meta, after)
	}
	return raw
}

func issueURLSameRepo(meta *repoWebMeta, num string) string {
	u := *meta.base
	u.Path = "/" + path.Join(strings.Trim(meta.base.Path, "/"), "issues", num)
	return u.String()
}

func issueURL(meta *repoWebMeta, owner, repo, num string) string {
	u := *meta.base
	u.Path = "/" + path.Join(owner, repo, "issues", num)
	return u.String()
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
