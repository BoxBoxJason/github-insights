//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"testing"
	"time"
)

// TestNormalizeRepo verifies that normalizeRepo correctly splits owner/repo
// strings and rejects invalid forms.
//
//nolint:funlen // table-driven test with many invalid input cases
func TestNormalizeRepo(t *testing.T) {
	t.Parallel()

	col := &Collector{}

	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "valid owner/repo",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "valid with org name",
			input:     "my-org/my-project",
			wantOwner: "my-org",
			wantRepo:  "my-project",
		},
		{
			name:    "no slash",
			input:   "noslash",
			wantErr: true,
		},
		{
			name:    "too many slashes",
			input:   "owner/repo/extra",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:      "just slash",
			input:     "/",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			owner, repo, err := col.normalizeRepo(testCase.input)

			if (err != nil) != testCase.wantErr {
				t.Fatalf("normalizeRepo(%q) error = %v, wantErr %v", testCase.input, err, testCase.wantErr)
			}

			if !testCase.wantErr {
				if owner != testCase.wantOwner {
					t.Errorf("owner = %q, want %q", owner, testCase.wantOwner)
				}

				if repo != testCase.wantRepo {
					t.Errorf("repo = %q, want %q", repo, testCase.wantRepo)
				}
			}
		})
	}
}

// TestTimeRangeQuery verifies that timeRangeQuery formats the date range as
// an RFC3339 GitHub search qualifier.
func TestTimeRangeQuery(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC)

	col := &Collector{start: start, end: end}

	got := col.timeRangeQuery()

	wantStart := start.Format(time.RFC3339)
	wantEnd := end.Format(time.RFC3339)
	want := wantStart + ".." + wantEnd

	if got != want {
		t.Errorf("timeRangeQuery() = %q, want %q", got, want)
	}
}
