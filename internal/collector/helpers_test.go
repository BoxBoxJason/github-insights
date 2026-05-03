//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
)

// Test-only constants shared across collector package test files.
const (
	testUserAlice         = "alice"
	stateApproved         = "APPROVED"
	stateChangesRequested = "CHANGES_REQUESTED"
	srcIssueBody          = "issue_body"
	srcIssueComment       = "issue_comment"
	testActivityKey       = "owner/repo"
)

// ptrOf is a generic helper for constructing pointer values in tests.
func ptrOf[T any](v T) *T { return &v }

// sliceEqual reports whether two string slices are equal.
func sliceEqual(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}

	return true
}

// makeReview builds a PullRequestReview with the given state and submission
// time.
func makeReview(state string, at time.Time) *github.PullRequestReview {
	return &github.PullRequestReview{
		State:       ptrOf(state),
		SubmittedAt: &github.Timestamp{Time: at},
	}
}

// makeReviewByUser builds a PullRequestReview with the given user login.
func makeReviewByUser(login, state string, at time.Time) *github.PullRequestReview {
	return &github.PullRequestReview{
		State:       ptrOf(state),
		SubmittedAt: &github.Timestamp{Time: at},
		User:        &github.User{Login: ptrOf(login)},
	}
}

// TestUniqueStrings verifies that uniqueStrings deduplicates, preserves
// order, and drops empty strings.
func TestUniqueStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{name: "nil input", input: nil, want: nil},
		{name: "empty slice", input: []string{}, want: nil},
		{name: "no duplicates", input: []string{"a", "b", "c"}, want: []string{"a", "b", "c"}},
		{name: "with duplicates", input: []string{"a", "b", "a", "c", "b"}, want: []string{"a", "b", "c"}},
		{name: "skips empty strings", input: []string{"a", "", "b"}, want: []string{"a", "b"}},
		{name: "all empty strings", input: []string{"", "", ""}, want: nil},
		{name: "preserves insertion order", input: []string{"z", "y", "x"}, want: []string{"z", "y", "x"}},
		{name: "single element", input: []string{"only"}, want: []string{"only"}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := uniqueStrings(testCase.input)

			if !sliceEqual(got, testCase.want) {
				t.Errorf("uniqueStrings(%v) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

// TestContainsMention verifies that containsMention detects @username
// references case-insensitively.
func TestContainsMention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		username string
		want     bool
	}{
		{name: "empty body", body: "", username: testUserAlice, want: false},
		{name: "empty username", body: "hello @alice", username: "", want: false},
		{name: "exact match", body: "hey @alice please review", username: testUserAlice, want: true},
		{name: "case insensitive body", body: "Hey @ALICE please review", username: testUserAlice, want: true},
		{name: "case insensitive username param", body: "hey @alice", username: "ALICE", want: true},
		{name: "at-sign required", body: "hey alice please review", username: testUserAlice, want: false},
		{name: "different user mentioned", body: "hey @bob", username: testUserAlice, want: false},
		{name: "mentioned in middle of text", body: "cc @alice for this", username: testUserAlice, want: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := containsMention(testCase.body, testCase.username)

			if got != testCase.want {
				t.Errorf("containsMention(%q, %q) = %v, want %v", testCase.body, testCase.username, got, testCase.want)
			}
		})
	}
}

