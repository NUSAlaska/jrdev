package jrdev

import (
	"reflect"
	"testing"
)

func TestParseIssueRefsFromGitLogMessage(t *testing.T) {
	cases := []struct {
		msg  string
		want []int
	}{
		{"", nil},
		{"fixes #12", []int{12}},
		{"Fixes #3 and closes #4", []int{3, 4}},
		{"RESOLVES: #99", []int{99}},
		{"prefix Closes  #7 suffix", []int{7}},
		{"multiline\n\nresolves #1\nFixes #2", []int{1, 2}},
		{"Fixes: #5", []int{5}},
		{"no match here", nil},
		{"Fixes gh-12", nil},
	}
	for _, tc := range cases {
		got := ParseIssueRefsFromGitLogMessage(tc.msg)
		if len(got) != len(tc.want) {
			t.Fatalf("msg %q: got %v want %v", tc.msg, got, tc.want)
		}
		if len(tc.want) > 0 && !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("msg %q: got %v want %v", tc.msg, got, tc.want)
		}
	}
}

func TestUnionSortedInts(t *testing.T) {
	got := UnionSortedInts([]int{3, 1}, []int{2, 3}, nil, []int{5})
	want := []int{1, 2, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

