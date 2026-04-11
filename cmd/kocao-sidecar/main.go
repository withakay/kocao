package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/withakay/kocao/internal/sidecar/tokensync"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultSecretName   = "kocao-agent-oauth"
	defaultPollInterval = 5 * time.Second
	defaultWatchPaths   = "/home/kocao/.local/share/opencode/auth.json:opencode-auth.json,/home/kocao/.codex/auth.json:codex-auth.json"
	defaultFeatures     = "tokensync"
)

func main() {
	namespace := flag.String("namespace", envOrDefault("KOCAO_NAMESPACE", ""), "Kubernetes namespace (default: auto-detect from in-cluster)")
	secretName := flag.String("secret-name", defaultSecretName, "Name of the Secret to patch")
	pollInterval := flag.Duration("poll-interval", defaultPollInterval, "How often to poll watched files")
	watchPaths := flag.String("watch-paths", defaultWatchPaths, "Comma-separated path:secretKey pairs")
	features := flag.String("features", defaultFeatures, "Comma-separated feature names to enable")
	flag.Parse()

	// Resolve namespace from in-cluster file if not set.
	ns := *namespace
	if ns == "" {
		data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err == nil {
			ns = strings.TrimSpace(string(data))
		}
		if ns == "" {
			ns = "default"
		}
	}

	slog.Info("kocao-sidecar starting",
		"namespace", ns,
		"secret", *secretName,
		"poll_interval", pollInterval.String(),
		"features", *features,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	enabledFeatures := parseFeatures(*features)

	if enabledFeatures["tokensync"] {
		mappings, err := ParseWatchPaths(*watchPaths)
		if err != nil {
			slog.Error("invalid watch-paths", "error", err)
			os.Exit(1)
		}

		cfg, err := rest.InClusterConfig()
		if err != nil {
			slog.Error("failed to get in-cluster config", "error", err)
			os.Exit(1)
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			slog.Error("failed to create kubernetes client", "error", err)
			os.Exit(1)
		}

		patcher := tokensync.NewK8sPatcher(clientset, ns, *secretName)
		watcher := tokensync.New(*pollInterval, mappings, patcher)

		slog.Info("tokensync enabled", "mappings", len(mappings))
		go func() {
			if err := watcher.Run(ctx); err != nil {
				slog.Error("watcher exited with error", "error", err)
			}
		}()
	}

	<-ctx.Done()
	slog.Info("kocao-sidecar shutting down")
}

// ParseWatchPaths parses "path1:key1,path2:key2" into FileMapping slices.
func ParseWatchPaths(raw string) ([]tokensync.FileMapping, error) {
	if raw == "" {
		return nil, nil
	}

	var mappings []tokensync.FileMapping
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid watch-path pair %q: expected path:secretKey", pair)
		}
		mappings = append(mappings, tokensync.FileMapping{
			Path:      parts[0],
			SecretKey: parts[1],
		})
	}
	return mappings, nil
}

func parseFeatures(raw string) map[string]bool {
	features := make(map[string]bool)
	for _, f := range strings.Split(raw, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			features[f] = true
		}
	}
	return features
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
