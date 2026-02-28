package controlplanecli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveConfigPrecedence(t *testing.T) {
	tmp := t.TempDir()
	homeCfg := filepath.Join(tmp, "home-settings.json")
	execCfg := filepath.Join(tmp, "exec-settings.json")
	explicitCfg := filepath.Join(tmp, "explicit-settings.json")

	if err := os.WriteFile(homeCfg, []byte(`{"api_url":"http://home:8080","token":"home-token","timeout":"10s","verbose":false}`), 0o644); err != nil {
		t.Fatalf("write home config: %v", err)
	}
	if err := os.WriteFile(execCfg, []byte(`{"api_url":"http://exec:8080","token":"exec-token","timeout":"20s"}`), 0o644); err != nil {
		t.Fatalf("write exec config: %v", err)
	}
	if err := os.WriteFile(explicitCfg, []byte(`{"token":"file-token","timeout":"30s"}`), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	env := map[string]string{
		EnvAPIURL:  "http://env:8080",
		EnvVerbose: "true",
	}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}

	cfg, err := resolveConfig(explicitCfg, []string{homeCfg, execCfg}, lookup)
	if err != nil {
		t.Fatalf("resolveConfig error: %v", err)
	}

	if cfg.BaseURL != "http://env:8080" {
		t.Fatalf("BaseURL = %q, want env override", cfg.BaseURL)
	}
	if cfg.Token != "file-token" {
		t.Fatalf("Token = %q, want explicit file override", cfg.Token)
	}
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if !cfg.Verbose {
		t.Fatalf("Verbose = false, want true from env")
	}
}

func TestExtractConfigPath(t *testing.T) {
	got, err := extractConfigPath([]string{"--config=~/kocao.json", "sessions", "ls"})
	if err != nil {
		t.Fatalf("extractConfigPath error: %v", err)
	}
	if got == "" || got == "~/kocao.json" {
		t.Fatalf("expected expanded config path, got %q", got)
	}
}

func TestResolveConfigRejectsUnsupportedExtension(t *testing.T) {
	_, err := resolveConfig("/tmp/settings.yaml", nil, os.LookupEnv)
	if err == nil {
		t.Fatalf("expected extension error")
	}
}
