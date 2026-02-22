package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Runtime struct {
	Env      string
	HTTPAddr string
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

	return Runtime{
		Env:       env,
		HTTPAddr:  httpAddr,
		Namespace: ns,
	}, nil
}

func inCluster(getenv func(string) string) bool {
	if strings.EqualFold(strings.TrimSpace(getenv("CP_IN_CLUSTER")), "true") {
		return true
	}
	return strings.TrimSpace(getenv("KUBERNETES_SERVICE_HOST")) != ""
}
