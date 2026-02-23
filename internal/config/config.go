package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Runtime struct {
	Env       string
	HTTPAddr  string
	AuditPath string

	// AttachWSAllowedOrigins is a comma-separated allowlist for websocket Origin validation.
	// Entries MUST be full origins (e.g. "https://kocao.example.com", "http://localhost:5173").
	// When empty, the control-plane applies environment defaults (strict in prod).
	AttachWSAllowedOrigins []string

	// BootstrapToken, when set, is inserted (if missing) into the token store with
	// wildcard scopes. Use this for initial bring-up only.
	BootstrapToken string
	// Namespace is required when running in-cluster.
	Namespace string
}

func Load() (Runtime, error) {
	_ = godotenv.Load()
	return LoadFrom(os.Getenv)
}

func LoadFrom(getenv func(string) string) (Runtime, error) {
	env := strings.TrimSpace(getenv("CP_ENV"))
	if env == "" {
		env = "dev"
	}
	switch env {
	case "dev", "test", "prod":
	default:
		return Runtime{}, fmt.Errorf("CP_ENV must be one of dev|test|prod (got %q)", env)
	}

	httpAddr := strings.TrimSpace(getenv("CP_HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	if _, err := net.ResolveTCPAddr("tcp", httpAddr); err != nil {
		return Runtime{}, fmt.Errorf("CP_HTTP_ADDR invalid (%q): %w", httpAddr, err)
	}

	ns := strings.TrimSpace(getenv("POD_NAMESPACE"))
	if ns == "" {
		ns = strings.TrimSpace(getenv("CP_NAMESPACE"))
	}
	if inCluster(getenv) && ns == "" {
		return Runtime{}, fmt.Errorf("namespace required in-cluster: set POD_NAMESPACE (recommended) or CP_NAMESPACE")
	}

	auditPath := strings.TrimSpace(getenv("CP_AUDIT_PATH"))
	if auditPath == "" {
		// Deprecated alias: CP_DB_PATH historically (mis)configured audit persistence.
		auditPath = strings.TrimSpace(getenv("CP_DB_PATH"))
	}
	if auditPath == "" {
		auditPath = "kocao.audit.jsonl"
	}

	bootstrapToken := strings.TrimSpace(getenv("CP_BOOTSTRAP_TOKEN"))
	if env == "prod" && bootstrapToken != "" {
		return Runtime{}, fmt.Errorf("CP_BOOTSTRAP_TOKEN is not allowed when CP_ENV=prod")
	}

	var allowedOrigins []string
	if raw := strings.TrimSpace(getenv("CP_ATTACH_WS_ALLOWED_ORIGINS")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			p := strings.TrimSpace(part)
			if p == "" {
				continue
			}
			allowedOrigins = append(allowedOrigins, p)
		}
	}

	return Runtime{
		Env:                    env,
		HTTPAddr:               httpAddr,
		AuditPath:              auditPath,
		AttachWSAllowedOrigins: allowedOrigins,
		BootstrapToken:         bootstrapToken,
		Namespace:              ns,
	}, nil
}

func inCluster(getenv func(string) string) bool {
	if strings.EqualFold(strings.TrimSpace(getenv("CP_IN_CLUSTER")), "true") {
		return true
	}
	return strings.TrimSpace(getenv("KUBERNETES_SERVICE_HOST")) != ""
}
