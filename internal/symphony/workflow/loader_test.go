package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	repoRoot := "/tmp/repo"
	if got := ResolvePath(repoRoot, ""); got != filepath.Join(repoRoot, DefaultFileName) {
		t.Fatalf("default path = %q", got)
	}
	if got := ResolvePath(repoRoot, "nested/WORKFLOW.md"); got != filepath.Join(repoRoot, "nested/WORKFLOW.md") {
		t.Fatalf("relative path = %q", got)
	}
}

func TestLoadWithoutFrontMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte("You are working on {{.issue.title}}."), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	def, err := Load(path)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}
	if len(def.Config) != 0 {
		t.Fatalf("config = %#v, want empty", def.Config)
	}
	out, err := def.Render(map[string]any{"title": "Issue title"}, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "You are working on Issue title." {
		t.Fatalf("rendered prompt = %q", out)
	}
}

func TestLoadFrontMatterAndTypedConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	content := strings.Join([]string{
		"---",
		"tracker:",
		"  kind: linear",
		"  api_key: $LINEAR_API_KEY",
		"workspace:",
		"  root: ~/tmp/workspaces",
		"agent:",
		"  max_concurrent_agents: 3",
		"  max_concurrent_agents_by_state:",
		"    Todo: 2",
		"codex:",
		"  command: codex app-server",
		"---",
		"Issue: {{.issue.title | upper}}",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	def, err := Load(path)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}
	cfg, err := def.TypedConfig(func(key string) string {
		if key == "LINEAR_API_KEY" {
			return "test-token"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("typed config: %v", err)
	}
	if cfg.Tracker.APIKey != "test-token" {
		t.Fatalf("tracker api key = %q", cfg.Tracker.APIKey)
	}
	if cfg.Agent.MaxConcurrentAgents != 3 {
		t.Fatalf("max concurrent = %d", cfg.Agent.MaxConcurrentAgents)
	}
	if cfg.Agent.MaxConcurrentAgentsByState["todo"] != 2 {
		t.Fatalf("state override = %#v", cfg.Agent.MaxConcurrentAgentsByState)
	}
	out, err := def.Render(map[string]any{"title": "hello"}, nil)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if out != "Issue: HELLO" {
		t.Fatalf("rendered prompt = %q", out)
	}
}

func TestLoadMissingFileReturnsTypedError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), DefaultFileName))
	var workflowErr *Error
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if workflowErr.Code != ErrCodeMissingWorkflowFile {
		t.Fatalf("error code = %q", workflowErr.Code)
	}
}

func TestLoadInvalidYAMLReturnsTypedError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte("---\ntracker: [\n---\nprompt"), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	_, err := Load(path)
	var workflowErr *Error
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if workflowErr.Code != ErrCodeWorkflowParseError {
		t.Fatalf("error code = %q", workflowErr.Code)
	}
}

func TestLoadFrontMatterMustBeMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte("---\n- a\n- b\n---\nprompt"), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	_, err := Load(path)
	var workflowErr *Error
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if workflowErr.Code != ErrCodeWorkflowFrontMatterNotMap {
		t.Fatalf("error code = %q", workflowErr.Code)
	}
}

func TestRenderFailsOnMissingVariable(t *testing.T) {
	def := Definition{Path: DefaultFileName, PromptTemplate: "Issue {{.issue.title}} for {{.issue.owner}}"}
	_, err := def.Render(map[string]any{"title": "hello"}, nil)
	var workflowErr *Error
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if workflowErr.Code != ErrCodeTemplateRenderError {
		t.Fatalf("error code = %q", workflowErr.Code)
	}
}

func TestRenderFailsOnUnknownFunction(t *testing.T) {
	def := Definition{Path: DefaultFileName, PromptTemplate: "{{.issue.title | doesNotExist}}"}
	_, err := def.Render(map[string]any{"title": "hello"}, nil)
	var workflowErr *Error
	if !errors.As(err, &workflowErr) {
		t.Fatalf("expected typed error, got %v", err)
	}
	if workflowErr.Code != ErrCodeTemplateParseError {
		t.Fatalf("error code = %q", workflowErr.Code)
	}
}
