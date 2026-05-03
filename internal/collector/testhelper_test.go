//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
	"go.uber.org/zap"
)

// newTestCollector creates a Collector whose GitHub client points at the
// provided mux.
// The test server is closed automatically when the test ends.
func newTestCollector(t *testing.T, mux *http.ServeMux) *Collector {
	t.Helper()

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}

	client := github.NewClient(nil)
	client.BaseURL = baseURL
	client.UploadURL = baseURL

	return &Collector{
		client:   client,
		username: "testuser",
		start:    time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		end:      time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC),
		limiter:  &RateLimiter{},
		logger:   zap.NewNop(),
	}
}

// jsonSearchResult wraps raw issue JSON items in an IssuesSearchResult
// envelope.
func jsonSearchResult(items ...string) string {
	if len(items) == 0 {
		return `{"items":[],"total_count":0}`
	}

	return fmt.Sprintf(`{"items":[%s],"total_count":%d}`, strings.Join(items, ","), len(items))
}

// issueJSON returns a minimal JSON issue with the given number, owner, and
// repo.
//
//nolint:unparam // owner always receives "owner" in current tests but is kept for clarity
func issueJSON(number int, owner, repo string) string {
	return fmt.Sprintf(
		`{"number":%d,"repository_url":"https://api.github.com/repos/%s/%s","html_url":"https://github.com/%s/%s/issues/%d","user":{"login":"alice"},"title":"Issue %d","body":"body text","created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
		number, owner, repo, owner, repo, number, number,
	)
}

// prIssueJSON returns a minimal JSON issue (as returned by search) that
// represents a PR.
func prIssueJSON(number int, owner, repo string) string {
	return fmt.Sprintf(
		`{"number":%d,"repository_url":"https://api.github.com/repos/%s/%s","html_url":"https://github.com/%s/%s/pull/%d","user":{"login":"alice"},"title":"PR %d","pull_request":{"url":""},"created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
		number, owner, repo, owner, repo, number, number,
	)
}

// prJSON returns minimal JSON for a PullRequest.
func prJSON(number int, state string) string {
	merged := state == "merged"
	actualState := state

	if merged {
		actualState = "closed"
	}

	//nolint:gocritic // %s is intentional in JSON (not %q)
	return fmt.Sprintf(
		`{"number":%d,"title":"PR %d","state":"%s","merged":%v,"html_url":"https://github.com/owner/repo/pull/%d","user":{"login":"alice"},"body":"description","additions":5,"deletions":2,"changed_files":1,"created_at":"2024-06-01T00:00:00Z","updated_at":"2024-06-01T00:00:00Z"}`,
		number, number, actualState, merged, number,
	)
}

// reviewsJSON returns a JSON array with one PullRequestReview by the given
// user.
//
//nolint:unparam // login and state vary across test call sites
func reviewsJSON(login, state string) string {
	//nolint:gocritic // %s is intentional in JSON (not %q)
	return fmt.Sprintf(
		`[{"id":1,"user":{"login":"%s"},"body":"review body","html_url":"https://github.com/...","state":"%s","submitted_at":"2024-06-01T12:00:00Z"}]`,
		login, state,
	)
}

// commentsJSON returns a JSON array with one IssueComment.
func commentsJSON(id int64, login, body string) string {
	//nolint:gocritic // %s is intentional in JSON (not %q)
	return fmt.Sprintf(
		`[{"id":%d,"user":{"login":"%s"},"body":"%s","html_url":"https://github.com/...","created_at":"2024-06-01T00:00:00Z"}]`,
		id, login, body,
	)
}

// jsonHandler returns an [http.HandlerFunc] that writes body as JSON with
// rate-limit headers that prevent the collector's call() loop from treating
// the response as rate-limited.
func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "59")
		w.Header().Set("X-RateLimit-Reset", "9999999999")
		_, _ = fmt.Fprint(w, body)
	}
}
