//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"sync"
	"testing"
	"time"
)

// TestRepoActivityHasContent verifies that HasContent returns true when any
// activity slice is non-empty.
//
//nolint:funlen // table-driven test covering every activity field
func TestRepoActivityHasContent(t *testing.T) {
	t.Parallel()

	base := RepoActivity{Repo: testActivityKey}

	tests := []struct {
		name   string
		modify func(*RepoActivity)
		want   bool
	}{
		{
			name:   "empty activity",
			modify: func(*RepoActivity) {},
			want:   false,
		},
		{
			name:   "has authored PRs",
			modify: func(r *RepoActivity) { r.PRsAuthored = []PullRequestActivity{{Number: 1}} },
			want:   true,
		},
		{
			name:   "has reviewed PRs",
			modify: func(r *RepoActivity) { r.PRsReviewed = []PullRequestReviewActivity{{Number: 2}} },
			want:   true,
		},
		{
			name:   "has issues created",
			modify: func(r *RepoActivity) { r.IssuesCreated = []IssueActivity{{Number: 3}} },
			want:   true,
		},
		{
			name:   "has issues commented",
			modify: func(r *RepoActivity) { r.IssuesCommented = []IssueActivity{{Number: 4}} },
			want:   true,
		},
		{
			name:   "has mentions",
			modify: func(r *RepoActivity) { r.Mentions = []MentionActivity{{Title: "m"}} },
			want:   true,
		},
		{
			name:   "has maintainer issues",
			modify: func(r *RepoActivity) { r.MaintainerNewIssues = []IssueActivity{{Number: 5}} },
			want:   true,
		},
		{
			name:   "has releases",
			modify: func(r *RepoActivity) { r.Releases = []ReleaseActivity{{Title: "v1.0"}} },
			want:   true,
		},
		{
			name:   "has tags",
			modify: func(r *RepoActivity) { r.Tags = []TagActivity{{Name: "v1.0"}} },
			want:   true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			activity := base
			testCase.modify(&activity)

			got := activity.HasContent()

			if got != testCase.want {
				t.Errorf("HasContent() = %v, want %v", got, testCase.want)
			}
		})
	}
}

// TestNewAggregator verifies that NewAggregator returns a non-nil aggregator
// with no initial activities.
func TestNewAggregator(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC)

	agg := NewAggregator(start, end)

	if agg == nil {
		t.Fatal("NewAggregator() returned nil")
	}

	if len(agg.Activities()) != 0 {
		t.Errorf("fresh aggregator has %d activities, want 0", len(agg.Activities()))
	}
}

// TestAggregator_getRepoCreatesCorrectActivity verifies that the first Add
// call initializes repo metadata correctly.
func TestAggregator_getRepoCreatesCorrectActivity(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.December, 31, 0, 0, 0, 0, time.UTC)
	agg := NewAggregator(start, end)

	agg.AddIssueCreated("myowner", "myrepo", IssueActivity{Number: 1})

	activities := agg.Activities()
	act, ok := activities["myowner/myrepo"]

	if !ok {
		t.Fatal("expected activity for myowner/myrepo, not found")
	}

	if act.Repo != "myowner/myrepo" {
		t.Errorf("Repo = %q, want myowner/myrepo", act.Repo)
	}

	if act.Owner != "myowner" {
		t.Errorf("Owner = %q, want myowner", act.Owner)
	}

	if act.Name != "myrepo" {
		t.Errorf("Name = %q, want myrepo", act.Name)
	}

	if act.RepoURL != "https://github.com/myowner/myrepo" {
		t.Errorf("RepoURL = %q, want https://github.com/myowner/myrepo", act.RepoURL)
	}

	if !act.Start.Equal(start) {
		t.Errorf("Start = %v, want %v", act.Start, start)
	}

	if !act.End.Equal(end) {
		t.Errorf("End = %v, want %v", act.End, end)
	}
}

