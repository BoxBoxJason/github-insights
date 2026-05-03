//nolint:testpackage // tests access unexported functions in the config package
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// discoveredUsername is the username written into auto-discovered config files.
const discoveredUsername = "foundme"

// TestParseDate verifies date parsing for RFC3339 and YYYY-MM-DD formats.
//
//nolint:funlen // table-driven test covering multiple date format cases
func TestParseDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		isEnd   bool
		wantErr bool
		want    time.Time
	}{
		{
			name:  "RFC3339",
			value: "2024-01-15T10:30:00Z",
			want:  time.Date(2024, time.January, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "YYYY-MM-DD start of day",
			value: "2024-01-15",
			isEnd: false,
			want:  time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "YYYY-MM-DD end of day",
			value: "2024-01-15",
			isEnd: true,
			want:  time.Date(2024, time.January, 15, 23, 59, 59, 0, time.UTC),
		},
		{
			name:  "whitespace is trimmed",
			value: "  2024-06-01  ",
			want:  time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "RFC3339 with timezone offset is normalized to UTC",
			value: "2024-03-10T08:00:00+05:30",
			want:  time.Date(2024, time.March, 10, 2, 30, 0, 0, time.UTC),
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			value:   "   ",
			wantErr: true,
		},
		{
			name:    "MM/DD/YYYY format",
			value:   "01/15/2024",
			wantErr: true,
		},
		{
			name:    "partial date YYYY-MM",
			value:   "2024-01",
			wantErr: true,
		},
		{
			name:    "invalid RFC3339",
			value:   "2024-01-15T99:00:00Z",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDate(testCase.value, testCase.isEnd)

			if (err != nil) != testCase.wantErr {
				t.Fatalf("ParseDate(%q, %v) error = %v, wantErr %v", testCase.value, testCase.isEnd, err, testCase.wantErr)
			}

			if !testCase.wantErr && !got.Equal(testCase.want) {
				t.Errorf("ParseDate(%q, %v) = %v, want %v", testCase.value, testCase.isEnd, got, testCase.want)
			}
		})
	}
}

// TestLoad_ValidYAML verifies that a complete config file is parsed
// correctly.
func TestLoad_ValidYAML(t *testing.T) {
	t.Parallel()

	content := `
username: testuser
start: "2024-01-01"
end: "2024-12-31"
output_dir: /tmp/out
token: mytoken
maintained_repos:
  - owner/repo1
  - owner/repo2
`
	path := filepath.Join(t.TempDir(), "config.yaml")

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, found, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !found {
		t.Fatal("Load() ok = false, want true")
	}

	if cfg.Username != "testuser" {
		t.Errorf("Username = %q, want %q", cfg.Username, "testuser")
	}

	if cfg.Token != "mytoken" {
		t.Errorf("Token = %q, want %q", cfg.Token, "mytoken")
	}

	if len(cfg.MaintainedRepos) != 2 {
		t.Errorf("MaintainedRepos len = %d, want 2", len(cfg.MaintainedRepos))
	}

	if cfg.MaintainedRepos[0] != "owner/repo1" {
		t.Errorf("MaintainedRepos[0] = %q, want owner/repo1", cfg.MaintainedRepos[0])
	}
}

// TestLoad_EmptyYAML verifies that an empty config file returns a zero
// RawConfig.
func TestLoad_EmptyYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	err := os.WriteFile(path, []byte(""), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, found, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !found {
		t.Fatal("Load() ok = false for empty file, want true")
	}

	if cfg.Username != "" || cfg.Token != "" || cfg.Start != "" || cfg.End != "" || len(cfg.MaintainedRepos) != 0 {
		t.Errorf("Load() = %+v, want zero RawConfig", cfg)
	}
}

// TestLoad_FileNotFound verifies that a missing file returns an error.
func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	_, _, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() expected error for missing file, got nil")
	}
}

// TestLoad_InvalidYAML verifies that invalid YAML returns an error.
func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	err := os.WriteFile(path, []byte("key: [\nbadyaml"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}

// TestLoad_NoDefaultConfig verifies that Load("") returns false when no
// default config exists.
//
//nolint:paralleltest // uses t.Chdir which is incompatible with t.Parallel
func TestLoad_NoDefaultConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg, found, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if found {
		t.Error("Load() ok = true, want false when no config files present")
	}

	if cfg.Username != "" || cfg.Token != "" || cfg.Start != "" || cfg.End != "" || len(cfg.MaintainedRepos) != 0 {
		t.Errorf("Load() returned non-zero config: %+v", cfg)
	}
}

// TestLoad_DefaultConfigDiscoveredYAML verifies that config.yaml is
// auto-discovered.
//
//nolint:paralleltest // uses t.Chdir which is incompatible with t.Parallel
func TestLoad_DefaultConfigDiscoveredYAML(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("username: "+discoveredUsername), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)

	cfg, found, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !found {
		t.Fatal("Load() ok = false, want true")
	}

	if cfg.Username != discoveredUsername {
		t.Errorf("Username = %q, want %s", cfg.Username, discoveredUsername)
	}
}

// TestLoad_DefaultConfigDiscoveredYML verifies that config.yml is
// auto-discovered.
//
//nolint:paralleltest // uses t.Chdir which is incompatible with t.Parallel
func TestLoad_DefaultConfigDiscoveredYML(t *testing.T) {
	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("username: "+discoveredUsername), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)

	cfg, found, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !found {
		t.Fatal("Load() ok = false, want true")
	}

	if cfg.Username != discoveredUsername {
		t.Errorf("Username = %q, want %s", cfg.Username, discoveredUsername)
	}
}
