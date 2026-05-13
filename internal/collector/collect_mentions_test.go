//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
)

// TestExtractMentions verifies that extractMentions finds @username
// references in issue bodies and comments.
//
//nolint:funlen // table-driven test with comprehensive mention detection cases
func TestExtractMentions(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.May, 1, 10, 0, 0, 0, time.UTC)

	makeIssue := func(body string) *github.Issue {
		return &github.Issue{
			Title:     new("Test issue"),
			HTMLURL:   new("https://github.com/owner/repo/issues/1"),
			User:      &github.User{Login: new("author")},
			Body:      new(body),
			CreatedAt: &github.Timestamp{Time: timestamp},
		}
	}

	makeComment := func(body, login string) *github.IssueComment {
		return &github.IssueComment{
			User:      &github.User{Login: new(login)},
			Body:      new(body),
			HTMLURL:   new("https://github.com/owner/repo/issues/1#issuecomment-1"),
			CreatedAt: &github.Timestamp{Time: timestamp},
		}
	}

	tests := []struct {
		name     string
		username string
		issue    *github.Issue
		comments []*github.IssueComment
		wantLen  int
		wantSrc  []string
	}{
		{
			name:     "no mentions anywhere",
			username: testUserAlice,
			issue:    makeIssue("nothing relevant here"),
			wantLen:  0,
		},
		{
			name:     "mention in issue body",
			username: testUserAlice,
			issue:    makeIssue("hey @alice please look at this"),
			wantLen:  1,
			wantSrc:  []string{srcIssueBody},
		},
		{
			name:     "mention in comment",
			username: testUserAlice,
			issue:    makeIssue("no mention here"),
			comments: []*github.IssueComment{makeComment("cc @alice", "bob")},
			wantLen:  1,
			wantSrc:  []string{srcIssueComment},
		},
		{
			name:     "mention in both body and comment",
			username: testUserAlice,
			issue:    makeIssue("@alice please review"),
			comments: []*github.IssueComment{makeComment("@alice any update?", "bob")},
			wantLen:  2,
			wantSrc:  []string{srcIssueBody, srcIssueComment},
		},
		{
			name:     "multiple comments only some mention user",
			username: testUserAlice,
			issue:    makeIssue("no mention"),
			comments: []*github.IssueComment{
				makeComment("cc @alice", "bob"),
				makeComment("nothing here", "carol"),
				makeComment("@alice again", "dave"),
			},
			wantLen: 2,
			wantSrc: []string{srcIssueComment, srcIssueComment},
		},
		{
			name:     "case insensitive in body",
			username: testUserAlice,
			issue:    makeIssue("Hey @ALICE can you help?"),
			wantLen:  1,
		},
		{
			name:     "case insensitive in comment",
			username: testUserAlice,
			comments: []*github.IssueComment{makeComment("@ALICE please check", "bob")},
			issue:    makeIssue(""),
			wantLen:  1,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := extractMentions(testCase.username, testCase.issue, testCase.comments)

			if len(got) != testCase.wantLen {
				t.Fatalf("extractMentions() returned %d mentions, want %d", len(got), testCase.wantLen)
			}

			for i, wantSrc := range testCase.wantSrc {
				if got[i].Source != wantSrc {
					t.Errorf("mentions[%d].Source = %q, want %q", i, got[i].Source, wantSrc)
				}
			}
		})
	}
}