// TestAggregator_SameRepoReused verifies that multiple adds to the same repo
// merge into one activity.
func TestAggregator_SameRepoReused(t *testing.T) {
	t.Parallel()

	agg := NewAggregator(time.Now(), time.Now())

	agg.AddIssueCreated("owner", "repo", IssueActivity{Number: 1})
	agg.AddIssueCreated("owner", "repo", IssueActivity{Number: 2})

	activities := agg.Activities()

	if len(activities) != 1 {
		t.Errorf("got %d activities, want 1 (same repo should be merged)", len(activities))
	}

	if len(activities[testActivityKey].IssuesCreated) != 2 {
		t.Errorf("IssuesCreated len = %d, want 2", len(activities[testActivityKey].IssuesCreated))
	}
}

// TestAggregator_DifferentReposSeparate verifies that adds to different
// repos create separate activities.
func TestAggregator_DifferentReposSeparate(t *testing.T) {
	t.Parallel()

	agg := NewAggregator(time.Now(), time.Now())

	agg.AddIssueCreated("owner", "repo-a", IssueActivity{Number: 1})
	agg.AddIssueCreated("owner", "repo-b", IssueActivity{Number: 2})

	if len(agg.Activities()) != 2 {
		t.Errorf("got %d activities, want 2", len(agg.Activities()))
	}
}

// TestAggregator_AllAddMethods verifies that every Add method appends to the
// correct activity slice.
func TestAggregator_AllAddMethods(t *testing.T) {
	t.Parallel()

	agg := NewAggregator(time.Now(), time.Now())
	owner, repo := "o", "r"

	agg.AddAuthoredPR(owner, repo, PullRequestActivity{Number: 1})
	agg.AddReviewedPR(owner, repo, PullRequestReviewActivity{Number: 2})
	agg.AddIssueCreated(owner, repo, IssueActivity{Number: 3})
	agg.AddIssueCommented(owner, repo, IssueActivity{Number: 4})
	agg.AddMention(owner, repo, MentionActivity{Title: "m"})
	agg.AddMaintainerIssue(owner, repo, IssueActivity{Number: 5})
	agg.AddRelease(owner, repo, ReleaseActivity{Title: "v1"})
	agg.AddTag(owner, repo, TagActivity{Name: "v1"})

	act := agg.Activities()["o/r"]

	if len(act.PRsAuthored) != 1 {
		t.Errorf("PRsAuthored len = %d, want 1", len(act.PRsAuthored))
	}

	if len(act.PRsReviewed) != 1 {
		t.Errorf("PRsReviewed len = %d, want 1", len(act.PRsReviewed))
	}

	if len(act.IssuesCreated) != 1 {
		t.Errorf("IssuesCreated len = %d, want 1", len(act.IssuesCreated))
	}

	if len(act.IssuesCommented) != 1 {
		t.Errorf("IssuesCommented len = %d, want 1", len(act.IssuesCommented))
	}

	if len(act.Mentions) != 1 {
		t.Errorf("Mentions len = %d, want 1", len(act.Mentions))
	}

	if len(act.MaintainerNewIssues) != 1 {
		t.Errorf("MaintainerNewIssues len = %d, want 1", len(act.MaintainerNewIssues))
	}

	if len(act.Releases) != 1 {
		t.Errorf("Releases len = %d, want 1", len(act.Releases))
	}

	if len(act.Tags) != 1 {
		t.Errorf("Tags len = %d, want 1", len(act.Tags))
	}
}

// TestAggregator_ConcurrentAdds verifies that concurrent AddIssueCreated
// calls are race-condition free.
func TestAggregator_ConcurrentAdds(t *testing.T) {
	t.Parallel()

	agg := NewAggregator(time.Now(), time.Now())

	const goroutines = 50

	var waitGroup sync.WaitGroup

	waitGroup.Add(goroutines)

	for i := range goroutines {
		go func(n int) {
			defer waitGroup.Done()

			agg.AddIssueCreated("owner", "repo", IssueActivity{Number: n})
		}(i)
	}

	waitGroup.Wait()

	act := agg.Activities()[testActivityKey]

	if len(act.IssuesCreated) != goroutines {
		t.Errorf("IssuesCreated len = %d, want %d", len(act.IssuesCreated), goroutines)
	}
}
