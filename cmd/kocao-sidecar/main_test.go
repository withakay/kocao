package main

import (
	"testing"
)

func TestParseWatchPaths(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "default paths",
			input: "/home/kocao/.local/share/opencode/auth.json:opencode-auth.json,/home/kocao/.codex/auth.json:codex-auth.json",
			want:  2,
		},
		{
			name:  "single path",
			input: "/tmp/auth.json:my-key",
			want:  1,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:    "missing key",
			input:   "/tmp/auth.json",
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   "/tmp/auth.json:",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   ":my-key",
			wantErr: true,
		},
		{
			name:  "trailing comma",
			input: "/tmp/a.json:key-a,",
			want:  1,
		},
		{
			name:  "whitespace trimmed",
			input: " /tmp/a.json:key-a , /tmp/b.json:key-b ",
			want:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWatchPaths(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.want {
				t.Fatalf("got %d mappings, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseWatchPaths_Values(t *testing.T) {
	mappings, err := ParseWatchPaths("/home/kocao/.local/share/opencode/auth.json:opencode-auth.json,/home/kocao/.codex/auth.json:codex-auth.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}

	if mappings[0].Path != "/home/kocao/.local/share/opencode/auth.json" {
		t.Errorf("mapping[0].Path = %q", mappings[0].Path)
	}
	if mappings[0].SecretKey != "opencode-auth.json" {
		t.Errorf("mapping[0].SecretKey = %q", mappings[0].SecretKey)
	}
	if mappings[1].Path != "/home/kocao/.codex/auth.json" {
		t.Errorf("mapping[1].Path = %q", mappings[1].Path)
	}
	if mappings[1].SecretKey != "codex-auth.json" {
		t.Errorf("mapping[1].SecretKey = %q", mappings[1].SecretKey)
	}
}

func TestFlagDefaults(t *testing.T) {
	// Verify the constants match expected defaults.
	if defaultSecretName != "kocao-agent-oauth" {
		t.Errorf("defaultSecretName = %q, want %q", defaultSecretName, "kocao-agent-oauth")
	}
	if defaultPollInterval.String() != "5s" {
		t.Errorf("defaultPollInterval = %v, want 5s", defaultPollInterval)
	}
	if defaultFeatures != "tokensync" {
		t.Errorf("defaultFeatures = %q, want %q", defaultFeatures, "tokensync")
	}
}

func TestParseFeatures(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]bool
	}{
		{
			name:  "single feature",
			input: "tokensync",
			want:  map[string]bool{"tokensync": true},
		},
		{
			name:  "multiple features",
			input: "tokensync,metrics,healthcheck",
			want:  map[string]bool{"tokensync": true, "metrics": true, "healthcheck": true},
		},
		{
			name:  "empty string",
			input: "",
			want:  map[string]bool{},
		},
		{
			name:  "whitespace trimmed",
			input: " tokensync , metrics ",
			want:  map[string]bool{"tokensync": true, "metrics": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFeatures(tt.input)
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("feature %q: got %v, want %v", k, got[k], v)
				}
			}
			if len(got) != len(tt.want) {
				t.Errorf("got %d features, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	if got := envOrDefault("VERY_UNLIKELY_ENV_VAR_12345", "fallback"); got != "fallback" {
		t.Errorf("envOrDefault() = %q, want %q", got, "fallback")
	}

	t.Setenv("TEST_ENV_OR_DEFAULT", "from-env")
	if got := envOrDefault("TEST_ENV_OR_DEFAULT", "fallback"); got != "from-env" {
		t.Errorf("envOrDefault() = %q, want %q", got, "from-env")
	}
}
