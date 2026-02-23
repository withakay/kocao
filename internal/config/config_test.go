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
	if cfg.AuditPath != "kocao.audit.jsonl" {
		t.Fatalf("expected default AuditPath, got %q", cfg.AuditPath)
	}
}

func TestLoadFrom_AuditPath_PrefersCPAuditPath(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{"CP_AUDIT_PATH": "/tmp/audit.jsonl"}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.AuditPath != "/tmp/audit.jsonl" {
		t.Fatalf("AuditPath = %q, want %q", cfg.AuditPath, "/tmp/audit.jsonl")
	}
}

func TestLoadFrom_AuditPath_DeprecatedAliasCPDBPath(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{"CP_DB_PATH": "legacy-audit.jsonl"}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.AuditPath != "legacy-audit.jsonl" {
		t.Fatalf("AuditPath = %q, want %q", cfg.AuditPath, "legacy-audit.jsonl")
	}
}

func TestLoadFrom_AuditPath_Precedence(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{"CP_AUDIT_PATH": "new.jsonl", "CP_DB_PATH": "old.jsonl"}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.AuditPath != "new.jsonl" {
		t.Fatalf("AuditPath = %q, want %q", cfg.AuditPath, "new.jsonl")
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

func TestLoadFrom_ProdRejectsBootstrapToken(t *testing.T) {
	_, err := LoadFrom(mapGetenv(map[string]string{"CP_ENV": "prod", "CP_BOOTSTRAP_TOKEN": "anything"}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadFrom_AttachAllowedOrigins_ParsesCSV(t *testing.T) {
	cfg, err := LoadFrom(mapGetenv(map[string]string{"CP_ATTACH_WS_ALLOWED_ORIGINS": " https://a.example ,http://localhost:5173, ,https://b.example "}))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(cfg.AttachWSAllowedOrigins) != 3 {
		t.Fatalf("AttachWSAllowedOrigins len=%d, want 3", len(cfg.AttachWSAllowedOrigins))
	}
	if cfg.AttachWSAllowedOrigins[0] != "https://a.example" {
		t.Fatalf("origin[0]=%q", cfg.AttachWSAllowedOrigins[0])
	}
	if cfg.AttachWSAllowedOrigins[1] != "http://localhost:5173" {
		t.Fatalf("origin[1]=%q", cfg.AttachWSAllowedOrigins[1])
	}
	if cfg.AttachWSAllowedOrigins[2] != "https://b.example" {
		t.Fatalf("origin[2]=%q", cfg.AttachWSAllowedOrigins[2])
	}
}

func mapGetenv(m map[string]string) func(string) string {
	return func(key string) string {
		return m[key]
	}
}
