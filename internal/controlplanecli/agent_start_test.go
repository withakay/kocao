package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAgentStart_Success(t *testing.T) {
	t.Setenv(EnvToken, "")

	var pollCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Create workspace session
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "ws-100",
				"repoURL": "https://github.com/example/repo",
				"phase":   "Active",
			})

		// Create harness run
		case r.URL.Path == "/api/v1/workspace-sessions/ws-100/harness-runs" && r.Method == http.MethodPost:
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode harness run request: %v", err)
			}
			imageProfile, ok := req["imageProfile"].(map[string]any)
			if !ok {
				t.Fatal("expected imageProfile in harness run request")
			}
			if imageProfile["profile"] != "go" {
				t.Fatalf("imageProfile.profile = %v, want go", imageProfile["profile"])
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "run-100",
				"workspaceSessionID": "ws-100",
				"repoURL":            "https://github.com/example/repo",
				"image":              "kocao/harness-runtime:dev",
				"imageProfile":       map[string]any{"requestedProfile": "go", "selectionPolicy": "auto", "selectedProfile": "go", "selectionSource": "explicit", "fallbackProfile": "full", "reason": "explicit-request"},
				"phase":              "Starting",
			})

		// Create agent session
		case r.URL.Path == "/api/v1/harness-runs/run-100/agent-session" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":          "as-100",
				"runId":              "run-100",
				"agent":              "claude",
				"imageProfile":       map[string]any{"requestedProfile": "go", "selectionPolicy": "auto", "selectedProfile": "go", "selectionSource": "explicit", "fallbackProfile": "full", "reason": "explicit-request"},
				"phase":              "Starting",
				"workspaceSessionId": "ws-100",
			})

		// Poll agent session — return Ready on second poll
		case r.URL.Path == "/api/v1/harness-runs/run-100/agent-session" && r.Method == http.MethodGet:
			n := pollCount.Add(1)
			phase := "Starting"
			if n >= 2 {
				phase = "Ready"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId":          "as-100",
				"runId":              "run-100",
				"agent":              "claude",
				"imageProfile":       map[string]any{"requestedProfile": "go", "selectionPolicy": "auto", "selectedProfile": "go", "selectionSource": "explicit", "fallbackProfile": "full", "reason": "explicit-request"},
				"phase":              phase,
				"workspaceSessionId": "ws-100",
			})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL, "--token", "test-token",
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "claude",
		"--image-profile", "go",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "as-100") {
		t.Errorf("expected session ID in output, got:\n%s", out)
	}
	if !strings.Contains(out, "run-100") {
		t.Errorf("expected run ID in output, got:\n%s", out)
	}
	if !strings.Contains(out, "claude") {
		t.Errorf("expected agent name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Ready") {
		t.Errorf("expected Ready phase in output, got:\n%s", out)
	}
	if !strings.Contains(out, "go (explicit)") {
		t.Errorf("expected image profile in output, got:\n%s", out)
	}

	// Verify progress messages went to stderr
	errOut := stderr.String()
	if !strings.Contains(errOut, "Creating workspace session... done") {
		t.Errorf("expected progress message on stderr, got:\n%s", errOut)
	}
	if !strings.Contains(errOut, "Waiting for agent session... ready") {
		t.Errorf("expected ready message on stderr, got:\n%s", errOut)
	}
}

func TestAgentStart_MissingRequiredFlags(t *testing.T) {
	t.Setenv(EnvToken, "")

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "missing --repo",
			args:    []string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "start", "--agent", "claude"},
			wantMsg: "missing required flag --repo",
		},
		{
			name:    "missing --agent",
			args:    []string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "start", "--repo", "https://github.com/example/repo"},
			wantMsg: "missing required flag --agent",
		},
		{
			name:    "missing both",
			args:    []string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "start"},
			wantMsg: "missing required flag --repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Main(tt.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("expected non-zero exit code, got 0; stdout=%s", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.wantMsg) {
				t.Errorf("expected error containing %q, got:\n%s", tt.wantMsg, stderr.String())
			}
		})
	}
}

