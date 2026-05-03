// Package config handles loading and parsing of github-insights configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RawConfig holds the raw, unvalidated fields from a YAML config file.
//
//nolint:tagliatelle // snake_case YAML tags match configuration file conventions
type RawConfig struct {
	Username        string   `yaml:"username"`
	Start           string   `yaml:"start"`
	End             string   `yaml:"end"`
	OutputDir       string   `yaml:"output_dir"`
	MaintainedRepos []string `yaml:"maintained_repos"`
	Token           string   `yaml:"token"`
}

// Load reads the config file at path (or searches for a default config if
// path is empty). It returns the parsed config, a boolean indicating whether
// a file was found, and any error.
func Load(path string) (RawConfig, bool, error) {
	if path == "" {
		defaultPath, ok, err := findDefaultConfig()
		if err != nil {
			return RawConfig{}, false, err
		}

		if !ok {
			return RawConfig{}, false, nil
		}

		path = defaultPath
	}

	//nolint:gosec // path is validated by caller
	data, err := os.ReadFile(path)
	if err != nil {
		return RawConfig{}, false, err //nolint:wrapcheck // os error passed directly to caller
	}

	var cfg RawConfig

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return RawConfig{}, false, err //nolint:wrapcheck // yaml error passed directly to caller
	}

	return cfg, true, nil
}

// findDefaultConfig searches for a config.yaml or config.yml in the working
// directory.
//
//nolint:gocritic // named returns would shadow outer vars
func findDefaultConfig() (string, bool, error) {
	candidates := []string{"config.yaml", "config.yml"}

	for _, candidate := range candidates {
		_, statErr := os.Stat(candidate)
		if statErr == nil {
			return candidate, true, nil
		} else if !os.IsNotExist(statErr) {
			return "", false, statErr //nolint:wrapcheck // os error passed directly to caller
		}
	}

	return "", false, nil
}

// ParseDate parses a date string in RFC3339 or YYYY-MM-DD format. When
// isEnd is true and the input is a date-only value, it resolves to
// end-of-day.
func ParseDate(value string, isEnd bool) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("empty date")
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err == nil {
		return parsed.UTC(), nil
	}

	parsed, err = time.Parse(time.DateOnly, trimmed)
	if err == nil {
		year, month, day := parsed.Date()

		if isEnd {
			return time.Date(year, month, day, 23, 59, 59, 0, time.UTC), nil
		}

		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
	}

	return time.Time{}, fmt.Errorf("invalid date %q: expected RFC3339 or YYYY-MM-DD", trimmed)
}
