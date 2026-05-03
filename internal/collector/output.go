package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// outputDirMode is the permission bits for the output directory.
const outputDirMode = 0o750

// outputFileMode is the permission bits for JSON output files.
const outputFileMode = 0o600

// WriteRepoOutputs writes each activity in activities to a JSON file under
// outputDir. Files are named after the repo slug (slash replaced with
// underscore). Empty or nil activities are skipped. Returns the count of
// files written and any error.
func WriteRepoOutputs(outputDir string, generatedAt time.Time, activities map[string]*RepoActivity) (int, error) {
	err := os.MkdirAll(outputDir, outputDirMode)
	if err != nil {
		return 0, err //nolint:wrapcheck // os error passed directly to caller
	}

	count := 0

	for _, activity := range activities {
		if activity == nil || !activity.HasContent() {
			continue
		}

		activity.GeneratedAt = generatedAt

		fileName := strings.ReplaceAll(activity.Repo, "/", "_") + ".json"
		path := filepath.Join(outputDir, fileName)

		payload, err := json.MarshalIndent(activity, "", "  ")
		if err != nil {
			return count, err //nolint:wrapcheck // json error passed directly to caller
		}

		writeErr := os.WriteFile(path, payload, outputFileMode)
		if writeErr != nil {
			return count, writeErr //nolint:wrapcheck // os error passed directly to caller
		}

		count++
	}

	return count, nil
}
