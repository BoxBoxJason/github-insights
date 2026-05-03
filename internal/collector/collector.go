// Package collector fetches GitHub activity for a given user over a date
// range.
package collector

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v85/github"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/ptr"
)

// repoPartCount is the number of parts in a "owner/repo" slug.
const repoPartCount = 2

// Options configures the Collector.
type Options struct {
	Username string
	Start    time.Time
	End      time.Time
	Logger   *zap.Logger
}

// Collector fetches GitHub activity for a specific user and time window.
type Collector struct {
	client   *github.Client
	username string
	start    time.Time
	end      time.Time
	limiter  *RateLimiter
	logger   *zap.Logger
	apiCalls atomic.Int64
}

// New creates a Collector with the given GitHub client and options.
func New(client *github.Client, opts Options) *Collector {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Collector{
		client:   client,
		username: opts.Username,
		start:    opts.Start,
		end:      opts.End,
		limiter:  &RateLimiter{},
		logger:   logger,
	}
}

// APICallCount returns the total number of GitHub API calls made so far.
func (c *Collector) APICallCount() int64 {
	return c.apiCalls.Load()
}

// Collect gathers all tracked GitHub activity and returns it keyed by
// "owner/repo".
//
//nolint:funlen // collection setup with 8 concurrent goroutines is necessarily verbose
func (c *Collector) Collect(ctx context.Context, maintainedRepos []string) (map[string]*RepoActivity, error) {
	c.logger.Info("collecting github activity",
		zap.String("user", c.username),
		zap.Time("from", c.start),
		zap.Time("to", c.end),
	)

	repos := maintainedRepos

	if len(repos) == 0 {
		discovered, err := c.discoverMaintainedRepos(ctx)
		if err != nil {
			return nil, err
		}

		repos = discovered
	}

	if len(repos) == 0 {
		c.logger.Info("no maintained repos found; release, tag, and maintainer-issue collection skipped")
	} else {
		c.logger.Info("maintained repos", zap.Int("count", len(repos)))
	}

	agg := NewAggregator(c.start, c.end)

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		return c.collectAuthoredPRs(ctx, agg)
	})
	group.Go(func() error {
		return c.collectReviewedPRs(ctx, agg)
	})
	group.Go(func() error {
		return c.collectIssuesCreated(ctx, agg)
	})
	group.Go(func() error {
		return c.collectIssuesCommented(ctx, agg)
	})
	group.Go(func() error {
		return c.collectMentions(ctx, agg)
	})

	if len(repos) > 0 {
		pinnedRepos := repos

		group.Go(func() error {
			return c.collectMaintainerIssues(ctx, agg, pinnedRepos)
		})
		group.Go(func() error {
			return c.collectReleases(ctx, agg, pinnedRepos)
		})
		group.Go(func() error {
			return c.collectTags(ctx, agg, pinnedRepos)
		})
	}

	err := group.Wait()
	if err != nil {
		return nil, err //nolint:wrapcheck // errgroup propagates the underlying error
	}

	return agg.Activities(), nil
}

// discoverMaintainedRepos returns all repos the authenticated user has
// push/maintain/admin access to.
//
//nolint:gocyclo,cyclop // pagination with multi-condition permission filtering
func (c *Collector) discoverMaintainedRepos(ctx context.Context) ([]string, error) {
	c.logger.Debug("discovering maintained repos")

	var repos []string

	seen := make(map[string]struct{})

	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner,collaborator,organization_member",
		ListOptions: github.ListOptions{PerPage: perPage},
	}

	for page := 1; ; page++ {
		opts.Page = page

		var list []*github.Repository

		err := c.call(ctx, func() (*github.Response, error) {
			var resp *github.Response

			var err error

			list, resp, err = c.client.Repositories.ListByAuthenticatedUser(ctx, opts)

			return resp, err //nolint:nilnil,wrapcheck // pagination callback: resp is needed even when err != nil
		})
		if err != nil {
			return nil, err
		}

		for _, repo := range list {
			if repo == nil || repo.FullName == nil {
				continue
			}

			perms := repo.Permissions
			if perms == nil {
				continue
			}

			if ptr.Deref(perms.Admin, false) || ptr.Deref(perms.Maintain, false) || ptr.Deref(perms.Push, false) {
				fullName := repo.GetFullName()
				if _, ok := seen[fullName]; ok {
					continue
				}

				seen[fullName] = struct{}{}
				repos = append(repos, fullName)
			}
		}

		if len(list) < opts.PerPage {
			break
		}
	}

	return repos, nil
}

// timeRangeQuery returns a GitHub search date range string for the
// collector's window.
func (c *Collector) timeRangeQuery() string {
	return fmt.Sprintf("%s..%s", c.start.Format(time.RFC3339), c.end.Format(time.RFC3339))
}

// normalizeRepo splits "owner/repo" into its parts, returning an error for
// invalid input.
//
//nolint:gocritic // named returns would shadow outer vars
func (c *Collector) normalizeRepo(full string) (string, string, error) {
	parts := strings.Split(full, "/")
	if len(parts) != repoPartCount {
		return "", "", fmt.Errorf("invalid repo %q", full)
	}

	return parts[0], parts[1], nil
}
