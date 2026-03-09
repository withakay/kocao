package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestMainSymphonyListAndPause(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/symphony-projects" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"symphonyProjects": []map[string]any{{
					"name":   "demo",
					"paused": false,
					"spec": map[string]any{
						"source": map[string]any{"project": map[string]any{"owner": "withakay", "number": 42}},
					},
					"status": map[string]any{"phase": "Ready"},
				}},
			})
		case r.URL.Path == "/api/v1/symphony-projects/demo/pause" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "demo", "paused": true, "spec": map[string]any{"source": map[string]any{"project": map[string]any{"owner": "withakay", "number": 42}}}, "status": map[string]any{"phase": "Paused"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "symphony", "ls"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo") {
		t.Fatalf("expected project in output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"--api-url", srv.URL, "--token", "test-token", "symphony", "pause", "demo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "pause symphony project demo") {
		t.Fatalf("expected pause output, got: %s", stdout.String())
	}
}

func TestMainSymphonyCreateJSON(t *testing.T) {
	t.Setenv(EnvToken, "")
	tempDir := t.TempDir()
	payloadPath := filepath.Join(tempDir, "project.json")
	if err := os.WriteFile(payloadPath, []byte(`{"name":"demo","spec":{"source":{"project":{"owner":"withakay","number":42},"tokenSecretRef":{"name":"github-token"},"activeStates":["Queued"],"terminalStates":["Done"]},"repositories":[{"owner":"withakay","name":"kocao"}],"runtime":{"image":"ghcr.io/withakay/kocao-harness:latest"}}}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/symphony-projects" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["name"] != "demo" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "demo", "paused": false, "spec": payload["spec"], "status": map[string]any{"phase": "Pending"}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "symphony", "create", "--file", payloadPath, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "demo"`) {
		t.Fatalf("expected created project JSON, got: %s", stdout.String())
	}
}
