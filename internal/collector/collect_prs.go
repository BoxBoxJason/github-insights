package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"golang.org/x/sync/errgroup"
)

// collectAuthoredPRs fetches pull requests authored by the tracked user
// and records them in agg.
func (c *Collector) collectAuthoredPRs(ctx context.Context, agg *Aggregator) error {
	c.logger.Debug("collecting authored pull requests")

	query := fmt.Sprintf("is:pr author:%s updated:%s", c.username, c.timeRangeQuery())

	issues, err := c.searchIssues(ctx, query)
	if err != nil {
		return err
	}

	group, ctx := errgroup.WithContext(ctx)

	for _, issue := range issues {
		group.Go(func() error {
			owner, repo, err := repoFromIssue(issue)
			if err != nil {
				return err
			}

			pullReq, err := c.getPullRequest(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			reviews, err := c.listAllPRReviews(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			reviewComments, err := c.listAllPRReviewComments(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			issueComments, err := c.listAllIssueComments(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			prActivity := buildPullRequestActivity(pullReq, reviews, reviewComments, issueComments)
			agg.AddAuthoredPR(owner, repo, prActivity)

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// collectReviewedPRs fetches pull requests reviewed by the tracked user
// and records only those where the user actually left a review.
func (c *Collector) collectReviewedPRs(ctx context.Context, agg *Aggregator) error {
	c.logger.Debug("collecting reviewed pull requests")

	query := fmt.Sprintf("is:pr reviewed-by:%s updated:%s", c.username, c.timeRangeQuery())

	issues, err := c.searchIssues(ctx, query)
	if err != nil {
		return err
	}

	group, ctx := errgroup.WithContext(ctx)

	for _, issue := range issues {
		group.Go(func() error {
			owner, repo, err := repoFromIssue(issue)
			if err != nil {
				return err
			}

			pullReq, err := c.getPullRequest(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			reviews, err := c.listAllPRReviews(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			userReviews := filterReviewsByUser(reviews, c.username)
			if len(userReviews) == 0 {
				return nil
			}

			prReviewActivity := PullRequestReviewActivity{
				Number:      pullReq.GetNumber(),
				Title:       pullReq.GetTitle(),
				URL:         pullReq.GetHTMLURL(),
				Status:      prStatus(pullReq, reviews),
				Author:      pullReq.GetUser().GetLogin(),
				UserReviews: buildReviews(userReviews),
				AllReviews:  buildReviews(reviews),
				CreatedAt:   pullReq.GetCreatedAt().Time,
				UpdatedAt:   pullReq.GetUpdatedAt().Time,
			}

			agg.AddReviewedPR(owner, repo, prReviewActivity)

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// buildPullRequestActivity assembles a PullRequestActivity from its parts.
func buildPullRequestActivity(pullReq *github.PullRequest, reviews []*github.PullRequestReview, reviewComments []*github.PullRequestComment, issueComments []*github.IssueComment) PullRequestActivity {
	comments := make([]Comment, 0, len(reviewComments)+len(issueComments)+len(reviews))

	for _, comment := range issueComments {
		comments = append(comments, commentFromIssueComment(comment))
	}

	for _, comment := range reviewComments {
		comments = append(comments, commentFromReviewComment(comment))
	}

	for _, review := range reviews {
		if strings.TrimSpace(review.GetBody()) == "" {
			continue
		}

		comments = append(comments, commentFromReview(review))
	}

	reviewModels := buildReviews(reviews)
	reviewers := reviewersFromReviews(reviews)

	var mergedAt *time.Time

	if pullReq.MergedAt != nil {
		merged := pullReq.GetMergedAt().Time
		mergedAt = &merged
	}

	return PullRequestActivity{
		Number:       pullReq.GetNumber(),
		Title:        pullReq.GetTitle(),
		URL:          pullReq.GetHTMLURL(),
		Status:       prStatus(pullReq, reviews),
		Author:       pullReq.GetUser().GetLogin(),
		MergedBy:     pullReq.GetMergedBy().GetLogin(),
		Reviewers:    reviewers,
		Description:  pullReq.GetBody(),
		Additions:    pullReq.GetAdditions(),
		Deletions:    pullReq.GetDeletions(),
		ChangedFiles: pullReq.GetChangedFiles(),
		Comments:     comments,
		Reviews:      reviewModels,
		CreatedAt:    pullReq.GetCreatedAt().Time,
		UpdatedAt:    pullReq.GetUpdatedAt().Time,
		MergedAt:     mergedAt,
	}
}

// prStatus derives a human-readable status string from a pull request and
// its reviews.
func prStatus(pullReq *github.PullRequest, reviews []*github.PullRequestReview) string {
	if pullReq.GetMerged() {
		return "Merged"
	}

	if strings.EqualFold(pullReq.GetState(), "open") {
		if pullReq.GetDraft() {
			return "Draft"
		}

		if len(reviews) == 0 {
			return "Awaiting Review"
		}

		switch latestReviewState(reviews) {
		case "CHANGES_REQUESTED":
			return "Changes Needed"
		case "APPROVED":
			return "Approved"
		}

		return "Open"
	}

	return "Closed"
}

// filterReviewsByUser returns reviews submitted by the given username
// (case-insensitive).
func filterReviewsByUser(reviews []*github.PullRequestReview, username string) []*github.PullRequestReview {
	if len(reviews) == 0 {
		return nil
	}

	lowered := strings.ToLower(username)
	filtered := make([]*github.PullRequestReview, 0, len(reviews))

	for _, review := range reviews {
		if review == nil {
			continue
		}

		if strings.ToLower(review.GetUser().GetLogin()) == lowered {
			filtered = append(filtered, review)
		}
	}

	return filtered
}

// buildReviews converts a slice of GitHub reviews to the internal Review model.
func buildReviews(reviews []*github.PullRequestReview) []Review {
	if len(reviews) == 0 {
		return nil
	}

	result := make([]Review, 0, len(reviews))

	for _, review := range reviews {
		result = append(result, reviewFromPullRequestReview(review))
	}

	return result
}
