package collector

import (
	"fmt"
	"sync"
	"time"
)

// RepoActivity aggregates all tracked GitHub activity for a single
// repository.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type RepoActivity struct {
	Repo                string                      `json:"repo"`
	RepoURL             string                      `json:"repo_url"`
	Owner               string                      `json:"owner"`
	Name                string                      `json:"name"`
	Start               time.Time                   `json:"start"`
	End                 time.Time                   `json:"end"`
	GeneratedAt         time.Time                   `json:"generated_at"`
	PRsAuthored         []PullRequestActivity       `json:"prs_authored,omitempty"`
	PRsReviewed         []PullRequestReviewActivity `json:"prs_reviewed,omitempty"`
	IssuesCreated       []IssueActivity             `json:"issues_created,omitempty"`
	IssuesCommented     []IssueActivity             `json:"issues_commented,omitempty"`
	Mentions            []MentionActivity           `json:"mentions,omitempty"`
	MaintainerNewIssues []IssueActivity             `json:"maintainer_new_issues,omitempty"`
	Releases            []ReleaseActivity           `json:"releases,omitempty"`
	Tags                []TagActivity               `json:"tags,omitempty"`
}

// HasContent reports whether the activity contains at least one tracked item.
func (r *RepoActivity) HasContent() bool {
	return len(r.PRsAuthored) > 0 ||
		len(r.PRsReviewed) > 0 ||
		len(r.IssuesCreated) > 0 ||
		len(r.IssuesCommented) > 0 ||
		len(r.Mentions) > 0 ||
		len(r.MaintainerNewIssues) > 0 ||
		len(r.Releases) > 0 ||
		len(r.Tags) > 0
}

// PullRequestActivity holds details about a pull request authored by the
// tracked user.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type PullRequestActivity struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Status       string     `json:"status"`
	Author       string     `json:"author"`
	MergedBy     string     `json:"merged_by,omitempty"`
	Reviewers    []string   `json:"reviewers,omitempty"`
	Description  string     `json:"description"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	Comments     []Comment  `json:"comments,omitempty"`
	Reviews      []Review   `json:"reviews,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
}

// PullRequestReviewActivity holds details about a PR that the tracked user
// reviewed.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type PullRequestReviewActivity struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	Author      string    `json:"author"`
	UserReviews []Review  `json:"user_reviews,omitempty"`
	AllReviews  []Review  `json:"all_reviews,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// IssueActivity holds details about a GitHub issue.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type IssueActivity struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Reactions int       `json:"reactions"`
	Comments  []Comment `json:"comments,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MentionActivity holds details about a mention of the tracked user.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type MentionActivity struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
}

// ReleaseActivity holds details about a repository release.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type ReleaseActivity struct {
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	TagName       string    `json:"tag_name"`
	Body          string    `json:"body"`
	BusinessValue string    `json:"business_value"`
	CreatedAt     time.Time `json:"created_at"`
	PublishedAt   time.Time `json:"published_at"`
	Draft         bool      `json:"draft"`
	Prerelease    bool      `json:"prerelease"`
}

// TagActivity holds details about a repository tag.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type TagActivity struct {
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	CommitSHA string    `json:"commit_sha"`
	CreatedAt time.Time `json:"created_at"`
}

// Comment is a unified representation of an issue comment, PR review
// comment, or review body.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type Comment struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// Review represents a pull request review.
//
//nolint:tagliatelle // snake_case JSON tags match the established API output format
type Review struct {
	ID          int64     `json:"id"`
	Author      string    `json:"author"`
	Body        string    `json:"body"`
	URL         string    `json:"url"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// Aggregator collects repository activities from concurrent goroutines.
type Aggregator struct {
	mu    sync.Mutex
	repos map[string]*RepoActivity
	start time.Time
	end   time.Time
}

// NewAggregator creates an Aggregator for the given time window.
func NewAggregator(start, end time.Time) *Aggregator {
	return &Aggregator{
		repos: make(map[string]*RepoActivity),
		start: start,
		end:   end,
	}
}

// AddAuthoredPR records a pull request authored by the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddAuthoredPR(owner, repo string, pr PullRequestActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.PRsAuthored = append(activity.PRsAuthored, pr)
}

// AddReviewedPR records a pull request reviewed by the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddReviewedPR(owner, repo string, pr PullRequestReviewActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.PRsReviewed = append(activity.PRsReviewed, pr)
}

// AddIssueCreated records an issue created by the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddIssueCreated(owner, repo string, issue IssueActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.IssuesCreated = append(activity.IssuesCreated, issue)
}

// AddIssueCommented records an issue commented on by the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddIssueCommented(owner, repo string, issue IssueActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.IssuesCommented = append(activity.IssuesCommented, issue)
}

// AddMention records a mention of the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddMention(owner, repo string, mention MentionActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.Mentions = append(activity.Mentions, mention)
}

// AddMaintainerIssue records a new issue opened in a repo maintained by
// the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddMaintainerIssue(owner, repo string, issue IssueActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.MaintainerNewIssues = append(activity.MaintainerNewIssues, issue)
}

// AddRelease records a release published in a repo maintained by the
// tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddRelease(owner, repo string, release ReleaseActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.Releases = append(activity.Releases, release)
}

// AddTag records a tag created in a repo maintained by the tracked user.
//
//nolint:gocritic // aggregator takes activities by value for immutability
func (a *Aggregator) AddTag(owner, repo string, tag TagActivity) {
	a.mu.Lock()
	defer a.mu.Unlock()

	activity := a.getRepo(owner, repo)
	activity.Tags = append(activity.Tags, tag)
}

// Activities returns the collected activities keyed by "owner/repo".
func (a *Aggregator) Activities() map[string]*RepoActivity {
	return a.repos
}

// getRepo returns (or creates) the RepoActivity for owner/repo.
// Must be called with a.mu held.
func (a *Aggregator) getRepo(owner, repo string) *RepoActivity {
	key := fmt.Sprintf("%s/%s", owner, repo)

	activity, ok := a.repos[key]
	if ok {
		return activity
	}

	activity = &RepoActivity{
		Repo:    key,
		RepoURL: fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		Owner:   owner,
		Name:    repo,
		Start:   a.start,
		End:     a.end,
	}
	a.repos[key] = activity

	return activity
}
