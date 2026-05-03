//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
	"go.uber.org/zap"
)

// TestRateLimiter_SleepUntil verifies the context-aware sleep-until-time
// helper.
func TestRateLimiter_SleepUntil(t *testing.T) {
	t.Parallel()

	rateLimiter := &RateLimiter{}

	t.Run("zero time returns immediately", func(t *testing.T) {
		t.Parallel()

		err := rateLimiter.SleepUntil(context.Background(), time.Time{})
		if err != nil {
			t.Errorf("SleepUntil(zero) unexpected error: %v", err)
		}
	})

	t.Run("past time returns immediately", func(t *testing.T) {
		t.Parallel()

		err := rateLimiter.SleepUntil(context.Background(), time.Now().Add(-time.Hour))
		if err != nil {
			t.Errorf("SleepUntil(past) unexpected error: %v", err)
		}
	})

	t.Run("cancelled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := rateLimiter.SleepUntil(ctx, time.Now().Add(time.Hour))

		if !errors.Is(err, context.Canceled) {
			t.Errorf("SleepUntil() error = %v, want context.Canceled", err)
		}
	})
}

// TestRateLimiter_SleepFor verifies the context-aware sleep-for-duration
// helper.
func TestRateLimiter_SleepFor(t *testing.T) {
	t.Parallel()

	rateLimiter := &RateLimiter{}

	t.Run("zero delay returns immediately", func(t *testing.T) {
		t.Parallel()

		err := rateLimiter.SleepFor(context.Background(), 0)
		if err != nil {
			t.Errorf("SleepFor(0) unexpected error: %v", err)
		}
	})

	t.Run("negative delay returns immediately", func(t *testing.T) {
		t.Parallel()

		err := rateLimiter.SleepFor(context.Background(), -time.Second)
		if err != nil {
			t.Errorf("SleepFor(negative) unexpected error: %v", err)
		}
	})

	t.Run("cancelled context returns context.Canceled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := rateLimiter.SleepFor(ctx, time.Hour)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("SleepFor() error = %v, want context.Canceled", err)
		}
	})
}

// TestCall_Success verifies that a successful fn is not retried.
func TestCall_Success(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	calls := 0

	err := col.call(context.Background(), func() (*github.Response, error) {
		calls++

		return nil, nil //nolint:nilnil // test stub: no response or error
	})
	if err != nil {
		t.Errorf("call() error = %v, want nil", err)
	}

	if calls != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

// TestCall_NonRateLimitErrorReturnedImmediately verifies that non-rate-limit
// errors are not retried.
func TestCall_NonRateLimitErrorReturnedImmediately(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	sentinel := errors.New("connection refused")
	calls := 0

	err := col.call(context.Background(), func() (*github.Response, error) {
		calls++

		return nil, sentinel
	})

	if !errors.Is(err, sentinel) {
		t.Errorf("call() error = %v, want %v", err, sentinel)
	}

	if calls != 1 {
		t.Errorf("fn called %d times, want 1 (no retry for non-rate-limit errors)", calls)
	}
}

// TestCall_RespRemainingZeroRetries verifies retry when Remaining=0 with a
// set Reset time.
func TestCall_RespRemainingZeroRetries(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	calls := 0

	err := col.call(context.Background(), func() (*github.Response, error) {
		calls++

		if calls == 1 {
			return &github.Response{
				Response: &http.Response{},
				Rate: github.Rate{
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Now().Add(-time.Second)},
				},
			}, nil
		}

		return nil, nil //nolint:nilnil // test stub: no response or error on retry
	})
	if err != nil {
		t.Errorf("call() error = %v, want nil after retry", err)
	}

	if calls != 2 {
		t.Errorf("fn called %d times, want 2 (retry after remaining=0)", calls)
	}
}

// TestCall_RateLimitErrorRetries verifies retry on github.RateLimitError.
func TestCall_RateLimitErrorRetries(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	calls := 0

	err := col.call(context.Background(), func() (*github.Response, error) {
		calls++

		if calls == 1 {
			return nil, &github.RateLimitError{
				Rate:     github.Rate{Reset: github.Timestamp{Time: time.Now().Add(-time.Second)}},
				Response: &http.Response{StatusCode: http.StatusForbidden},
				Message:  "API rate limit exceeded",
			}
		}

		return nil, nil //nolint:nilnil // test stub: no response or error on retry
	})
	if err != nil {
		t.Errorf("call() error = %v, want nil after retry", err)
	}

	if calls != 2 {
		t.Errorf("fn called %d times, want 2", calls)
	}
}

// TestCall_AbuseRateLimitErrorRetries verifies retry on
// github.AbuseRateLimitError.
func TestCall_AbuseRateLimitErrorRetries(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	calls := 0
	zero := time.Duration(0)

	err := col.call(context.Background(), func() (*github.Response, error) {
		calls++

		if calls == 1 {
			return nil, &github.AbuseRateLimitError{
				Response:   &http.Response{StatusCode: http.StatusForbidden},
				Message:    "You have exceeded a secondary rate limit",
				RetryAfter: &zero,
			}
		}

		return nil, nil //nolint:nilnil // test stub: no response or error on retry
	})
	if err != nil {
		t.Errorf("call() error = %v, want nil after retry", err)
	}

	if calls != 2 {
		t.Errorf("fn called %d times, want 2", calls)
	}
}

// TestCall_RateLimitContextCancelledDuringSleep verifies ctx cancellation
// during rate-limit sleep.
func TestCall_RateLimitContextCancelledDuringSleep(t *testing.T) {
	t.Parallel()

	col := &Collector{limiter: &RateLimiter{}, logger: zap.NewNop()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := col.call(ctx, func() (*github.Response, error) {
		return nil, &github.RateLimitError{
			Rate:     github.Rate{Reset: github.Timestamp{Time: time.Now().Add(time.Hour)}},
			Response: &http.Response{StatusCode: http.StatusForbidden},
		}
	})
	if err == nil {
		t.Error("expected error when context is cancelled, got nil")
	}
}
