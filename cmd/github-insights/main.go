// Package main is the entry point for the github-insights CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/boxboxjason/github-insights/internal/collector"
	"github.com/boxboxjason/github-insights/internal/config"
	"github.com/boxboxjason/github-insights/internal/gh"
)

// defaultOutputDir is the output directory used when none is configured.
const defaultOutputDir = "out"

// main is the CLI entry point.
func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run parses flags, loads config, and orchestrates GitHub data collection.
//
//nolint:cyclop,gocyclo,funlen // flag parsing and config resolution are necessarily verbose
func run(ctx context.Context) error {
	var (
		configPath     string
		startFlag      stringFlag
		endFlag        stringFlag
		outFlag        stringFlag
		tokenFlag      stringFlag
		usernameFlag   stringFlag
		maintainedFlag stringSliceFlag
		verbose        bool
	)

	flag.StringVar(&configPath, "config", "", "Path to YAML config file (default: config.yaml if present)")
	flag.Var(&startFlag, "start", "Start date (RFC3339 or YYYY-MM-DD)")
	flag.Var(&endFlag, "end", "End date (RFC3339 or YYYY-MM-DD, defaults to now)")
	flag.Var(&outFlag, "out", "Output directory for JSON files")
	flag.Var(&tokenFlag, "token", "GitHub token (optional; overrides GITHUB_TOKEN and config file)")
	flag.Var(&usernameFlag, "username", "GitHub username to query (overrides GITHUB_USERNAME and config file)")
	flag.Var(&maintainedFlag, "maintained", "Maintained repo list (repeatable, or comma-separated)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose (debug-level) logging")
	flag.Parse()

	logger, err := buildLogger(verbose)
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}

	defer logger.Sync() //nolint:errcheck // best-effort flush on exit

	rawConfig, _, err := config.Load(configPath)
	if err != nil {
		return err //nolint:wrapcheck // config load error is self-explanatory
	}

	token := resolveToken(&rawConfig, tokenFlag)
	username := resolveUsername(&rawConfig, usernameFlag)

	startValue := rawConfig.Start
	if startFlag.IsSet {
		startValue = startFlag.Value
	}

	if startValue == "" {
		return errors.New("missing start date: set --start or config start")
	}

	startTime, err := config.ParseDate(startValue, false)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}

	endValue := rawConfig.End
	if endFlag.IsSet {
		endValue = endFlag.Value
	}

	var endTime time.Time

	if endValue == "" {
		endTime = time.Now().UTC()
	} else {
		endTime, err = config.ParseDate(endValue, true)
		if err != nil {
			return fmt.Errorf("invalid end date: %w", err)
		}
	}

	outputDir := rawConfig.OutputDir
	if outFlag.IsSet {
		outputDir = outFlag.Value
	}

	if outputDir == "" {
		outputDir = defaultOutputDir
	}

	outputDir = filepath.Clean(outputDir)

	maintainedRepos := rawConfig.MaintainedRepos
	if maintainedFlag.IsSet {
		maintainedRepos = maintainedFlag.Values
	}

	client := gh.NewClient(token)

	if username == "" {
		if token == "" {
			return errors.New("missing GitHub username: set --username, GITHUB_USERNAME env var, or config username (required when no token is provided)")
		}

		resolved, authErr := gh.AuthenticatedUser(ctx, client)
		if authErr != nil {
			return fmt.Errorf("resolve authenticated user: %w", authErr)
		}

		username = resolved
	}

	col := collector.New(client, collector.Options{
		Username: username,
		Start:    startTime,
		End:      endTime,
		Logger:   logger,
	})

	activities, err := col.Collect(ctx, maintainedRepos)
	if err != nil {
		return err //nolint:wrapcheck // collector error is self-explanatory
	}

	generatedAt := time.Now().UTC()

	files, err := collector.WriteRepoOutputs(outputDir, generatedAt, activities)
	if err != nil {
		return err //nolint:wrapcheck // output error is self-explanatory
	}

	logger.Info("wrote output files",
		zap.Int("files", files),
		zap.String("dir", outputDir),
	)
	logger.Info("github api calls", zap.Int64("total", col.APICallCount()))

	return nil
}

// buildLogger constructs a console zap logger at INFO level normally, or
// DEBUG level when verbose is true.
func buildLogger(verbose bool) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	if verbose {
		level = zapcore.DebugLevel
	}

	cfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(level),
		Encoding: "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			MessageKey:     "M",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if verbose {
		cfg.EncoderConfig.CallerKey = "C"
		cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		cfg.Development = true
	}

	return cfg.Build() //nolint:wrapcheck // zap error is descriptive enough
}

// resolveToken returns the GitHub token, with flag > env > config
// priority.
func resolveToken(raw *config.RawConfig, tokenFlag stringFlag) string {
	token := raw.Token

	if env := os.Getenv("GITHUB_TOKEN"); env != "" {
		token = env
	}

	if tokenFlag.IsSet {
		token = tokenFlag.Value
	}

	return token
}

// resolveUsername returns the GitHub username, with flag > env > config
// priority.
func resolveUsername(raw *config.RawConfig, usernameFlag stringFlag) string {
	username := raw.Username

	if env := os.Getenv("GITHUB_USERNAME"); env != "" {
		username = env
	}

	if usernameFlag.IsSet {
		username = usernameFlag.Value
	}

	return username
}

// stringFlag is a [flag.Value] that records whether it was explicitly set.
type stringFlag struct {
	Value string
	IsSet bool
}

// String returns the current string value.
func (s *stringFlag) String() string {
	return s.Value
}

// Set stores the value and marks the flag as set.
func (s *stringFlag) Set(value string) error {
	s.Value = value
	s.IsSet = true

	return nil
}

// stringSliceFlag is a [flag.Value] that accumulates comma-separated string
// values.
type stringSliceFlag struct {
	Values []string
	IsSet  bool
}

// String returns a comma-joined representation of all values.
func (s *stringSliceFlag) String() string {
	return strings.Join(s.Values, ",")
}

// Set appends the comma-split, whitespace-trimmed parts of value to the
// slice.
func (s *stringSliceFlag) Set(value string) error {
	for part := range strings.SplitSeq(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		s.Values = append(s.Values, trimmed)
	}

	s.IsSet = true

	return nil
}
