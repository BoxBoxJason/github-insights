//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
)

// TestBuildIssueActivity verifies that buildIssueActivity maps issue and
// comment fields correctly.
//
//nolint:funlen,gocyclo,cyclop // table-driven test with multiple subtest cases
func TestBuildIssueActivity(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.March, 10, 12, 0, 0, 0, time.UTC)

	t.Run("populates fields from issue", func(t *testing.T) {
		t.Parallel()

		issue := &github.Issue{
			Number:    ptrOf(42),
			Title:     ptrOf("Bug: crash on startup"),
			HTMLURL:   ptrOf("https://github.com/owner/repo/issues/42"),
			User:      &github.User{Login: ptrOf("reporter")},
			Body:      ptrOf("Steps to reproduce..."),
			CreatedAt: &github.Timestamp{Time: timestamp},
			UpdatedAt: &github.Timestamp{Time: timestamp},
		}

		got := buildIssueActivity(issue, nil)

		if got.Number != 42 {
			t.Errorf("Number = %d, want 42", got.Number)
		}

		if got.Title != "Bug: crash on startup" {
			t.Errorf("Title = %q, want 'Bug: crash on startup'", got.Title)
		}

		if got.Author != "reporter" {
			t.Errorf("Author = %q, want reporter", got.Author)
		}

		if got.Body != "Steps to reproduce..." {
			t.Errorf("Body = %q", got.Body)
		}

		if got.Reactions != 0 {
			t.Errorf("Reactions = %d, want 0", got.Reactions)
		}

		if !got.CreatedAt.Equal(timestamp) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, timestamp)
		}
	})

	t.Run("counts reactions when present", func(t *testing.T) {
		t.Parallel()

		issue := &github.Issue{
			Reactions: &github.Reactions{TotalCount: ptrOf(7)},
			CreatedAt: &github.Timestamp{Time: timestamp},
			UpdatedAt: &github.Timestamp{Time: timestamp},
		}

		got := buildIssueActivity(issue, nil)

		if got.Reactions != 7 {
			t.Errorf("Reactions = %d, want 7", got.Reactions)
		}
	})

	t.Run("converts comments", func(t *testing.T) {
		t.Parallel()

		issue := &github.Issue{
			CreatedAt: &github.Timestamp{Time: timestamp},
			UpdatedAt: &github.Timestamp{Time: timestamp},
		}
		comments := []*github.IssueComment{
			{
				ID:        ptrOf(int64(1)),
				User:      &github.User{Login: ptrOf(testUserAlice)},
				Body:      ptrOf("first comment"),
				HTMLURL:   ptrOf("https://github.com/owner/repo/issues/42#issuecomment-1"),
				CreatedAt: &github.Timestamp{Time: timestamp},
			},
			{
				ID:        ptrOf(int64(2)),
				User:      &github.User{Login: ptrOf("bob")},
				Body:      ptrOf("second comment"),
				HTMLURL:   ptrOf("https://github.com/owner/repo/issues/42#issuecomment-2"),
				CreatedAt: &github.Timestamp{Time: timestamp},
			},
		}

		got := buildIssueActivity(issue, comments)

		if len(got.Comments) != 2 {
			t.Errorf("Comments len = %d, want 2", len(got.Comments))
		}

		if got.Comments[0].Author != testUserAlice {
			t.Errorf("Comments[0].Author = %q, want alice", got.Comments[0].Author)
		}
	})

	t.Run("nil comments produces empty comment slice", func(t *testing.T) {
		t.Parallel()

		issue := &github.Issue{
			CreatedAt: &github.Timestamp{Time: timestamp},
			UpdatedAt: &github.Timestamp{Time: timestamp},
		}

		got := buildIssueActivity(issue, nil)

		if len(got.Comments) != 0 {
			t.Errorf("Comments len = %d, want 0", len(got.Comments))
		}
	})
}
