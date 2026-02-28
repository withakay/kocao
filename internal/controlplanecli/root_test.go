package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMainSessionsListJSON(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization header = %q", got)
		}
		if r.URL.Path != "/api/v1/workspace-sessions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"workspaceSessions": []map[string]any{{"id": "s1", "phase": "Active"}}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "sessions", "ls", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workspaceSessions") {
		t.Fatalf("stdout missing workspaceSessions: %s", stdout.String())
	}
}

func TestMainSessionStatusIncludesRun(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions/sess-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sess-1", "displayName": "demo", "phase": "Active"})
		case r.URL.Path == "/api/v1/harness-runs":
			if r.URL.Query().Get("workspaceSessionID") != "sess-1" {
				t.Fatalf("workspaceSessionID query = %q", r.URL.Query().Get("workspaceSessionID"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"harnessRuns": []map[string]any{{"id": "run-1", "phase": "Running", "podName": "pod-1"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "sessions", "status", "sess-1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "run-1") {
		t.Fatalf("expected run in output, got: %s", stdout.String())
	}
}

func TestMainSessionLogs(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions/sess-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "sess-1", "displayName": "demo", "phase": "Active"})
		case r.URL.Path == "/api/v1/harness-runs":
			_ = json.NewEncoder(w).Encode(map[string]any{"harnessRuns": []map[string]any{{"id": "run-1", "phase": "Running", "podName": "pod-1"}}})
		case r.URL.Path == "/api/v1/pods/pod-1/logs":
			_ = json.NewEncoder(w).Encode(map[string]any{"podName": "pod-1", "tailLines": 5, "logs": "line-1\nline-2\n"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "sessions", "logs", "sess-1", "--tail", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "line-1") {
		t.Fatalf("expected logs in output, got: %s", stdout.String())
	}
}

func TestMainMissingToken(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "sessions", "ls"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing bearer token") {
		t.Fatalf("expected missing token message, got: %s", stderr.String())
	}
}

func TestMainDebugShowsDiagnosticsOnNonJSONResponse(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>not json</body></html>"))
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--debug", "--api-url", srv.URL, "--token", "test-token", "sessions", "ls"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	out := stderr.String()
	if !strings.Contains(out, "debug: -> GET") {
		t.Fatalf("expected debug request line, got: %s", out)
	}
	if !strings.Contains(out, "received non-JSON response") {
		t.Fatalf("expected non-json error message, got: %s", out)
	}
}
