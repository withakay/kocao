package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestAgentList_TableOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{
					{"id": "ws-1", "phase": "Active"},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-1/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-1",
						"runId":              "run-1",
						"displayName":        "demo",
						"runtime":            "opencode",
						"agent":              "claude",
						"phase":              "Running",
						"workspaceSessionId": "ws-1",
						"createdAt":          "2025-06-15T10:30:00Z",
					},
					{
						"sessionId":          "as-2",
						"runId":              "run-2",
						"displayName":        "test",
						"runtime":            "opencode",
						"agent":              "gpt4",
						"phase":              "Stopped",
						"workspaceSessionId": "ws-1",
						"createdAt":          "2025-06-15T11:00:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	// Verify header row
	if !strings.Contains(out, "SESSION ID") {
		t.Errorf("missing SESSION ID header, got:\n%s", out)
	}
	if !strings.Contains(out, "RUN") {
		t.Errorf("missing RUN header, got:\n%s", out)
	}
	if !strings.Contains(out, "AGENT") {
		t.Errorf("missing AGENT header, got:\n%s", out)
	}
	if !strings.Contains(out, "PHASE") {
		t.Errorf("missing PHASE header, got:\n%s", out)
	}
	if !strings.Contains(out, "WORKSPACE") {
		t.Errorf("missing WORKSPACE header, got:\n%s", out)
	}
	if !strings.Contains(out, "CREATED") {
		t.Errorf("missing CREATED header, got:\n%s", out)
	}

	// Verify data rows
	if !strings.Contains(out, "as-1") {
		t.Errorf("missing session as-1, got:\n%s", out)
	}
	if !strings.Contains(out, "as-2") {
		t.Errorf("missing session as-2, got:\n%s", out)
	}
	if !strings.Contains(out, "claude") {
		t.Errorf("missing agent claude, got:\n%s", out)
	}
	if !strings.Contains(out, "Running") {
		t.Errorf("missing phase Running, got:\n%s", out)
	}
}

func TestAgentList_JSONOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{
					{"id": "ws-1", "phase": "Active"},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-1/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-1",
						"runId":              "run-1",
						"agent":              "claude",
						"phase":              "Running",
						"workspaceSessionId": "ws-1",
						"createdAt":          "2025-06-15T10:30:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	var result []AgentSession
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal JSON output: %v\nraw: %s", err, stdout.String())
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[0].SessionID != "as-1" {
		t.Fatalf("SessionID = %q, want as-1", result[0].SessionID)
	}
	if result[0].Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", result[0].Agent)
	}
}

func TestAgentList_YAMLOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{
					{"id": "ws-1", "phase": "Active"},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-1/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-1",
						"runId":              "run-1",
						"agent":              "claude",
						"phase":              "Running",
						"workspaceSessionId": "ws-1",
						"createdAt":          "2025-06-15T10:30:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list", "--output", "yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	// YAML output should contain key-value pairs
	if !strings.Contains(out, "sessionId: as-1") {
		t.Errorf("YAML output missing sessionId, got:\n%s", out)
	}
	if !strings.Contains(out, "agent: claude") {
		t.Errorf("YAML output missing agent, got:\n%s", out)
	}
}

func TestAgentList_EmptyList(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{
					{"id": "ws-1", "phase": "Active"},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-1/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "no agent sessions found") {
		t.Errorf("expected 'no agent sessions found' message, got:\n%s", out)
	}
}

func TestAgentList_EmptyWorkspaces(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "no agent sessions found") {
		t.Errorf("expected 'no agent sessions found' message, got:\n%s", out)
	}
}

func TestAgentList_WorkspaceFilter(t *testing.T) {
	t.Setenv(EnvToken, "")

	var mu sync.Mutex
	var requestedPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPaths = append(requestedPaths, r.URL.Path)
		mu.Unlock()
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions/ws-42/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-99",
						"runId":              "run-99",
						"agent":              "claude",
						"phase":              "Running",
						"workspaceSessionId": "ws-42",
						"createdAt":          "2025-06-15T10:30:00Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list", "--workspace", "ws-42"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	// Should NOT have called ListWorkspaceSessions
	mu.Lock()
	paths := append([]string{}, requestedPaths...)
	mu.Unlock()
	for _, p := range paths {
		if p == "/api/v1/workspace-sessions" {
			t.Error("should not list all workspace sessions when --workspace is provided")
		}
	}

	out := stdout.String()
	if !strings.Contains(out, "as-99") {
		t.Errorf("expected session as-99 in output, got:\n%s", out)
	}
}

func TestAgentList_MultipleWorkspaces(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{
					{"id": "ws-1", "phase": "Active"},
					{"id": "ws-2", "phase": "Active"},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-1/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-1",
						"runId":              "run-1",
						"agent":              "claude",
						"phase":              "Running",
						"workspaceSessionId": "ws-1",
					},
				},
			})
		case r.URL.Path == "/api/v1/workspace-sessions/ws-2/agent-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agentSessions": []map[string]any{
					{
						"sessionId":          "as-2",
						"runId":              "run-2",
						"agent":              "gpt4",
						"phase":              "Stopped",
						"workspaceSessionId": "ws-2",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "as-1") {
		t.Errorf("missing session as-1 from ws-1, got:\n%s", out)
	}
	if !strings.Contains(out, "as-2") {
		t.Errorf("missing session as-2 from ws-2, got:\n%s", out)
	}
}

func TestAgentList_APIError(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "internal server error"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "internal server error") {
		t.Errorf("expected error message in stderr, got:\n%s", stderr.String())
	}
}

func TestAgentList_WorkspaceFilterAPIError(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "workspace not found"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "list", "--workspace", "ws-missing"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
}

func TestAgentList_LsAlias(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workspaceSessions": []map[string]any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "ls"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	// "ls" alias should work the same as "list"
	if !strings.Contains(stdout.String(), "no agent sessions found") {
		t.Errorf("expected 'no agent sessions found' message via ls alias, got:\n%s", stdout.String())
	}
}
