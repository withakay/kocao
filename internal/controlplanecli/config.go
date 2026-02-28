package controlplanecli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvAPIURL  = "KOCAO_API_URL"
	EnvToken   = "KOCAO_TOKEN"
	EnvTimeout = "KOCAO_TIMEOUT"
	EnvVerbose = "KOCAO_VERBOSE"
)

type Config struct {
	BaseURL   string
	Token     string
	Timeout   time.Duration
	Verbose   bool
	LogOutput io.Writer
}

func DefaultConfig() Config {
	return Config{
		BaseURL: "http://127.0.0.1:8080",
		Timeout: 15 * time.Second,
	}
}

type configOverlay struct {
	BaseURL *string
	Token   *string
	Timeout *time.Duration
	Verbose *bool
}

type settingsFile struct {
	APIURL  *string `json:"api_url"`
	Token   *string `json:"token"`
	Timeout *string `json:"timeout"`
	Verbose *bool   `json:"verbose"`
}

func ResolveConfig(explicitConfigPath string) (Config, error) {
	defaultPaths, err := defaultConfigPaths()
	if err != nil {
		return Config{}, err
	}
	return resolveConfig(explicitConfigPath, defaultPaths, os.LookupEnv)
}

func resolveConfig(explicitConfigPath string, defaultPaths []string, lookupEnv func(string) (string, bool)) (Config, error) {
	cfg := DefaultConfig()

	for _, p := range defaultPaths {
		overlay, found, err := loadConfigOverlay(p)
		if err != nil {
			return Config{}, err
		}
		if !found {
			continue
		}
		if err := cfg.applyOverlay(overlay); err != nil {
			return Config{}, fmt.Errorf("apply config file %q: %w", p, err)
		}
	}

	if strings.TrimSpace(explicitConfigPath) != "" {
		overlay, found, err := loadConfigOverlay(explicitConfigPath)
		if err != nil {
			return Config{}, err
		}
		if !found {
			return Config{}, fmt.Errorf("config file not found: %s", explicitConfigPath)
		}
		if err := cfg.applyOverlay(overlay); err != nil {
			return Config{}, fmt.Errorf("apply config file %q: %w", explicitConfigPath, err)
		}
	}

	envOverlay, err := envOverlay(lookupEnv)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.applyOverlay(envOverlay); err != nil {
		return Config{}, err
	}

	return cfg.normalized()
}

func defaultConfigPaths() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	return []string{
		filepath.Join(home, ".config", "kocao", "settings.json"),
		filepath.Join(execDir, "settings.json"),
	}, nil
}

func loadConfigOverlay(filePath string) (configOverlay, bool, error) {
	clean := strings.TrimSpace(filePath)
	if clean == "" {
		return configOverlay{}, false, nil
	}
	if ext := strings.ToLower(filepath.Ext(clean)); ext != ".json" {
		return configOverlay{}, false, fmt.Errorf("unsupported config extension %q (use .json)", ext)
	}
	b, err := os.ReadFile(clean)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configOverlay{}, false, nil
		}
		return configOverlay{}, false, fmt.Errorf("read config file %q: %w", clean, err)
	}
	var parsed settingsFile
	if err := json.Unmarshal(b, &parsed); err != nil {
		return configOverlay{}, false, fmt.Errorf("parse config file %q: %w", clean, err)
	}

	var timeout *time.Duration
	if parsed.Timeout != nil {
		d, err := time.ParseDuration(strings.TrimSpace(*parsed.Timeout))
		if err != nil {
			return configOverlay{}, false, fmt.Errorf("invalid timeout in %q: %w", clean, err)
		}
		timeout = &d
	}

	return configOverlay{
		BaseURL: parsed.APIURL,
		Token:   parsed.Token,
		Timeout: timeout,
		Verbose: parsed.Verbose,
	}, true, nil
}

func envOverlay(lookupEnv func(string) (string, bool)) (configOverlay, error) {
	var out configOverlay

	if v, ok := lookupEnv(EnvAPIURL); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			out.BaseURL = &v
		}
	}
	if v, ok := lookupEnv(EnvToken); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			out.Token = &v
		}
	}
	if v, ok := lookupEnv(EnvTimeout); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return configOverlay{}, fmt.Errorf("invalid %s: %w", EnvTimeout, err)
			}
			out.Timeout = &d
		}
	}
	if v, ok := lookupEnv(EnvVerbose); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			parsed, err := strconv.ParseBool(v)
			if err != nil {
				return configOverlay{}, fmt.Errorf("invalid %s: %w", EnvVerbose, err)
			}
			out.Verbose = &parsed
		}
	}

	return out, nil
}

func (c *Config) applyOverlay(overlay configOverlay) error {
	if overlay.BaseURL != nil {
		c.BaseURL = strings.TrimSpace(*overlay.BaseURL)
	}
	if overlay.Token != nil {
		c.Token = strings.TrimSpace(*overlay.Token)
	}
	if overlay.Timeout != nil {
		if *overlay.Timeout <= 0 {
			return fmt.Errorf("timeout must be greater than 0")
		}
		c.Timeout = *overlay.Timeout
	}
	if overlay.Verbose != nil {
		c.Verbose = *overlay.Verbose
	}
	return nil
}

func (c Config) normalized() (Config, error) {
	baseURL := strings.TrimSpace(c.BaseURL)
	if baseURL == "" {
		return Config{}, ErrMissingAPIURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return Config{}, fmt.Errorf("parse api url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Config{}, fmt.Errorf("api url must use http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return Config{}, fmt.Errorf("api url must include host")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return Config{
		BaseURL:   u.String(),
		Token:     strings.TrimSpace(c.Token),
		Timeout:   timeout,
		Verbose:   c.Verbose,
		LogOutput: c.LogOutput,
	}, nil
}
