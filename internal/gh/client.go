// Package gh provides GitHub API client construction and authentication
// helpers.
package gh

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/v85/github"
)

// httpTimeout is the HTTP client timeout for all GitHub API calls.
const httpTimeout = 30 * time.Second

// NewClient constructs an authenticated GitHub client using the given token.
func NewClient(token string) *github.Client {
	httpClient := &http.Client{Timeout: httpTimeout}
	ghClient := github.NewClient(httpClient).WithAuthToken(token)
	ghClient.UserAgent = "github-insights"

	return ghClient
}

// AuthenticatedUser returns the login name of the authenticated user.
func AuthenticatedUser(ctx context.Context, client *github.Client) (string, error) {
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err //nolint:wrapcheck // caller handles GitHub API errors
	}

	return user.GetLogin(), nil
}
