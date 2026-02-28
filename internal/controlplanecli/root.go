package controlplanecli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func Main(args []string, stdout io.Writer, stderr io.Writer) int {
	if stdout == nil || stderr == nil {
		return 2
	}

	configPath, err := extractConfigPath(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", err.Error())
		return 2
	}

	cfg, err := ResolveConfig(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", explainCommandError(err))
		return 1
	}
	var debug bool
	configFlag := configPath
	fs := flag.NewFlagSet("kocao", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&configFlag, "config", configFlag, "config file path (.json)")
	fs.StringVar(&cfg.BaseURL, "api-url", cfg.BaseURL, "control-plane base URL")
	fs.StringVar(&cfg.Token, "token", cfg.Token, "bearer token")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "HTTP timeout (e.g. 15s)")
	fs.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "print request/response diagnostics")
	fs.BoolVar(&debug, "debug", false, "alias for --verbose")
	fs.Usage = func() {
		writeRootUsage(stderr)
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if debug {
		cfg.Verbose = true
	}
	parsedConfigPath := ""
	if strings.TrimSpace(configFlag) != "" {
		var parseErr error
		parsedConfigPath, parseErr = expandPath(configFlag)
		if parseErr != nil {
			_, _ = fmt.Fprintf(stderr, "error: %s\n", parseErr.Error())
			return 2
		}
	}
	if strings.TrimSpace(parsedConfigPath) != strings.TrimSpace(configPath) {
		_, _ = fmt.Fprintln(stderr, "error: --config value changed during parse; pass a single --config flag")
		return 2
	}
	cfg.LogOutput = stderr

	rest := fs.Args()
	if len(rest) == 0 {
		writeRootUsage(stdout)
		return 0
	}

	cmd := strings.ToLower(strings.TrimSpace(rest[0]))
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		writeRootUsage(stdout)
		return 0
	}

	var cmdErr error
	switch cmd {
	case "sessions":
		cmdErr = runSessionsCommand(rest[1:], cfg, stdout, stderr)
	default:
		cmdErr = fmt.Errorf("unknown command %q", cmd)
	}
	if cmdErr != nil {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", explainCommandError(cmdErr))
		msg := cmdErr.Error()
		if strings.HasPrefix(msg, "usage:") || strings.HasPrefix(msg, "unknown") || strings.Contains(msg, "flag provided but not defined") {
			return 2
		}
		return 1
	}

	return 0
}

func extractConfigPath(args []string) (string, error) {
	var configPath string
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		switch {
		case a == "--config":
			if i+1 >= len(args) {
				return "", fmt.Errorf("--config requires a file path")
			}
			configPath = strings.TrimSpace(args[i+1])
			i++
		case strings.HasPrefix(a, "--config="):
			configPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		}
	}
	if configPath == "" {
		return "", nil
	}
	abs, err := expandPath(configPath)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func expandPath(p string) (string, error) {
	v := strings.TrimSpace(p)
	if v == "" {
		return "", fmt.Errorf("config path cannot be empty")
	}
	if strings.HasPrefix(v, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		v = home + v[1:]
	}
	return v, nil
}
