//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"
)

// TestWithinRange verifies that withinRange correctly identifies times
// inside and outside the date range.
func TestWithinRange(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name  string
		value time.Time
		want  bool
	}{
		{
			name:  "zero value is never in range",
			value: time.Time{},
			want:  false,
		},
		{
			name:  "before start",
			value: time.Date(2023, time.December, 31, 23, 59, 59, 0, time.UTC),
			want:  false,
		},
		{
			name:  "exactly at start",
			value: start,
			want:  true,
		},
		{
			name:  "mid range",
			value: time.Date(2024, time.June, 15, 12, 0, 0, 0, time.UTC),
			want:  true,
		},
		{
			name:  "exactly at end",
			value: end,
			want:  true,
		},
		{
			name:  "after end",
			value: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC),
			want:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := withinRange(testCase.value, start, end)

			if got != testCase.want {
				t.Errorf("withinRange(%v, %v, %v) = %v, want %v", testCase.value, start, end, got, testCase.want)
			}
		})
	}
}
