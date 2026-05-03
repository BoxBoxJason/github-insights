package collector

import (
	"context"

	"github.com/google/go-github/v85/github"
)

// searchIssues runs a GitHub issue search query and returns all matching
// issues, following pagination automatically.
func (c *Collector) searchIssues(ctx context.Context, query string) ([]*github.Issue, error) {
	var all []*github.Issue

	opts := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: perPage}}

	for page := 1; ; page++ {
		opts.Page = page

		var result *github.IssuesSearchResult

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			result, resp, err = c.client.Search.Issues(ctx, query, opts)

			return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
		})
		if err != nil {
			return nil, err
		}

		all = append(all, result.Issues...)

		if len(result.Issues) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// listAllIssueComments returns all comments for an issue, following
// pagination.
//
//nolint:dupl // listAllIssueComments and listAllPRReviewComments share the same pagination structure
func (c *Collector) listAllIssueComments(ctx context.Context, owner, repo string, number int) ([]*github.IssueComment, error) {
	var all []*github.IssueComment

	opts := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: perPage}}

	for page := 1; ; page++ {
		opts.Page = page

		var comments []*github.IssueComment

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			comments, resp, err = c.client.Issues.ListComments(ctx, owner, repo, number, opts)

			return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
		})
		if err != nil {
			return nil, err
		}

		all = append(all, comments...)

		if len(comments) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// listAllPRReviews returns all reviews for a pull request, following
// pagination.
func (c *Collector) listAllPRReviews(ctx context.Context, owner, repo string, number int) ([]*github.PullRequestReview, error) {
	var all []*github.PullRequestReview

	opts := &github.ListOptions{PerPage: perPage}

	for page := 1; ; page++ {
		opts.Page = page

		var reviews []*github.PullRequestReview

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			reviews, resp, err = c.client.PullRequests.ListReviews(ctx, owner, repo, number, opts)

			return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
		})
		if err != nil {
			return nil, err
		}

		all = append(all, reviews...)

		if len(reviews) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// listAllPRReviewComments returns all review comments for a PR, following
// pagination.
//
//nolint:dupl // listAllIssueComments and listAllPRReviewComments share the same pagination structure
func (c *Collector) listAllPRReviewComments(ctx context.Context, owner, repo string, number int) ([]*github.PullRequestComment, error) {
	var all []*github.PullRequestComment

	opts := &github.PullRequestListCommentsOptions{ListOptions: github.ListOptions{PerPage: perPage}}

	for page := 1; ; page++ {
		opts.Page = page

		var comments []*github.PullRequestComment

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			comments, resp, err = c.client.PullRequests.ListComments(ctx, owner, repo, number, opts)

			return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
		})
		if err != nil {
			return nil, err
		}

		all = append(all, comments...)

		if len(comments) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// getPullRequest fetches a single pull request by number.
//
//nolint:dupl // getPullRequest and getIssue share the same single-call structure
func (c *Collector) getPullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	var pullReq *github.PullRequest

	err := c.call(ctx, func() (*github.Response, error) {
		var resp *github.Response

		var err error

		pullReq, resp, err = c.client.PullRequests.Get(ctx, owner, repo, number)

		return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
	})
	if err != nil {
		return nil, err
	}

	return pullReq, nil
}

// getIssue fetches a single issue by number.
//
//nolint:dupl // getPullRequest and getIssue share the same single-call structure
func (c *Collector) getIssue(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	var issue *github.Issue

	err := c.call(ctx, func() (*github.Response, error) {
		var resp *github.Response

		var err error

		issue, resp, err = c.client.Issues.Get(ctx, owner, repo, number)

		return resp, err //nolint:nilnil,wrapcheck // resp needed alongside err for rate-limit inspection
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}
