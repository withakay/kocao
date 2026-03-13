package workflow

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v3"
)

const DefaultFileName = "WORKFLOW.md"

const (
	ErrCodeMissingWorkflowFile       = "missing_workflow_file"
	ErrCodeWorkflowParseError        = "workflow_parse_error"
	ErrCodeWorkflowFrontMatterNotMap = "workflow_front_matter_not_a_map"
	ErrCodeTemplateParseError        = "template_parse_error"
	ErrCodeTemplateRenderError       = "template_render_error"
	ErrCodeWorkflowValidationError   = "workflow_validation_error"
)

type Error struct {
	Code string
	Path string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Path) != "" {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Path, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Code, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

type Definition struct {
	Path           string
	Config         map[string]any
	PromptTemplate string
}

type TrackerConfig struct {
	Kind           string
	Endpoint       string
	APIKey         string
	ProjectSlug    string
	ActiveStates   []string
	TerminalStates []string
}

type PollingConfig struct {
	IntervalMS int
}

type WorkspaceConfig struct {
	Root string
}

type HooksConfig struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	TimeoutMS    int
}

type AgentConfig struct {
	MaxConcurrentAgents        int
	MaxTurns                   int
	MaxRetryBackoffMS          int
	MaxConcurrentAgentsByState map[string]int
}

type CodexConfig struct {
	Command           string
	ApprovalPolicy    string
	ThreadSandbox     string
	TurnSandboxPolicy string
	TurnTimeoutMS     int
	ReadTimeoutMS     int
	StallTimeoutMS    int
}

type Config struct {
	Tracker   TrackerConfig
	Polling   PollingConfig
	Workspace WorkspaceConfig
	Hooks     HooksConfig
	Agent     AgentConfig
	Codex     CodexConfig
}

func ResolvePath(repoRoot, explicitPath string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	explicitPath = strings.TrimSpace(explicitPath)
	if explicitPath == "" {
		if repoRoot == "" {
			return DefaultFileName
		}
		return filepath.Join(repoRoot, DefaultFileName)
	}
	if filepath.IsAbs(explicitPath) || repoRoot == "" {
		return explicitPath
	}
	return filepath.Join(repoRoot, explicitPath)
}

func Load(path string) (Definition, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultFileName
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Definition{}, &Error{Code: ErrCodeMissingWorkflowFile, Path: path, Err: err}
		}
		return Definition{}, &Error{Code: ErrCodeWorkflowParseError, Path: path, Err: err}
	}

	config, prompt, err := parseDocument(path, string(b))
	if err != nil {
		return Definition{}, err
	}
	return Definition{Path: path, Config: config, PromptTemplate: prompt}, nil
}

func (d Definition) TypedConfig(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	root := d.Config
	if root == nil {
		root = map[string]any{}
	}

	trackerMap := nestedMap(root, "tracker")
	pollingMap := nestedMap(root, "polling")
	workspaceMap := nestedMap(root, "workspace")
	hooksMap := nestedMap(root, "hooks")
	agentMap := nestedMap(root, "agent")
	codexMap := nestedMap(root, "codex")

	cfg := Config{
		Tracker: TrackerConfig{
			Kind:           strings.TrimSpace(stringValue(trackerMap["kind"])),
			Endpoint:       strings.TrimSpace(stringValue(trackerMap["endpoint"])),
			APIKey:         resolveEnvString(stringValue(trackerMap["api_key"]), getenv),
			ProjectSlug:    strings.TrimSpace(stringValue(trackerMap["project_slug"])),
			ActiveStates:   stateListValue(trackerMap["active_states"], []string{"Todo", "In Progress"}),
			TerminalStates: stateListValue(trackerMap["terminal_states"], []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"}),
		},
		Polling: PollingConfig{
			IntervalMS: intValue(pollingMap["interval_ms"], 30000),
		},
		Workspace: WorkspaceConfig{
			Root: expandPathLike(resolveEnvString(stringValue(workspaceMap["root"]), getenv)),
		},
		Hooks: HooksConfig{
			AfterCreate:  strings.TrimSpace(stringValue(hooksMap["after_create"])),
			BeforeRun:    strings.TrimSpace(stringValue(hooksMap["before_run"])),
			AfterRun:     strings.TrimSpace(stringValue(hooksMap["after_run"])),
			BeforeRemove: strings.TrimSpace(stringValue(hooksMap["before_remove"])),
			TimeoutMS:    positiveOrDefault(intValue(hooksMap["timeout_ms"], 60000), 60000),
		},
		Agent: AgentConfig{
			MaxConcurrentAgents:        positiveOrDefault(intValue(agentMap["max_concurrent_agents"], 10), 10),
			MaxTurns:                   positiveOrDefault(intValue(agentMap["max_turns"], 20), 20),
			MaxRetryBackoffMS:          positiveOrDefault(intValue(agentMap["max_retry_backoff_ms"], 300000), 300000),
			MaxConcurrentAgentsByState: stateLimitMap(agentMap["max_concurrent_agents_by_state"]),
		},
		Codex: CodexConfig{
			Command:           strings.TrimSpace(defaultString(stringValue(codexMap["command"]), "codex app-server")),
			ApprovalPolicy:    strings.TrimSpace(stringValue(codexMap["approval_policy"])),
			ThreadSandbox:     strings.TrimSpace(stringValue(codexMap["thread_sandbox"])),
			TurnSandboxPolicy: strings.TrimSpace(stringValue(codexMap["turn_sandbox_policy"])),
			TurnTimeoutMS:     positiveOrDefault(intValue(codexMap["turn_timeout_ms"], 3600000), 3600000),
			ReadTimeoutMS:     positiveOrDefault(intValue(codexMap["read_timeout_ms"], 5000), 5000),
			StallTimeoutMS:    intValue(codexMap["stall_timeout_ms"], 300000),
		},
	}

	if cfg.Workspace.Root == "" {
		cfg.Workspace.Root = os.TempDir() + string(os.PathSeparator) + "symphony_workspaces"
	}
	if cfg.Tracker.Kind == "linear" && cfg.Tracker.Endpoint == "" {
		cfg.Tracker.Endpoint = "https://api.linear.app/graphql"
	}
	if cfg.Tracker.Kind == "linear" && cfg.Tracker.APIKey == "" {
		cfg.Tracker.APIKey = strings.TrimSpace(getenv("LINEAR_API_KEY"))
	}
	if cfg.Codex.Command == "" {
		return Config{}, &Error{Code: ErrCodeWorkflowValidationError, Path: d.Path, Err: fmt.Errorf("codex.command is required")}
	}
	return cfg, nil
}

func (d Definition) Render(issue map[string]any, attempt *int) (string, error) {
	funcs := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"trim":  strings.TrimSpace,
		"join":  func(items []string, sep string) string { return strings.Join(items, sep) },
		"default": func(fallback, value any) any {
			if value == nil {
				return fallback
			}
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) == "" {
					return fallback
				}
			}
			return value
		},
	}
	tmpl, err := template.New("workflow").Funcs(funcs).Option("missingkey=error").Parse(d.PromptTemplate)
	if err != nil {
		return "", &Error{Code: ErrCodeTemplateParseError, Path: d.Path, Err: err}
	}
	data := map[string]any{"issue": issue}
	if attempt != nil {
		data["attempt"] = *attempt
	} else {
		data["attempt"] = nil
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", &Error{Code: ErrCodeTemplateRenderError, Path: d.Path, Err: err}
	}
	return strings.TrimSpace(out.String()), nil
}