// TestRepoFromIssue verifies that repoFromIssue extracts owner and repo from
// the issue URL fields.
//
//nolint:funlen // table-driven test with all URL edge cases
func TestRepoFromIssue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		repoURL   string
		rawURL    string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "repository_url gives owner and repo",
			repoURL:   "https://api.github.com/repos/myowner/myrepo",
			wantOwner: "myowner",
			wantRepo:  "myrepo",
		},
		{
			name:      "trailing slash in repository_url is trimmed",
			repoURL:   "https://api.github.com/repos/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "falls back to url when repository_url is empty",
			rawURL:    "https://api.github.com/repos/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "error when both urls are empty",
			wantErr: true,
		},
		{
			name:    "url with no slash triggers unexpected url error",
			rawURL:  "nodomain",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			issue := &github.Issue{}

			if testCase.repoURL != "" {
				issue.RepositoryURL = ptrOf(testCase.repoURL)
			}

			if testCase.rawURL != "" {
				issue.URL = ptrOf(testCase.rawURL)
			}

			owner, repo, err := repoFromIssue(issue)

			if (err != nil) != testCase.wantErr {
				t.Fatalf("repoFromIssue() error = %v, wantErr %v", err, testCase.wantErr)
			}

			if !testCase.wantErr {
				if owner != testCase.wantOwner {
					t.Errorf("owner = %q, want %q", owner, testCase.wantOwner)
				}

				if repo != testCase.wantRepo {
					t.Errorf("repo = %q, want %q", repo, testCase.wantRepo)
				}
			}
		})
	}
}

