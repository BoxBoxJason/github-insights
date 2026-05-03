package collector

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
)

// repoFromIssue extracts the owner and repo name from an issue's
// repository URL.
//
//nolint:gocritic // named returns would shadow outer vars
func repoFromIssue(issue *github.Issue) (string, string, error) {
	url := strings.TrimSuffix(issue.GetRepositoryURL(), "/")

	if url == "" {
		url = strings.TrimSuffix(issue.GetURL(), "/")
	}

	if url == "" {
		return "", "", errors.New("issue missing repository url")
	}

	parts := strings.Split(url, "/")
	if len(parts) < repoPartCount {
		return "", "", fmt.Errorf("unexpected repository url: %s", url)
	}

	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	return owner, repo, nil
}

// uniqueStrings returns a deduplicated copy of values, preserving insertion
// order. Empty strings are omitted.
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))

	var result []string

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

// reviewersFromReviews returns a deduplicated list of reviewer logins from
// reviews.
func reviewersFromReviews(reviews []*github.PullRequestReview) []string {
	reviewers := make([]string, 0, len(reviews))

	for _, review := range reviews {
		if review == nil {
			continue
		}

		reviewers = append(reviewers, review.GetUser().GetLogin())
	}

	return uniqueStrings(reviewers)
}

// latestReviewState returns the state of the most recently submitted
// review.
func latestReviewState(reviews []*github.PullRequestReview) string {
	var latest time.Time

	state := ""

	for _, review := range reviews {
		if review == nil || review.SubmittedAt == nil {
			continue
		}

		submitted := review.GetSubmittedAt().Time
		if submitted.After(latest) {
			latest = submitted
			state = strings.ToUpper(review.GetState())
		}
	}

	return state
}

// commentFromIssueComment converts a GitHub issue comment to the internal
// Comment model.
func commentFromIssueComment(comment *github.IssueComment) Comment {
	return Comment{
		ID:        comment.GetID(),
		Author:    comment.GetUser().GetLogin(),
		Body:      comment.GetBody(),
		URL:       comment.GetHTMLURL(),
		Type:      "issue_comment",
		CreatedAt: comment.GetCreatedAt().Time,
	}
}

// commentFromReviewComment converts a GitHub PR review comment to the
// internal Comment model.
func commentFromReviewComment(comment *github.PullRequestComment) Comment {
	return Comment{
		ID:        comment.GetID(),
		Author:    comment.GetUser().GetLogin(),
		Body:      comment.GetBody(),
		URL:       comment.GetHTMLURL(),
		Type:      "review_comment",
		CreatedAt: comment.GetCreatedAt().Time,
	}
}

// commentFromReview converts a GitHub PR review (with body) to the
// internal Comment model.
func commentFromReview(review *github.PullRequestReview) Comment {
	return Comment{
		ID:        review.GetID(),
		Author:    review.GetUser().GetLogin(),
		Body:      review.GetBody(),
		URL:       review.GetHTMLURL(),
		Type:      "review",
		CreatedAt: review.GetSubmittedAt().Time,
	}
}

// reviewFromPullRequestReview converts a GitHub PR review to the internal
// Review model.
func reviewFromPullRequestReview(review *github.PullRequestReview) Review {
	return Review{
		ID:          review.GetID(),
		Author:      review.GetUser().GetLogin(),
		Body:        review.GetBody(),
		URL:         review.GetHTMLURL(),
		State:       review.GetState(),
		SubmittedAt: review.GetSubmittedAt().Time,
	}
}

// containsMention reports whether body contains a @username mention
// (case-insensitive).
func containsMention(body, username string) bool {
	if body == "" || username == "" {
		return false
	}

	needle := "@" + strings.ToLower(username)

	return strings.Contains(strings.ToLower(body), needle)
}
