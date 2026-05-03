//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"context"
	"net/http"
	"testing"
)

// Test-only constants for collect_network_test assertions.
const (
	approvedStatus   = "Approved"
	reviewerUsername = "reviewer"
)

// TestDiscoverMaintainedRepos verifies that only repos with
// push/maintain/admin perms are returned.
func TestDiscoverMaintainedRepos(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/user/repos", jsonHandler(`[
		{"full_name":"owner/repo-admin","permissions":{"admin":true,"maintain":false,"push":false}},
		{"full_name":"owner/repo-maintain","permissions":{"admin":false,"maintain":true,"push":false}},
		{"full_name":"owner/repo-push","permissions":{"admin":false,"maintain":false,"push":true}},
		{"full_name":"owner/repo-readonly","permissions":{"admin":false,"maintain":false,"push":false}}
	]`))

	col := newTestCollector(t, mux)

	repos, err := col.discoverMaintainedRepos(context.Background())
	if err != nil {
		t.Fatalf("discoverMaintainedRepos() error = %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("got %d repos, want 3 (admin + maintain + push)", len(repos))
	}

	want := map[string]bool{
		"owner/repo-admin":    true,
		"owner/repo-maintain": true,
		"owner/repo-push":     true,
	}

	for _, repoName := range repos {
		if !want[repoName] {
			t.Errorf("unexpected repo %q in result", repoName)
		}
	}
}

// TestDiscoverMaintainedRepos_NoPerms verifies that read-only repos are
// excluded.
func TestDiscoverMaintainedRepos_NoPerms(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/user/repos", jsonHandler(`[
		{"full_name":"owner/repo","permissions":{"admin":false,"maintain":false,"push":false}}
	]`))

	col := newTestCollector(t, mux)

	repos, err := col.discoverMaintainedRepos(context.Background())
	if err != nil {
		t.Fatalf("discoverMaintainedRepos() error = %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("got %d repos, want 0 for read-only repo", len(repos))
	}
}

// TestDiscoverMaintainedRepos_Empty verifies that an empty repo list returns
// no results.
func TestDiscoverMaintainedRepos_Empty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/user/repos", jsonHandler(`[]`))

	col := newTestCollector(t, mux)

	repos, err := col.discoverMaintainedRepos(context.Background())
	if err != nil {
		t.Fatalf("discoverMaintainedRepos() error = %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("got %d repos, want 0", len(repos))
	}
}

// TestDiscoverMaintainedRepos_Deduplicates verifies that duplicate full
// names are deduplicated.
func TestDiscoverMaintainedRepos_Deduplicates(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/user/repos", jsonHandler(`[
		{"full_name":"owner/repo","permissions":{"push":true}},
		{"full_name":"owner/repo","permissions":{"admin":true}}
	]`))

	col := newTestCollector(t, mux)

	repos, err := col.discoverMaintainedRepos(context.Background())
	if err != nil {
		t.Fatalf("discoverMaintainedRepos() error = %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("got %d repos, want 1 (duplicate should be de-duped)", len(repos))
	}
}

// TestCollectIssuesCreated verifies that an authored issue is collected and
// stored.
func TestCollectIssuesCreated(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(issueJSON(10, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/issues/10",
		jsonHandler(issueJSON(10, "owner", "repo")))
	mux.HandleFunc("/repos/owner/repo/issues/10/comments",
		jsonHandler(`[]`))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectIssuesCreated(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectIssuesCreated() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]
	if act == nil {
		t.Fatal("expected activity for owner/repo, not found")
	}

	if len(act.IssuesCreated) != 1 {
		t.Errorf("IssuesCreated len = %d, want 1", len(act.IssuesCreated))
	}

	if act.IssuesCreated[0].Number != 10 {
		t.Errorf("issue number = %d, want 10", act.IssuesCreated[0].Number)
	}
}

// TestCollectIssuesCreated_WithComments verifies that issue comments are
// fetched and stored.
func TestCollectIssuesCreated_WithComments(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(issueJSON(11, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/issues/11",
		jsonHandler(issueJSON(11, "owner", "repo")))
	mux.HandleFunc("/repos/owner/repo/issues/11/comments",
		jsonHandler(commentsJSON(5, "bob", "thanks for filing")))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectIssuesCreated(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectIssuesCreated() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.IssuesCreated[0].Comments) != 1 {
		t.Errorf("Comments len = %d, want 1", len(act.IssuesCreated[0].Comments))
	}
}

// TestCollectIssuesCommented verifies that commented issues are collected
// and stored.
func TestCollectIssuesCommented(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(issueJSON(20, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/issues/20",
		jsonHandler(issueJSON(20, "owner", "repo")))
	mux.HandleFunc("/repos/owner/repo/issues/20/comments",
		jsonHandler(commentsJSON(1, "testuser", "my comment")))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectIssuesCommented(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectIssuesCommented() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.IssuesCommented) != 1 {
		t.Errorf("IssuesCommented len = %d, want 1", len(act.IssuesCommented))
	}

	if len(act.IssuesCommented[0].Comments) != 1 {
		t.Errorf("Comments len = %d, want 1", len(act.IssuesCommented[0].Comments))
	}
}

// TestCollectAuthoredPRs verifies that an authored PR with reviews is
// collected.
func TestCollectAuthoredPRs(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(prIssueJSON(5, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/pulls/5",
		jsonHandler(prJSON(5, "open")))
	mux.HandleFunc("/repos/owner/repo/pulls/5/reviews",
		jsonHandler(reviewsJSON(reviewerUsername, "APPROVED")))
	mux.HandleFunc("/repos/owner/repo/pulls/5/comments",
		jsonHandler(`[]`))
	mux.HandleFunc("/repos/owner/repo/issues/5/comments",
		jsonHandler(commentsJSON(1, "alice", "nice PR")))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectAuthoredPRs(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectAuthoredPRs() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.PRsAuthored) != 1 {
		t.Fatalf("PRsAuthored len = %d, want 1", len(act.PRsAuthored))
	}

	prActivity := act.PRsAuthored[0]

	if prActivity.Number != 5 {
		t.Errorf("PR number = %d, want 5", prActivity.Number)
	}

	if prActivity.Status != approvedStatus {
		t.Errorf("PR status = %q, want %s", prActivity.Status, approvedStatus)
	}

	if len(prActivity.Reviewers) != 1 || prActivity.Reviewers[0] != reviewerUsername {
		t.Errorf("Reviewers = %v, want [%s]", prActivity.Reviewers, reviewerUsername)
	}
}

// TestCollectAuthoredPRs_Empty verifies that an empty search result produces
// no activities.
func TestCollectAuthoredPRs_Empty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(`{"items":[],"total_count":0}`))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectAuthoredPRs(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectAuthoredPRs() error = %v", err)
	}

	if len(agg.Activities()) != 0 {
		t.Errorf("expected 0 activities for empty results, got %d", len(agg.Activities()))
	}
}

// TestCollectReviewedPRs_UserHasReviews verifies that a PR reviewed by the
// user is collected.
func TestCollectReviewedPRs_UserHasReviews(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(prIssueJSON(8, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/pulls/8",
		jsonHandler(prJSON(8, "merged")))
	mux.HandleFunc("/repos/owner/repo/pulls/8/reviews",
		// testuser is the collector's username
		jsonHandler(reviewsJSON("testuser", "APPROVED")))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectReviewedPRs(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectReviewedPRs() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.PRsReviewed) != 1 {
		t.Fatalf("PRsReviewed len = %d, want 1", len(act.PRsReviewed))
	}

	if act.PRsReviewed[0].Number != 8 {
		t.Errorf("PR number = %d, want 8", act.PRsReviewed[0].Number)
	}
}

// TestCollectReviewedPRs_UserHasNoReviews verifies that PRs the user didn't
// review are skipped.
func TestCollectReviewedPRs_UserHasNoReviews(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues",
		jsonHandler(jsonSearchResult(prIssueJSON(9, "owner", "repo"))))
	mux.HandleFunc("/repos/owner/repo/pulls/9",
		jsonHandler(prJSON(9, "open")))
	mux.HandleFunc("/repos/owner/repo/pulls/9/reviews",
		// different user's review — testuser has nothing
		jsonHandler(reviewsJSON("otheruser", "APPROVED")))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectReviewedPRs(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectReviewedPRs() error = %v", err)
	}

	if len(agg.Activities()) != 0 {
		t.Errorf("expected 0 activities when user has no reviews, got %d", len(agg.Activities()))
	}
}

// TestCollectMentions_BodyMention verifies that a mention in the issue body
// is recorded in the aggregator.
func TestCollectMentions_BodyMention(t *testing.T) {
	t.Parallel()

	body := `{"number":3,"repository_url":"https://api.github.com/repos/owner/repo","html_url":"https://github.com/owner/repo/issues/3","user":{"login":"someone"},"title":"Help needed","body":"cc @testuser please","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", jsonHandler(jsonSearchResult(body)))
	mux.HandleFunc("/repos/owner/repo/issues/3", jsonHandler(
		`{"number":3,"html_url":"https://github.com/owner/repo/issues/3","user":{"login":"someone"},"title":"Help needed","body":"cc @testuser please","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
	))
	mux.HandleFunc("/repos/owner/repo/issues/3/comments", jsonHandler(`[]`))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectMentions(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectMentions() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.Mentions) != 1 {
		t.Fatalf("Mentions len = %d, want 1", len(act.Mentions))
	}

	if act.Mentions[0].Source != "issue_body" {
		t.Errorf("mention source = %q, want issue_body", act.Mentions[0].Source)
	}
}

// TestCollectMentions_CommentMention verifies that a mention in a comment
// body is recorded in the aggregator.
func TestCollectMentions_CommentMention(t *testing.T) {
	t.Parallel()

	issueBody := `{"number":4,"repository_url":"https://api.github.com/repos/owner/repo","html_url":"https://github.com/owner/repo/issues/4","user":{"login":"someone"},"title":"Help","body":"no mention","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", jsonHandler(jsonSearchResult(issueBody)))
	mux.HandleFunc("/repos/owner/repo/issues/4", jsonHandler(
		`{"number":4,"html_url":"https://github.com/owner/repo/issues/4","user":{"login":"someone"},"title":"Help","body":"no mention","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
	))
	mux.HandleFunc("/repos/owner/repo/issues/4/comments", jsonHandler(
		`[{"id":1,"user":{"login":"bob"},"body":"hey @testuser can you check this?","html_url":"https://github.com/...","created_at":"2024-06-01T00:00:00Z"}]`,
	))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectMentions(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectMentions() error = %v", err)
	}

	act := agg.Activities()[testActivityKey]

	if len(act.Mentions) != 1 {
		t.Fatalf("Mentions len = %d, want 1", len(act.Mentions))
	}

	if act.Mentions[0].Source != srcIssueComment {
		t.Errorf("mention source = %q, want issue_comment", act.Mentions[0].Source)
	}
}

// TestCollectMentions_NoMentions verifies that items without actual mentions
// are skipped.
func TestCollectMentions_NoMentions(t *testing.T) {
	t.Parallel()

	issueBody := `{"number":5,"repository_url":"https://api.github.com/repos/owner/repo","html_url":"https://github.com/owner/repo/issues/5","user":{"login":"someone"},"title":"Issue","body":"nothing to see","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`
	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", jsonHandler(jsonSearchResult(issueBody)))
	mux.HandleFunc("/repos/owner/repo/issues/5", jsonHandler(
		`{"number":5,"html_url":"https://github.com/owner/repo/issues/5","user":{"login":"someone"},"title":"Issue","body":"nothing to see","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
	))
	mux.HandleFunc("/repos/owner/repo/issues/5/comments", jsonHandler(`[]`))

	col := newTestCollector(t, mux)
	agg := NewAggregator(col.start, col.end)

	err := col.collectMentions(context.Background(), agg)
	if err != nil {
		t.Fatalf("collectMentions() error = %v", err)
	}

	if len(agg.Activities()) != 0 {
		t.Errorf("expected 0 activities when no mentions, got %d", len(agg.Activities()))
	}
}
