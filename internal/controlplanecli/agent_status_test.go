package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAgentStatus_Success(t *testing.T) {
	t.Setenv(EnvToken, "")

	now := time.Date(2026, 4, 12, 13, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-abc/agent-session" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId":          "sess-123",
			"runId":              "run-abc",
			"agent":              "codex",
			"runtime":            "sandbox-agent",
			"phase":              "Ready",
			"workspaceSessionId": "ws-789",
			"createdAt":          now.Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "status", "run-abc"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"Session ID:", "sess-123",
		"Run ID:", "run-abc",
		"Agent:", "codex",
		"Runtime:", "sandbox-agent",
		"Phase:", "Ready",
		"Workspace:", "ws-789",
		"Created:", "2026-04-12T13:00:00Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

func TestAgentStatus_JSONOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	now := time.Date(2026, 4, 12, 13, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId":          "sess-123",
			"runId":              "run-abc",
			"agent":              "codex",
			"runtime":            "sandbox-agent",
			"phase":              "Ready",
			"workspaceSessionId": "ws-789",
			"createdAt":          now.Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "status", "run-abc", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	var session AgentSession
	if err := json.Unmarshal(stdout.Bytes(), &session); err != nil {
		t.Fatalf("unmarshal JSON output: %v\nraw: %s", err, stdout.String())
	}
	if session.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want sess-123", session.SessionID)
	}
	if session.RunID != "run-abc" {
		t.Errorf("RunID = %q, want run-abc", session.RunID)
	}
	if session.Agent != "codex" {
		t.Errorf("Agent = %q, want codex", session.Agent)
	}
	if session.Runtime != "sandbox-agent" {
		t.Errorf("Runtime = %q, want sandbox-agent", session.Runtime)
	}
	if session.Phase != "Ready" {
		t.Errorf("Phase = %q, want Ready", session.Phase)
	}
	if session.WorkspaceID != "ws-789" {
		t.Errorf("WorkspaceID = %q, want ws-789", session.WorkspaceID)
	}
}

func TestAgentStatus_MissingRunID(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "status"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage error, got:\n%s", stderr.String())
	}
}

func TestAgentStatus_NotFound(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session not found"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "status", "run-missing"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected not found error, got:\n%s", stderr.String())
	}
}
