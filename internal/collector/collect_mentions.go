package collector

import (
	"context"
	"fmt"

	"github.com/google/go-github/v85/github"
	"golang.org/x/sync/errgroup"
)

// collectMentions fetches issues and PRs where the tracked user was mentioned,
// then records each mention in agg.
func (c *Collector) collectMentions(ctx context.Context, agg *Aggregator) error {
	c.logger.Debug("collecting mentions")

	query := fmt.Sprintf("mentions:%s is:open updated:%s", c.username, c.timeRangeQuery())

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

			mentions := extractMentions(c.username, details, comments)
			for i := range mentions {
				agg.AddMention(owner, repo, mentions[i])
			}

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// extractMentions scans an issue body and its comments for @username mentions.
func extractMentions(username string, issue *github.Issue, comments []*github.IssueComment) []MentionActivity {
	var mentions []MentionActivity

	if containsMention(issue.GetBody(), username) {
		mentions = append(mentions, MentionActivity{
			Title:     issue.GetTitle(),
			URL:       issue.GetHTMLURL(),
			Author:    issue.GetUser().GetLogin(),
			Body:      issue.GetBody(),
			CreatedAt: issue.GetCreatedAt().Time,
			Source:    "issue_body",
		})
	}

	for _, comment := range comments {
		if !containsMention(comment.GetBody(), username) {
			continue
		}

		mentions = append(mentions, MentionActivity{
			Title:     issue.GetTitle(),
			URL:       comment.GetHTMLURL(),
			Author:    comment.GetUser().GetLogin(),
			Body:      comment.GetBody(),
			CreatedAt: comment.GetCreatedAt().Time,
			Source:    "issue_comment",
		})
	}

	return mentions
}
