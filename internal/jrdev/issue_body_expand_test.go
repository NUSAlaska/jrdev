package jrdev

import (
	"net/url"
	"strings"
	"testing"
)

func mustMeta(t *testing.T, raw string) *repoWebMeta {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &repoWebMeta{base: u}
}

func TestExpandGitHubLinksInIssueBody_sameRepoHash(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	got := expandGitHubLinksInIssueBody(meta, "See PRD at #4 and Org/Other#9.")
	want := "See PRD at https://github.com/NUSAlaska/NUS-App/issues/4 and https://github.com/Org/Other/issues/9."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandGitHubLinksInIssueBody_crossRepo(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "Track NUSAlaska/docs#12."
	got := expandGitHubLinksInIssueBody(meta, in)
	want := "Track https://github.com/NUSAlaska/docs/issues/12."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandGitHubLinksInIssueBody_markdown(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "Link to [PRD](#3) and [ticket](/NUSAlaska/NUS-App/issues/5) and ![i](https://x/y.png)."
	got := expandGitHubLinksInIssueBody(meta, in)
	if !strings.Contains(got, "https://github.com/NUSAlaska/NUS-App/issues/3") {
		t.Fatalf("missing expanded #3 link: %q", got)
	}
	if !strings.Contains(got, "https://github.com/NUSAlaska/NUS-App/issues/5") {
		t.Fatalf("missing expanded /.../issues/5: %q", got)
	}
	if !strings.Contains(got, "https://x/y.png") {
		t.Fatalf("should keep absolute image url: %q", got)
	}
}

func TestExpandGitHubLinksInIssueBody_issuesPathInMd(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "See [doc](issues/7)."
	got := expandGitHubLinksInIssueBody(meta, in)
	want := "See [doc](https://github.com/NUSAlaska/NUS-App/issues/7)."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExpandGitHubLinksInIssueBody_codeFence(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "prose #1\n```\n#2 not expanded\n```\nafter #3"
	got := expandGitHubLinksInIssueBody(meta, in)
	if !strings.Contains(got, "https://github.com/NUSAlaska/NUS-App/issues/1") {
		t.Fatalf("prose #1: %q", got)
	}
	if !strings.Contains(got, "#2 not expanded") {
		t.Fatalf("code fence should keep #2: %q", got)
	}
	if !strings.Contains(got, "https://github.com/NUSAlaska/NUS-App/issues/3") {
		t.Fatalf("after fence #3: %q", got)
	}
}

func TestExpandGitHubLinksInIssueBody_preserveAnchor(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "Jump [x](#issue-12345)"
	got := expandGitHubLinksInIssueBody(meta, in)
	if got != in {
		t.Fatalf("got %q want unchanged", got)
	}
}

func TestExpandGitHubLinksInIssueBody_headers(t *testing.T) {
	meta := mustMeta(t, "https://github.com/NUSAlaska/NUS-App")
	in := "### 2024 roadmap\n\nTicket #88"
	got := expandGitHubLinksInIssueBody(meta, in)
	if strings.Contains(got, "### https://") {
		t.Fatalf("should not expand header hashes: %q", got)
	}
	if !strings.Contains(got, "https://github.com/NUSAlaska/NUS-App/issues/88") {
		t.Fatalf("should expand ticket ref: %q", got)
	}
}
