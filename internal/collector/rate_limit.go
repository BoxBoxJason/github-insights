package collector

import (
	"context"
	"errors"
	"time"

	"github.com/google/go-github/v85/github"
	"go.uber.org/zap"
)

// defaultAbuseDelay is the sleep duration used when the API returns an abuse
// rate-limit error without a Retry-After header.
const defaultAbuseDelay = 30 * time.Second

// RateLimiter tracks GitHub rate-limit state and provides helpers to sleep
// until the rate limit resets.
type RateLimiter struct{}

// SleepUntil blocks until the given time, or until ctx is cancelled.
func (r *RateLimiter) SleepUntil(ctx context.Context, until time.Time) error {
	if until.IsZero() {
		return nil
	}

	if time.Now().After(until) {
		return nil
	}

	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // context error passed directly to caller
	case <-timer.C:
		return nil
	}
}

// SleepFor blocks for the given duration, or until ctx is cancelled.
func (r *RateLimiter) SleepFor(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // context error passed directly to caller
	case <-timer.C:
		return nil
	}
}

// call invokes fn and retries automatically on GitHub rate-limit errors.
// It respects ctx cancellation during any sleep between retries. Each
// invocation of fn counts as one GitHub API call.
//
//nolint:gocyclo,cyclop // rate-limit retry logic requires multiple error branches
func (c *Collector) call(ctx context.Context, fn func() (*github.Response, error)) error {
	for {
		c.apiCalls.Add(1)

		resp, err := fn()
		if err == nil {
			// Only treat Remaining==0 as rate-limited when Reset is set; a zero
			// Reset means no rate-limit headers were present in the response.
			if resp != nil && resp.Rate.Remaining == 0 && !resp.Rate.Reset.IsZero() {
				c.logger.Debug("rate limit exhausted, waiting for reset",
					zap.Time("retry_after", resp.Rate.Reset.Time),
				)

				sleepErr := c.limiter.SleepUntil(ctx, resp.Rate.Reset.Time)
				if sleepErr != nil {
					return sleepErr
				}

				continue
			}

			return nil
		}

		var rateErr *github.RateLimitError
		if errors.As(err, &rateErr) {
			c.logger.Debug("rate limit error, waiting for reset",
				zap.Time("retry_after", rateErr.Rate.Reset.Time),
			)

			sleepErr := c.limiter.SleepUntil(ctx, rateErr.Rate.Reset.Time)
			if sleepErr != nil {
				return sleepErr
			}

			continue
		}

		var abuseErr *github.AbuseRateLimitError
		if errors.As(err, &abuseErr) {
			delay := defaultAbuseDelay
			if abuseErr.RetryAfter != nil {
				delay = *abuseErr.RetryAfter
			}

			c.logger.Debug("abuse rate limit, waiting",
				zap.Duration("delay", delay),
			)

			sleepErr := c.limiter.SleepFor(ctx, delay)
			if sleepErr != nil {
				return sleepErr
			}

			continue
		}

		return err
	}
}
