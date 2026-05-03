package collector

import (
	"context"
	"fmt"

	"github.com/google/go-github/v85/github"
	"golang.org/x/sync/errgroup"
)

// collectIssuesCreated fetches issues the user authored during the tracked
// time window and records them in agg.
//
//nolint:dupl // collectIssuesCreated and collectIssuesCommented share the same structure
func (c *Collector) collectIssuesCreated(ctx context.Context, agg *Aggregator) error {
	c.logger.Debug("collecting created issues")

	query := fmt.Sprintf("is:issue author:%s created:%s", c.username, c.timeRangeQuery())

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

			details, err := c.getIssue(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			comments, err := c.listAllIssueComments(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			activity := buildIssueActivity(details, comments)
			agg.AddIssueCreated(owner, repo, activity)

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// collectIssuesCommented fetches issues the user commented on during the
// tracked time window and records them in agg.
//
//nolint:dupl // collectIssuesCreated and collectIssuesCommented share the same structure
func (c *Collector) collectIssuesCommented(ctx context.Context, agg *Aggregator) error {
	c.logger.Debug("collecting commented issues")

	query := fmt.Sprintf("is:issue commenter:%s updated:%s", c.username, c.timeRangeQuery())

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

			details, err := c.getIssue(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			comments, err := c.listAllIssueComments(ctx, owner, repo, issue.GetNumber())
			if err != nil {
				return err
			}

			activity := buildIssueActivity(details, comments)
			agg.AddIssueCommented(owner, repo, activity)

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// buildIssueActivity converts a GitHub issue and its comments into an
// IssueActivity.
func buildIssueActivity(issue *github.Issue, comments []*github.IssueComment) IssueActivity {
	commentModels := make([]Comment, 0, len(comments))

	for _, comment := range comments {
		commentModels = append(commentModels, commentFromIssueComment(comment))
	}

	reactions := 0
	if issue.Reactions != nil {
		reactions = issue.Reactions.GetTotalCount()
	}

	return IssueActivity{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		URL:       issue.GetHTMLURL(),
		Author:    issue.GetUser().GetLogin(),
		Body:      issue.GetBody(),
		Reactions: reactions,
		Comments:  commentModels,
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
	}
}
