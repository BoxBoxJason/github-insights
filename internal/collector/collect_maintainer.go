package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// perPage is the page size used for all GitHub list API calls.
const perPage = 100

// collectMaintainerIssues fetches new issues opened in each maintained repo
// during the tracked time window and records them in agg.
func (c *Collector) collectMaintainerIssues(ctx context.Context, agg *Aggregator, repos []string) error {
	group, ctx := errgroup.WithContext(ctx)

	for _, repoFull := range repos {
		group.Go(func() error {
			owner, repo, err := c.normalizeRepo(repoFull)
			if err != nil {
				return err
			}

			c.logger.Debug("collecting maintainer issues", zap.String("repo", repoFull))

			query := fmt.Sprintf("repo:%s/%s is:issue created:%s", owner, repo, c.timeRangeQuery())

			issues, err := c.searchIssues(ctx, query)
			if err != nil {
				return err
			}

			for _, issue := range issues {
				if issue == nil {
					continue
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
				agg.AddMaintainerIssue(owner, repo, activity)
			}

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// collectReleases fetches releases published in each maintained repo
// during the tracked time window and records them in agg.
func (c *Collector) collectReleases(ctx context.Context, agg *Aggregator, repos []string) error {
	group, ctx := errgroup.WithContext(ctx)

	for _, repoFull := range repos {
		group.Go(func() error {
			owner, repo, err := c.normalizeRepo(repoFull)
			if err != nil {
				return err
			}

			c.logger.Debug("collecting releases", zap.String("repo", repoFull))

			releases, err := c.listReleases(ctx, owner, repo)
			if err != nil {
				return err
			}

			for _, release := range releases {
				timestamp := release.GetPublishedAt().Time

				if timestamp.IsZero() {
					timestamp = release.GetCreatedAt().Time
				}

				if !withinRange(timestamp, c.start, c.end) {
					continue
				}

				title := release.GetName()
				if title == "" {
					title = release.GetTagName()
				}

				business := strings.TrimSpace(title + "\n\n" + release.GetBody())

				agg.AddRelease(owner, repo, ReleaseActivity{
					Title:         title,
					URL:           release.GetHTMLURL(),
					TagName:       release.GetTagName(),
					Body:          release.GetBody(),
					BusinessValue: business,
					CreatedAt:     release.GetCreatedAt().Time,
					PublishedAt:   release.GetPublishedAt().Time,
					Draft:         release.GetDraft(),
					Prerelease:    release.GetPrerelease(),
				})
			}

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// collectTags fetches tags created in each maintained repo
// during the tracked time window and records them in agg.
func (c *Collector) collectTags(ctx context.Context, agg *Aggregator, repos []string) error {
	group, ctx := errgroup.WithContext(ctx)

	for _, repoFull := range repos {
		group.Go(func() error {
			owner, repo, err := c.normalizeRepo(repoFull)
			if err != nil {
				return err
			}

			c.logger.Debug("collecting tags", zap.String("repo", repoFull))

			tags, err := c.listTags(ctx, owner, repo)
			if err != nil {
				return err
			}

			for _, tag := range tags {
				commitDate, err := c.getTagCommitDate(ctx, owner, repo, tag)
				if err != nil {
					return err
				}

				if !withinRange(commitDate, c.start, c.end) {
					continue
				}

				tagName := tag.GetName()
				url := fmt.Sprintf("https://github.com/%s/%s/tree/%s", owner, repo, tagName)

				agg.AddTag(owner, repo, TagActivity{
					Title:     tagName,
					URL:       url,
					Name:      tagName,
					CommitSHA: tag.GetCommit().GetSHA(),
					CreatedAt: commitDate,
				})
			}

			return nil
		})
	}

	return group.Wait() //nolint:wrapcheck // errgroup aggregates goroutine errors
}

// listReleases returns all releases for the given repo, following pagination.
//
//nolint:dupl // listReleases and listTags share pagination structure
func (c *Collector) listReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error) {
	var all []*github.RepositoryRelease

	opts := &github.ListOptions{PerPage: perPage}

	for page := 1; ; page++ {
		opts.Page = page

		var releases []*github.RepositoryRelease

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			releases, resp, err = c.client.Repositories.ListReleases(ctx, owner, repo, opts)

			return resp, err //nolint:nilnil,wrapcheck // pagination callback: resp needed alongside err
		})
		if err != nil {
			return nil, err
		}

		all = append(all, releases...)

		if len(releases) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// listTags returns all tags for the given repo, following pagination.
//
//nolint:dupl // listReleases and listTags share pagination structure
func (c *Collector) listTags(ctx context.Context, owner, repo string) ([]*github.RepositoryTag, error) {
	var all []*github.RepositoryTag

	opts := &github.ListOptions{PerPage: perPage}

	for page := 1; ; page++ {
		opts.Page = page

		var tags []*github.RepositoryTag

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			tags, resp, err = c.client.Repositories.ListTags(ctx, owner, repo, opts)

			return resp, err //nolint:nilnil,wrapcheck // pagination callback: resp needed alongside err
		})
		if err != nil {
			return nil, err
		}

		all = append(all, tags...)

		if len(tags) < opts.PerPage {
			break
		}
	}

	return all, nil
}

// getTagCommitDate fetches the commit date for a given tag via its SHA.
func (c *Collector) getTagCommitDate(ctx context.Context, owner, repo string, tag *github.RepositoryTag) (time.Time, error) {
	sha := tag.GetCommit().GetSHA()
	if sha == "" {
		return time.Time{}, nil
	}

	var commit *github.RepositoryCommit

	err := c.call(ctx, func() (*github.Response, error) {
		var resp *github.Response

		var err error

		commit, resp, err = c.client.Repositories.GetCommit(ctx, owner, repo, sha, nil)

		return resp, err //nolint:nilnil,wrapcheck // pagination callback: resp needed alongside err
	})
	if err != nil {
		return time.Time{}, err
	}

	if commit.Commit != nil && commit.Commit.Committer != nil {
		return commit.Commit.Committer.GetDate().Time, nil
	}

	return time.Time{}, nil
}

// withinRange reports whether value falls within [start, end] (inclusive).
func withinRange(value, start, end time.Time) bool {
	if value.IsZero() {
		return false
	}

	if value.Before(start) {
		return false
	}

	if value.After(end) {
		return false
	}

	return true
}
