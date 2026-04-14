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

		case r.URL.Path == "/api/v1/harness-runs/run-done/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":          "as-done",
				"runId":              "run-done",
				"displayName":        "my-agent",
				"runtime":            "opencode",
				"agent":              "claude",
				"phase":              "Completed",
				"workspaceSessionId": "ws-5",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "stop", "run-done"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Agent session stopped") {
		t.Errorf("expected stop summary in stdout, got:\n%s", out)
	}
	if !strings.Contains(out, "run-done") {
		t.Errorf("expected run ID in output, got:\n%s", out)
	}
}

// TestAgentStop_SlowServer verifies that the stop command does not hang
// indefinitely when the server is slow to respond. The CLI should use a
// bounded context timeout.
func TestAgentStop_SlowServer(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server that takes longer than the client timeout.
		// The test uses a 1s timeout, so sleeping 5s should trigger it.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	done := make(chan int, 1)
	go func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Main([]string{
			"--api-url", srv.URL,
			"--token", "test-token",
			"--timeout", "1s",
			"agent", "stop", "run-slow",
		}, &stdout, &stderr)
		done <- code
	}()

	select {
	case code := <-done:
		// The stop command should fail (non-zero) due to timeout, not hang.
		if code == 0 {
			t.Fatal("expected non-zero exit code for timed-out stop")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stop command hung despite timeout - context timeout not applied")
	}
}