// TestLatestReviewState verifies that latestReviewState returns the state of
// the most recently submitted review.
func TestLatestReviewState(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)

	tests := []struct {
		name    string
		reviews []*github.PullRequestReview
		want    string
	}{
		{name: "nil slice", reviews: nil, want: ""},
		{name: "nil review in slice", reviews: []*github.PullRequestReview{nil}, want: ""},
		{
			name:    "review with nil SubmittedAt is skipped",
			reviews: []*github.PullRequestReview{{State: ptrOf(stateApproved)}},
			want:    "",
		},
		{
			name:    "single review",
			reviews: []*github.PullRequestReview{makeReview(stateApproved, now)},
			want:    stateApproved,
		},
		{
			name: "picks most recent review",
			reviews: []*github.PullRequestReview{
				makeReview(stateChangesRequested, earlier),
				makeReview(stateApproved, later),
			},
			want: stateApproved,
		},
		{
			name: "order does not matter — picks latest by time",
			reviews: []*github.PullRequestReview{
				makeReview(stateApproved, later),
				makeReview(stateChangesRequested, earlier),
			},
			want: stateApproved,
		},
		{
			name:    "state is uppercased",
			reviews: []*github.PullRequestReview{makeReview("approved", now)},
			want:    stateApproved,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := latestReviewState(testCase.reviews)

			if got != testCase.want {
				t.Errorf("latestReviewState() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestReviewersFromReviews verifies that reviewersFromReviews returns unique
// reviewer logins.
func TestReviewersFromReviews(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name    string
		reviews []*github.PullRequestReview
		want    []string
	}{
		{name: "nil", reviews: nil, want: nil},
		{
			name:    "single reviewer",
			reviews: []*github.PullRequestReview{makeReviewByUser(testUserAlice, stateApproved, now)},
			want:    []string{testUserAlice},
		},
		{
			name: "deduplicates repeated reviewer",
			reviews: []*github.PullRequestReview{
				makeReviewByUser(testUserAlice, stateApproved, now),
				makeReviewByUser("bob", stateApproved, now),
				makeReviewByUser(testUserAlice, stateChangesRequested, now),
			},
			want: []string{testUserAlice, "bob"},
		},
		{
			name:    "nil review in slice is skipped",
			reviews: []*github.PullRequestReview{nil},
			want:    nil,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := reviewersFromReviews(testCase.reviews)

			if !sliceEqual(got, testCase.want) {
				t.Errorf("reviewersFromReviews() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestCommentFromIssueComment verifies that commentFromIssueComment maps
// IssueComment fields correctly.
func TestCommentFromIssueComment(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.January, 15, 10, 0, 0, 0, time.UTC)
	comment := &github.IssueComment{
		ID:        ptrOf(int64(42)),
		User:      &github.User{Login: ptrOf(testUserAlice)},
		Body:      ptrOf("looks good"),
		HTMLURL:   ptrOf("https://github.com/owner/repo/issues/1#issuecomment-42"),
		CreatedAt: &github.Timestamp{Time: timestamp},
	}

	got := commentFromIssueComment(comment)

	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}

	if got.Author != testUserAlice {
		t.Errorf("Author = %q, want alice", got.Author)
	}

	if got.Body != "looks good" {
		t.Errorf("Body = %q, want 'looks good'", got.Body)
	}

	if got.Type != "issue_comment" {
		t.Errorf("Type = %q, want issue_comment", got.Type)
	}

	if !got.CreatedAt.Equal(timestamp) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, timestamp)
	}
}

// TestCommentFromReviewComment verifies that commentFromReviewComment maps
// PullRequestComment fields correctly.
func TestCommentFromReviewComment(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.February, 1, 12, 0, 0, 0, time.UTC)
	comment := &github.PullRequestComment{
		ID:        ptrOf(int64(99)),
		User:      &github.User{Login: ptrOf("bob")},
		Body:      ptrOf("nit: formatting"),
		HTMLURL:   ptrOf("https://github.com/owner/repo/pull/5#discussion_r99"),
		CreatedAt: &github.Timestamp{Time: timestamp},
	}

	got := commentFromReviewComment(comment)

	if got.ID != 99 {
		t.Errorf("ID = %d, want 99", got.ID)
	}

	if got.Author != "bob" {
		t.Errorf("Author = %q, want bob", got.Author)
	}

	if got.Type != "review_comment" {
		t.Errorf("Type = %q, want review_comment", got.Type)
	}

	if !got.CreatedAt.Equal(timestamp) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, timestamp)
	}
}

// TestCommentFromReview verifies that commentFromReview maps
// PullRequestReview body fields correctly.
func TestCommentFromReview(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.March, 5, 8, 0, 0, 0, time.UTC)
	review := &github.PullRequestReview{
		ID:          ptrOf(int64(7)),
		User:        &github.User{Login: ptrOf("carol")},
		Body:        ptrOf("LGTM"),
		HTMLURL:     ptrOf("https://github.com/owner/repo/pull/2#pullrequestreview-7"),
		State:       ptrOf(stateApproved),
		SubmittedAt: &github.Timestamp{Time: timestamp},
	}

	got := commentFromReview(review)

	if got.ID != 7 {
		t.Errorf("ID = %d, want 7", got.ID)
	}

	if got.Author != "carol" {
		t.Errorf("Author = %q, want carol", got.Author)
	}

	if got.Type != "review" {
		t.Errorf("Type = %q, want review", got.Type)
	}

	if !got.CreatedAt.Equal(timestamp) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, timestamp)
	}
}

// TestReviewFromPullRequestReview verifies that reviewFromPullRequestReview
// maps all review fields correctly.
func TestReviewFromPullRequestReview(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2024, time.April, 10, 14, 30, 0, 0, time.UTC)
	review := &github.PullRequestReview{
		ID:          ptrOf(int64(55)),
		User:        &github.User{Login: ptrOf("dave")},
		Body:        ptrOf("please fix the imports"),
		HTMLURL:     ptrOf("https://github.com/owner/repo/pull/3#pullrequestreview-55"),
		State:       ptrOf(stateChangesRequested),
		SubmittedAt: &github.Timestamp{Time: timestamp},
	}

	got := reviewFromPullRequestReview(review)

	if got.ID != 55 {
		t.Errorf("ID = %d, want 55", got.ID)
	}

	if got.Author != "dave" {
		t.Errorf("Author = %q, want dave", got.Author)
	}

	if got.State != stateChangesRequested {
		t.Errorf("State = %q, want CHANGES_REQUESTED", got.State)
	}

	if !got.SubmittedAt.Equal(timestamp) {
		t.Errorf("SubmittedAt = %v, want %v", got.SubmittedAt, timestamp)
	}
}