func TestAgentStart_Timeout(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "ws-timeout",
				"phase": "Active",
			})

		case r.URL.Path == "/api/v1/workspace-sessions/ws-timeout/harness-runs" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "run-timeout",
				"phase": "Starting",
			})

		case r.URL.Path == "/api/v1/harness-runs/run-timeout/agent-session" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-timeout",
				"runId":     "run-timeout",
				"agent":     "codex",
				"phase":     "Starting",
			})

		// Always return Starting — never Ready
		case r.URL.Path == "/api/v1/harness-runs/run-timeout/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-timeout",
				"runId":     "run-timeout",
				"agent":     "codex",
				"phase":     "Starting",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL, "--token", "test-token",
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "codex",
		"--timeout", "3s",
	}, &stdout, &stderr)

	if code == 0 {
		t.Fatalf("expected non-zero exit code for timeout, got 0")
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "timed out") {
		t.Errorf("expected timeout error, got:\n%s", errOut)
	}
}

func TestAgentStart_ReuseWorkspace(t *testing.T) {
	t.Setenv(EnvToken, "")

	var workspaceCreated atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Should NOT create workspace session
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			workspaceCreated.Store(true)
			http.Error(w, "unexpected workspace creation", http.StatusInternalServerError)
			return

		// Create harness run under existing workspace
		case r.URL.Path == "/api/v1/workspace-sessions/ws-existing/harness-runs" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "run-reuse",
				"workspaceSessionID": "ws-existing",
				"phase":              "Starting",
			})

		// Create agent session
		case r.URL.Path == "/api/v1/harness-runs/run-reuse/agent-session" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-reuse",
				"runId":     "run-reuse",
				"agent":     "opencode",
				"phase":     "Starting",
			})

		// Poll — immediately Ready
		case r.URL.Path == "/api/v1/harness-runs/run-reuse/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-reuse",
				"runId":     "run-reuse",
				"agent":     "opencode",
				"phase":     "Ready",
			})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL, "--token", "test-token",
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "opencode",
		"--workspace", "ws-existing",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if workspaceCreated.Load() {
		t.Fatal("should not create workspace session when --workspace is provided")
	}

	out := stdout.String()
	if !strings.Contains(out, "as-reuse") {
		t.Errorf("expected session ID in output, got:\n%s", out)
	}
	if !strings.Contains(out, "run-reuse") {
		t.Errorf("expected run ID in output, got:\n%s", out)
	}
}

func TestAgentStart_JSONOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "ws-json",
				"phase": "Active",
			})

		case r.URL.Path == "/api/v1/workspace-sessions/ws-json/harness-runs" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "run-json",
				"phase": "Starting",
			})

		case r.URL.Path == "/api/v1/harness-runs/run-json/agent-session" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-json",
				"runId":     "run-json",
				"agent":     "pi",
				"phase":     "Starting",
			})

		case r.URL.Path == "/api/v1/harness-runs/run-json/agent-session" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sessionId": "as-json",
				"runId":     "run-json",
				"agent":     "pi",
				"phase":     "Ready",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL, "--token", "test-token",
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "pi",
		"--output", "json",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}

	// Verify stdout is valid JSON
	var result agentStartResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nraw: %s", err, stdout.String())
	}

	if result.SessionID != "as-json" {
		t.Errorf("sessionId = %q, want as-json", result.SessionID)
	}
	if result.RunID != "run-json" {
		t.Errorf("runId = %q, want run-json", result.RunID)
	}
	if result.Agent != "pi" {
		t.Errorf("agent = %q, want pi", result.Agent)
	}
	if result.Phase != "Ready" {
		t.Errorf("phase = %q, want Ready", result.Phase)
	}

	// Verify progress messages are on stderr, not stdout
	if strings.Contains(stdout.String(), "Creating") {
		t.Error("progress messages should not appear in stdout when using JSON output")
	}
}
