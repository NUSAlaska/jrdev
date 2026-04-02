package jrdev

import (
	"fmt"
	"regexp"
	"strings"
)

var fenceStartForTag = func(tag string) *regexp.Regexp {
	// Case-insensitive opening fence: ```tag optional whitespace
	escaped := regexp.QuoteMeta(strings.ToLower(tag))
	return regexp.MustCompile("(?i)^```\\s*" + escaped + "\\s*$")
}

// ExtractSingleMarkdownFence returns the inner text of exactly one ```tag fenced block (opening tag match is case-insensitive).
func ExtractSingleMarkdownFence(tag, agentOutput string) (inner string, err error) {
	re := fenceStartForTag(tag)
	lines := strings.Split(agentOutput, "\n")
	var starts []int
	for i, line := range lines {
		if re.MatchString(strings.TrimRight(line, "\r")) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return "", fmt.Errorf("jrdev: no %q fenced code block in agent output", tag)
	}
	if len(starts) > 1 {
		return "", fmt.Errorf("jrdev: expected exactly one %q fenced block, found %d", tag, len(starts))
	}
	startIdx := starts[0]
	var endIdx int
	foundEnd := false
	for j := startIdx + 1; j < len(lines); j++ {
		if strings.HasPrefix(strings.TrimRight(lines[j], "\r"), "```") {
			endIdx = j
			foundEnd = true
			break
		}
	}
	if !foundEnd {
		return "", fmt.Errorf("jrdev: unclosed %q fenced code block", tag)
	}
	var b strings.Builder
	for j := startIdx + 1; j < endIdx; j++ {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.TrimRight(lines[j], "\r"))
	}
	return strings.TrimSpace(b.String()), nil
}