func parseDocument(path, raw string) (map[string]any, string, error) {
	if !strings.HasPrefix(raw, "---") {
		return map[string]any{}, strings.TrimSpace(raw), nil
	}
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}, strings.TrimSpace(raw), nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, "", &Error{Code: ErrCodeWorkflowParseError, Path: path, Err: fmt.Errorf("front matter delimiter not closed")}
	}
	frontMatter := strings.Join(lines[1:end], "\n")
	prompt := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	if strings.TrimSpace(frontMatter) == "" {
		return map[string]any{}, prompt, nil
	}
	var decoded any
	if err := yaml.Unmarshal([]byte(frontMatter), &decoded); err != nil {
		return nil, "", &Error{Code: ErrCodeWorkflowParseError, Path: path, Err: err}
	}
	config, ok := normalizeMap(decoded)
	if !ok {
		return nil, "", &Error{Code: ErrCodeWorkflowFrontMatterNotMap, Path: path, Err: fmt.Errorf("front matter must decode to a map")}
	}
	return config, prompt, nil
}

func normalizeMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = normalizeValue(item)
		}
		return out, true
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[fmt.Sprint(key)] = normalizeValue(item)
		}
		return out, true
	default:
		return nil, false
	}
}

func normalizeValue(value any) any {
	if mapped, ok := normalizeMap(value); ok {
		return mapped
	}
	slice, ok := value.([]any)
	if ok {
		out := make([]any, len(slice))
		for i := range slice {
			out[i] = normalizeValue(slice[i])
		}
		return out
	}
	return value
}

func nestedMap(root map[string]any, key string) map[string]any {
	if root == nil {
		return map[string]any{}
	}
	mapped, ok := normalizeMap(root[key])
	if !ok {
		return map[string]any{}
	}
	return mapped
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func positiveOrDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func stateListValue(value any, fallback []string) []string {
	var items []string
	switch typed := value.(type) {
	case string:
		for _, part := range strings.Split(typed, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	case []any:
		for _, item := range typed {
			if trimmed := strings.TrimSpace(stringValue(item)); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	case []string:
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				items = append(items, trimmed)
			}
		}
	}
	if len(items) == 0 {
		return append([]string(nil), fallback...)
	}
	return items
}

func stateLimitMap(value any) map[string]int {
	mapped, ok := normalizeMap(value)
	if !ok {
		return map[string]int{}
	}
	out := make(map[string]int, len(mapped))
	for key, item := range mapped {
		parsed := intValue(item, 0)
		if parsed > 0 {
			out[strings.ToLower(strings.TrimSpace(key))] = parsed
		}
	}
	return out
}

func resolveEnvString(value string, getenv func(string) string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "$") {
		return value
	}
	name := strings.TrimPrefix(value, "$")
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return strings.TrimSpace(getenv(name))
}

func expandPathLike(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "~") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if value == "~" {
				value = home
			} else if strings.HasPrefix(value, "~/") {
				value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
			}
		}
	}
	if strings.HasPrefix(value, "$") {
		return os.ExpandEnv(value)
	}
	return value
}
