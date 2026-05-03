//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
)

// openPR returns a minimal open PullRequest for use in tests.
func openPR() *github.PullRequest {
	return &github.PullRequest{State: ptrOf("open")}
}

// TestPRStatus verifies that prStatus returns the correct status string for
// various PR states.
//
//nolint:funlen // table-driven test covering all PR status transitions
func TestPRStatus(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		pr      *github.PullRequest
		reviews []*github.PullRequestReview
		want    string
	}{
		{
			name: "merged",
			pr:   &github.PullRequest{Merged: ptrOf(true), State: ptrOf("closed")},
			want: "Merged",
		},
		{
			name: "draft",
			pr:   &github.PullRequest{State: ptrOf("open"), Draft: ptrOf(true)},
			want: "Draft",
		},
		{
			name:    "open with no reviews awaits review",
			pr:      openPR(),
			reviews: nil,
			want:    "Awaiting Review",
		},
		{
			name:    "open with empty reviews awaits review",
			pr:      openPR(),
			reviews: []*github.PullRequestReview{},
			want:    "Awaiting Review",
		},
		{
			name:    "approved",
			pr:      openPR(),
			reviews: []*github.PullRequestReview{makeReview(stateApproved, now)},
			want:    "Approved",
		},
		{
			name:    "changes requested",
			pr:      openPR(),
			reviews: []*github.PullRequestReview{makeReview(stateChangesRequested, now)},
			want:    "Changes Needed",
		},
		{
			name: "most recent review wins: approved after changes requested",
			pr:   openPR(),
			reviews: []*github.PullRequestReview{
				makeReview(stateChangesRequested, now.Add(-time.Hour)),
				makeReview(stateApproved, now),
			},
			want: "Approved",
		},
		{
			name: "most recent review wins: changes requested after approved",
			pr:   openPR(),
			reviews: []*github.PullRequestReview{
				makeReview(stateApproved, now.Add(-time.Hour)),
				makeReview(stateChangesRequested, now),
			},
			want: "Changes Needed",
		},
		{
			name:    "open with unrecognized review state falls back to Open",
			pr:      openPR(),
			reviews: []*github.PullRequestReview{makeReview("COMMENTED", now)},
			want:    "Open",
		},
		{
			name: "closed non-merged",
			pr:   &github.PullRequest{State: ptrOf("closed")},
			want: "Closed",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := prStatus(testCase.pr, testCase.reviews)

			if got != testCase.want {
				t.Errorf("prStatus() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestFilterReviewsByUser verifies that filterReviewsByUser returns only
// reviews submitted by the given username.
//
//nolint:funlen // table-driven test covering all filtering edge cases
func TestFilterReviewsByUser(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name     string
		reviews  []*github.PullRequestReview
		username string
		wantLen  int
	}{
		{
			name:     "nil reviews",
			reviews:  nil,
			username: testUserAlice,
			wantLen:  0,
		},
		{
			name:     "empty reviews",
			reviews:  []*github.PullRequestReview{},
			username: testUserAlice,
			wantLen:  0,
		},
		{
			name: "matches by login",
			reviews: []*github.PullRequestReview{
				makeReviewByUser(testUserAlice, stateApproved, now),
				makeReviewByUser("bob", stateApproved, now),
			},
			username: testUserAlice,
			wantLen:  1,
		},
		{
			name: "case insensitive match",
			reviews: []*github.PullRequestReview{
				makeReviewByUser("Alice", stateApproved, now),
			},
			username: testUserAlice,
			wantLen:  1,
		},
		{
			name: "no matching user",
			reviews: []*github.PullRequestReview{
				makeReviewByUser("bob", stateApproved, now),
			},
			username: testUserAlice,
			wantLen:  0,
		},
		{
			name:     "nil review in slice is skipped",
			reviews:  []*github.PullRequestReview{nil},
			username: testUserAlice,
			wantLen:  0,
		},
		{
			name: "multiple reviews by same user all returned",
			reviews: []*github.PullRequestReview{
				makeReviewByUser(testUserAlice, stateChangesRequested, now.Add(-time.Hour)),
				makeReviewByUser(testUserAlice, stateApproved, now),
			},
			username: testUserAlice,
			wantLen:  2,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := filterReviewsByUser(testCase.reviews, testCase.username)

			if len(got) != testCase.wantLen {
				t.Errorf("filterReviewsByUser() returned %d reviews, want %d", len(got), testCase.wantLen)
			}
		})
	}
}

// TestBuildReviews verifies that buildReviews maps PullRequestReview fields
// to ReviewActivity correctly.
func TestBuildReviews(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("nil reviews returns nil", func(t *testing.T) {
		t.Parallel()

		got := buildReviews(nil)

		if got != nil {
			t.Errorf("buildReviews(nil) = %v, want nil", got)
		}
	})

	t.Run("empty reviews returns nil", func(t *testing.T) {
		t.Parallel()

		got := buildReviews([]*github.PullRequestReview{})

		if got != nil {
			t.Errorf("buildReviews([]) = %v, want nil", got)
		}
	})

	t.Run("maps review fields correctly", func(t *testing.T) {
		t.Parallel()

		reviews := []*github.PullRequestReview{
			makeReviewByUser(testUserAlice, stateApproved, now),
		}

		got := buildReviews(reviews)

		if len(got) != 1 {
			t.Fatalf("buildReviews() len = %d, want 1", len(got))
		}

		if got[0].Author != testUserAlice {
			t.Errorf("Author = %q, want alice", got[0].Author)
		}

		if got[0].State != stateApproved {
			t.Errorf("State = %q, want APPROVED", got[0].State)
		}
	})
}
