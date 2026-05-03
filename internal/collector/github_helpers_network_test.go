//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestSearchIssues_SinglePage verifies that a single-page search result is
// returned.
func TestSearchIssues_SinglePage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", jsonHandler(jsonSearchResult(issueJSON(1, "owner", "repo"))))

	col := newTestCollector(t, mux)

	issues, err := col.searchIssues(context.Background(), "is:issue author:testuser")
	if err != nil {
		t.Fatalf("searchIssues() error = %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("searchIssues() returned %d issues, want 1", len(issues))
	}

	if issues[0].GetNumber() != 1 {
		t.Errorf("issue number = %d, want 1", issues[0].GetNumber())
	}
}

// TestSearchIssues_Empty verifies that an empty search result returns no
// issues.
func TestSearchIssues_Empty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", jsonHandler(`{"items":[],"total_count":0}`))

	col := newTestCollector(t, mux)

	issues, err := col.searchIssues(context.Background(), "is:issue author:nobody")
	if err != nil {
		t.Fatalf("searchIssues() error = %v", err)
	}

	if len(issues) != 0 {
		t.Errorf("searchIssues() returned %d issues, want 0", len(issues))
	}
}

// TestSearchIssues_Pagination verifies that multi-page results are fully
// collected.
func TestSearchIssues_Pagination(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/search/issues", func(responseWriter http.ResponseWriter, req *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.Header().Set("X-RateLimit-Remaining", "59")
		responseWriter.Header().Set("X-RateLimit-Reset", "9999999999")

		page := req.URL.Query().Get("page")
		if page == "" || page == "1" {
			// Return exactly 100 items → triggers fetch of page 2
			items := make([]string, perPage)
			for i := range items {
				items[i] = issueJSON(i+1, "owner", "repo")
			}

			_, _ = fmt.Fprintf(responseWriter, `{"items":[%s],"total_count":101}`, strings.Join(items, ","))
		} else {
			// Page 2: 1 item → stops pagination
			_, _ = fmt.Fprint(responseWriter, jsonSearchResult(issueJSON(101, "owner", "repo")))
		}
	})

	col := newTestCollector(t, mux)

	issues, err := col.searchIssues(context.Background(), "is:issue author:testuser")
	if err != nil {
		t.Fatalf("searchIssues() error = %v", err)
	}

	if len(issues) != 101 {
		t.Errorf("searchIssues() returned %d issues, want 101 (2 pages)", len(issues))
	}
}

// TestListAllIssueComments_SinglePage verifies that issue comments are
// fetched correctly.
func TestListAllIssueComments_SinglePage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/42/comments",
		jsonHandler(commentsJSON(1, "alice", "great issue")))

	col := newTestCollector(t, mux)

	comments, err := col.listAllIssueComments(context.Background(), "owner", "repo", 42)
	if err != nil {
		t.Fatalf("listAllIssueComments() error = %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("listAllIssueComments() returned %d comments, want 1", len(comments))
	}

	if comments[0].GetUser().GetLogin() != "alice" {
		t.Errorf("comment author = %q, want alice", comments[0].GetUser().GetLogin())
	}
}

// TestListAllIssueComments_Empty verifies that an empty comment list is
// handled correctly.
func TestListAllIssueComments_Empty(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/1/comments", jsonHandler(`[]`))

	col := newTestCollector(t, mux)

	comments, err := col.listAllIssueComments(context.Background(), "owner", "repo", 1)
	if err != nil {
		t.Fatalf("listAllIssueComments() error = %v", err)
	}

	if len(comments) != 0 {
		t.Errorf("listAllIssueComments() returned %d, want 0", len(comments))
	}
}

// TestListAllPRReviews_SinglePage verifies that PR reviews are fetched
// correctly.
func TestListAllPRReviews_SinglePage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/pulls/5/reviews",
		jsonHandler(reviewsJSON("reviewer", stateApproved)))

	col := newTestCollector(t, mux)

	reviews, err := col.listAllPRReviews(context.Background(), "owner", "repo", 5)
	if err != nil {
		t.Fatalf("listAllPRReviews() error = %v", err)
	}

	if len(reviews) != 1 {
		t.Fatalf("listAllPRReviews() returned %d reviews, want 1", len(reviews))
	}

	if reviews[0].GetState() != stateApproved {
		t.Errorf("review state = %q, want APPROVED", reviews[0].GetState())
	}

	if reviews[0].GetUser().GetLogin() != "reviewer" {
		t.Errorf("review author = %q, want reviewer", reviews[0].GetUser().GetLogin())
	}
}

// TestListAllPRReviewComments_SinglePage verifies that PR review comments
// are fetched correctly.
func TestListAllPRReviewComments_SinglePage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/pulls/5/comments",
		jsonHandler(commentsJSON(99, "bob", "nit: formatting")))

	col := newTestCollector(t, mux)

	comments, err := col.listAllPRReviewComments(context.Background(), "owner", "repo", 5)
	if err != nil {
		t.Fatalf("listAllPRReviewComments() error = %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("listAllPRReviewComments() returned %d, want 1", len(comments))
	}

	if comments[0].GetBody() != "nit: formatting" {
		t.Errorf("comment body = %q, want 'nit: formatting'", comments[0].GetBody())
	}
}

// TestGetPullRequest verifies that a single PR is fetched correctly.
func TestGetPullRequest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/pulls/7", jsonHandler(prJSON(7, "open")))

	col := newTestCollector(t, mux)

	pullReq, err := col.getPullRequest(context.Background(), "owner", "repo", 7)
	if err != nil {
		t.Fatalf("getPullRequest() error = %v", err)
	}

	if pullReq.GetNumber() != 7 {
		t.Errorf("pr.Number = %d, want 7", pullReq.GetNumber())
	}

	if pullReq.GetState() != "open" {
		t.Errorf("pr.State = %q, want open", pullReq.GetState())
	}
}

// TestGetIssue verifies that a single issue is fetched correctly.
func TestGetIssue(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/issues/3", jsonHandler(issueJSON(3, "owner", "repo")))

	col := newTestCollector(t, mux)

	issue, err := col.getIssue(context.Background(), "owner", "repo", 3)
	if err != nil {
		t.Fatalf("getIssue() error = %v", err)
	}

	if issue.GetNumber() != 3 {
		t.Errorf("issue.Number = %d, want 3", issue.GetNumber())
	}

	if issue.GetTitle() != "Issue 3" {
		t.Errorf("issue.Title = %q, want 'Issue 3'", issue.GetTitle())
	}
}
