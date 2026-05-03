package main

import (
	"testing"

	"github.com/boxboxjason/github-insights/internal/config"
)

// TestResolveToken verifies that resolveToken respects flag > env > config
// priority.
func TestResolveToken(t *testing.T) {
	tests := []struct {
		name      string
		rawToken  string
		envToken  string
		flagToken stringFlag
		want      string
	}{
		{
			name:     "config token used when no env or flag",
			rawToken: "config-token",
			want:     "config-token",
		},
		{
			name:     "env overrides config",
			rawToken: "config-token",
			envToken: "env-token",
			want:     "env-token",
		},
		{
			name:      "flag overrides env and config",
			rawToken:  "config-token",
			envToken:  "env-token",
			flagToken: stringFlag{Value: "flag-token", IsSet: true},
			want:      "flag-token",
		},
		{
			name:      "flag not set does not override env",
			rawToken:  "config-token",
			envToken:  "env-token",
			flagToken: stringFlag{Value: "", IsSet: false},
			want:      "env-token",
		},
		{
			name: "all empty returns empty",
			want: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.envToken != "" {
				t.Setenv("GITHUB_TOKEN", testCase.envToken)
			} else {
				t.Setenv("GITHUB_TOKEN", "")
			}

			raw := config.RawConfig{Token: testCase.rawToken}
			got := resolveToken(&raw, testCase.flagToken)

			if got != testCase.want {
				t.Errorf("resolveToken() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestResolveUsername verifies that resolveUsername respects flag > env >
// config priority.
func TestResolveUsername(t *testing.T) {
	tests := []struct {
		name        string
		rawUsername string
		envUsername string
		flagUser    stringFlag
		want        string
	}{
		{
			name:        "config username used when no env or flag",
			rawUsername: "config-user",
			want:        "config-user",
		},
		{
			name:        "env overrides config",
			rawUsername: "config-user",
			envUsername: "env-user",
			want:        "env-user",
		},
		{
			name:        "flag overrides env and config",
			rawUsername: "config-user",
			envUsername: "env-user",
			flagUser:    stringFlag{Value: "flag-user", IsSet: true},
			want:        "flag-user",
		},
		{
			name:        "flag not set does not override env",
			rawUsername: "config-user",
			envUsername: "env-user",
			flagUser:    stringFlag{Value: "", IsSet: false},
			want:        "env-user",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.envUsername != "" {
				t.Setenv("GITHUB_USERNAME", testCase.envUsername)
			} else {
				t.Setenv("GITHUB_USERNAME", "")
			}

			raw := config.RawConfig{Username: testCase.rawUsername}
			got := resolveUsername(&raw, testCase.flagUser)

			if got != testCase.want {
				t.Errorf("resolveUsername() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestStringFlag verifies the stringFlag value type.
func TestStringFlag(t *testing.T) {
	t.Parallel()

	t.Run("IsSet is false before Set is called", func(t *testing.T) {
		t.Parallel()

		var flag stringFlag

		if flag.IsSet {
			t.Error("IsSet should be false before Set()")
		}
	})

	t.Run("Set stores value and marks IsSet", func(t *testing.T) {
		t.Parallel()

		var flag stringFlag

		err := flag.Set("hello")
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if !flag.IsSet {
			t.Error("IsSet = false, want true after Set()")
		}

		if flag.Value != "hello" {
			t.Errorf("Value = %q, want hello", flag.Value)
		}
	})

	t.Run("String returns current value", func(t *testing.T) {
		t.Parallel()

		flag := stringFlag{Value: "myval"}

		if flag.String() != "myval" {
			t.Errorf("String() = %q, want myval", flag.String())
		}
	})
}

// TestStringSliceFlag verifies the stringSliceFlag value type.
//
//nolint:funlen,gocyclo,cyclop // table-driven test with multiple sub-cases
func TestStringSliceFlag(t *testing.T) {
	t.Parallel()

	t.Run("IsSet is false before Set is called", func(t *testing.T) {
		t.Parallel()

		var flag stringSliceFlag

		if flag.IsSet {
			t.Error("IsSet should be false before Set()")
		}
	})

	t.Run("Set splits comma-separated values", func(t *testing.T) {
		t.Parallel()

		var flag stringSliceFlag

		err := flag.Set("a,b,c")
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if !flag.IsSet {
			t.Error("IsSet = false after Set()")
		}

		if len(flag.Values) != 3 {
			t.Fatalf("Values len = %d, want 3", len(flag.Values))
		}

		if flag.Values[0] != "a" || flag.Values[1] != "b" || flag.Values[2] != "c" {
			t.Errorf("Values = %v, want [a b c]", flag.Values)
		}
	})

	t.Run("Set trims whitespace from each value", func(t *testing.T) {
		t.Parallel()

		var flag stringSliceFlag

		err := flag.Set("  a  ,  b  ")
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if flag.Values[0] != "a" || flag.Values[1] != "b" {
			t.Errorf("Values = %v, want [a b]", flag.Values)
		}
	})

	t.Run("Set skips empty parts", func(t *testing.T) {
		t.Parallel()

		var flag stringSliceFlag

		err := flag.Set("a,,b")
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if len(flag.Values) != 2 {
			t.Errorf("Values len = %d, want 2 (empty parts skipped)", len(flag.Values))
		}
	})

	t.Run("multiple Set calls accumulate values", func(t *testing.T) {
		t.Parallel()

		var flag stringSliceFlag

		_ = flag.Set("a,b")
		_ = flag.Set("c")

		if len(flag.Values) != 3 {
			t.Errorf("Values len = %d, want 3 after two Set() calls", len(flag.Values))
		}
	})

	t.Run("String returns comma-joined values", func(t *testing.T) {
		t.Parallel()

		flag := stringSliceFlag{Values: []string{"x", "y", "z"}}
		got := flag.String()

		if got != "x,y,z" {
			t.Errorf("String() = %q, want x,y,z", got)
		}
	})
}
