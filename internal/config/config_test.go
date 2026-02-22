package config

import "testing"

func TestLoadFrom_ValidDefaults(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.Env != "dev" {
		t.Fatalf("expected Env=dev, got %q", cfg.Env)
	}
	if cfg.HTTPAddr == "" {
		t.Fatalf("expected HTTPAddr non-empty")
	}
}

func TestLoadFrom_InvalidEnv(t *testing.T) {
	_, err := LoadFrom(mapGetenv(map[string]string{"CP_ENV": "nope"}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadFrom_InClusterRequiresNamespace(t *testing.T) {
	_, err := LoadFrom(mapGetenv(map[string]string{"CP_IN_CLUSTER": "true"}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadFrom_InClusterNamespaceOK(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{"CP_IN_CLUSTER": "true", "POD_NAMESPACE": "kocao-system"}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.Namespace != "kocao-system" {
		t.Fatalf("expected namespace, got %q", cfg.Namespace)
	}
}

func mapGetenv(m map[string]string) func(string) string {
	return func(key string) string {
		return m[key]
	}
}
