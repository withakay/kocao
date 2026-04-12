package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentStop_Success(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/harness-runs/run-42/agent-session/stop" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case r.URL.Path == "/api/v1/harness-runs/run-42/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":          "as-42",
				"runId":              "run-42",
				"displayName":        "my-agent",
				"runtime":            "opencode",
				"agent":              "claude",
				"phase":              "Stopped",
				"workspaceSessionId": "ws-5",
			})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "stop", "run-42"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Agent session stopped") {
		t.Errorf("expected 'Agent session stopped' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "run-42") {
		t.Errorf("expected run ID in output, got:\n%s", out)
	}
}

func TestAgentStop_SuccessJSON(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/harness-runs/run-42/agent-session/stop" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)

		case r.URL.Path == "/api/v1/harness-runs/run-42/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":          "as-42",
				"runId":              "run-42",
				"displayName":        "my-agent",
				"runtime":            "opencode",
				"agent":              "claude",
				"phase":              "Stopped",
				"workspaceSessionId": "ws-5",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "stop", "run-42", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"runId"`) {
		t.Errorf("expected JSON output with runId, got:\n%s", out)
	}
	if !strings.Contains(out, `"run-42"`) {
		t.Errorf("expected run-42 in JSON output, got:\n%s", out)
	}
}

func TestAgentStop_MissingRunID(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "stop"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage error in stderr, got:\n%s", stderr.String())
	}
}

func TestAgentStop_NotFound(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session not found"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "stop", "run-missing"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected 'not found' in stderr, got:\n%s", stderr.String())
	}
}

func TestAgentStop_AlreadyStopped(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/harness-runs/run-done/agent-session/stop" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session already stopped"})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "stop", "run-done"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "already stopped") {
		t.Errorf("expected 'already stopped' in stderr, got:\n%s", errOut)
	}
}
