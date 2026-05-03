//nolint:testpackage // white-box tests require access to unexported identifiers
package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteRepoOutputs verifies file creation, JSON validity, directory
// creation, and filtering behavior.
//
//nolint:funlen,gocognit,gocyclo,cyclop // table-driven test covering all output scenarios
func TestWriteRepoOutputs(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, time.June, 1, 12, 0, 0, 0, time.UTC)

	activityWithContent := &RepoActivity{
		Repo:    testActivityKey,
		Owner:   "owner",
		Name:    "repo",
		RepoURL: "https://github.com/owner/repo",
		Start:   now.Add(-24 * time.Hour),
		End:     now,
		Tags:    []TagActivity{{Name: "v1.0"}},
	}

	activityEmpty := &RepoActivity{
		Repo:  "owner/empty",
		Owner: "owner",
		Name:  "empty",
	}

	t.Run("writes file for activity with content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			testActivityKey: activityWithContent,
		}

		count, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}

		path := filepath.Join(dir, "owner_repo.json")
		_, statErr := os.Stat(path)

		if os.IsNotExist(statErr) {
			t.Errorf("expected file %q to exist", path)
		}
	})

	t.Run("skips activity with no content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			"owner/empty": activityEmpty,
		}

		count, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		if count != 0 {
			t.Errorf("count = %d, want 0 for empty activity", count)
		}
	})

	t.Run("skips nil activity", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			"owner/nil": nil,
		}

		count, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		if count != 0 {
			t.Errorf("count = %d, want 0 for nil activity", count)
		}
	})

	t.Run("counts only activities with content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			testActivityKey: activityWithContent,
			"owner/empty":   activityEmpty,
			"owner/nil":     nil,
		}

		count, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("output file is valid JSON with correct fields", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			testActivityKey: activityWithContent,
		}

		_, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		data, readErr := os.ReadFile(filepath.Join(dir, "owner_repo.json")) //nolint:gosec // test-generated path
		if readErr != nil {
			t.Fatalf("ReadFile() error = %v", readErr)
		}

		var parsed RepoActivity

		unmarshalErr := json.Unmarshal(data, &parsed)
		if unmarshalErr != nil {
			t.Fatalf("output is not valid JSON: %v", unmarshalErr)
		}

		if parsed.Repo != testActivityKey {
			t.Errorf("parsed.Repo = %q, want owner/repo", parsed.Repo)
		}

		if !parsed.GeneratedAt.Equal(now) {
			t.Errorf("parsed.GeneratedAt = %v, want %v", parsed.GeneratedAt, now)
		}
	})

	t.Run("creates output directory if missing", func(t *testing.T) {
		t.Parallel()

		dir := filepath.Join(t.TempDir(), "deeply", "nested", "dir")
		activities := map[string]*RepoActivity{
			testActivityKey: activityWithContent,
		}

		count, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		if count != 1 {
			t.Errorf("count = %d, want 1", count)
		}
	})

	t.Run("filename replaces slash with underscore", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		activities := map[string]*RepoActivity{
			"my-org/my-repo": {
				Repo:  "my-org/my-repo",
				Owner: "my-org",
				Name:  "my-repo",
				Tags:  []TagActivity{{Name: "v2.0"}},
			},
		}

		_, err := WriteRepoOutputs(dir, now, activities)
		if err != nil {
			t.Fatalf("WriteRepoOutputs() error = %v", err)
		}

		expected := filepath.Join(dir, "my-org_my-repo.json")
		_, statErr := os.Stat(expected)

		if os.IsNotExist(statErr) {
			t.Errorf("expected file %q to exist", expected)
		}
	})
}
