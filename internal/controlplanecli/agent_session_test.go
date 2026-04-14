package controlplanecli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	c, err := NewClient(Config{
		BaseURL: serverURL,
		Token:   "test-token",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestListAgentSessions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/workspace-sessions/ws-1/agent-sessions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agentSessions": []map[string]any{
				{"sessionId": "as-1", "runId": "run-1", "displayName": "demo", "runtime": "opencode", "agent": "claude", "phase": "Running", "workspaceSessionId": "ws-1"},
				{"sessionId": "as-2", "runId": "run-2", "displayName": "test", "runtime": "opencode", "agent": "gpt4", "phase": "Stopped", "workspaceSessionId": "ws-1"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	sessions, err := client.ListAgentSessions(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("ListAgentSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	if sessions[0].SessionID != "as-1" {
		t.Fatalf("sessions[0].SessionID = %q, want as-1", sessions[0].SessionID)
	}
	if sessions[0].Runtime != "opencode" {
		t.Fatalf("sessions[0].Runtime = %q, want opencode", sessions[0].Runtime)
	}
	if sessions[1].Phase != "Stopped" {
		t.Fatalf("sessions[1].Phase = %q, want Stopped", sessions[1].Phase)
	}
}

func TestGetAgentSession(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-42/agent-session" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId":          "as-42",
			"runId":              "run-42",
			"displayName":        "my-agent",
			"runtime":            "opencode",
			"agent":              "claude",
			"phase":              "Running",
			"workspaceSessionId": "ws-5",
			"createdAt":          now.Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	session, err := client.GetAgentSession(context.Background(), "run-42")
	if err != nil {
		t.Fatalf("GetAgentSession: %v", err)
	}
	if session.SessionID != "as-42" {
		t.Fatalf("SessionID = %q, want as-42", session.SessionID)
	}
	if session.RunID != "run-42" {
		t.Fatalf("RunID = %q, want run-42", session.RunID)
	}
	if session.DisplayName != "my-agent" {
		t.Fatalf("DisplayName = %q, want my-agent", session.DisplayName)
	}
	if session.Runtime != "opencode" {
		t.Fatalf("Runtime = %q, want opencode", session.Runtime)
	}
	if session.Agent != "claude" {
		t.Fatalf("Agent = %q, want claude", session.Agent)
	}
	if session.Phase != "Running" {
		t.Fatalf("Phase = %q, want Running", session.Phase)
	}
	if session.WorkspaceID != "ws-5" {
		t.Fatalf("WorkspaceID = %q, want ws-5", session.WorkspaceID)
	}
	if session.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func TestCreateAgentSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-10/agent-session" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		// CreateAgentSession sends no body, so Content-Type should be absent
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Fatalf("expected no Content-Type header for bodyless POST, got %q", ct)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId":          "as-new",
			"runId":              "run-10",
			"displayName":        "new-session",
			"runtime":            "opencode",
			"agent":              "claude",
			"phase":              "Starting",
			"workspaceSessionId": "ws-3",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	session, err := client.CreateAgentSession(context.Background(), "run-10")
	if err != nil {
		t.Fatalf("CreateAgentSession: %v", err)
	}
	if session.SessionID != "as-new" {
		t.Fatalf("SessionID = %q, want as-new", session.SessionID)
	}
	if session.Phase != "Starting" {
		t.Fatalf("Phase = %q, want Starting", session.Phase)
	}
}

func TestStopAgentSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-99/agent-session/stop" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	err := client.StopAgentSession(context.Background(), "run-99")
	if err != nil {
		t.Fatalf("StopAgentSession: %v", err)
	}
}

func TestSendPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-7/agent-session/prompt" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Prompt != "hello world" {
			t.Fatalf("prompt = %q, want hello world", req.Prompt)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"seq": 1, "timestamp": "2025-06-15T10:30:00Z", "data": map[string]any{"type": "response", "text": "hi"}},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	resp, err := client.SendPrompt(context.Background(), "run-7", "hello world")
	if err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(resp.Events))
	}
	if resp.Events[0].Seq != 1 {
		t.Fatalf("events[0].Seq = %d, want 1", resp.Events[0].Seq)
	}
}

func TestStreamEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-5/agent-session/events/stream" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"seq\":1}\n\ndata: {\"seq\":2}\n\n"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	rc, err := client.StreamEvents(context.Background(), "run-5")
	if err != nil {
		t.Fatalf("StreamEvents: %v", err)
	}
	defer func() { _ = rc.Close() }()

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(body), `"seq":1`) {
		t.Fatalf("body missing seq:1: %s", body)
	}
	if !strings.Contains(string(body), `"seq":2`) {
		t.Fatalf("body missing seq:2: %s", body)
	}
}

func TestGetEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/harness-runs/run-3/agent-session/events" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"seq": 1, "timestamp": "2025-06-15T10:30:00Z", "data": map[string]any{"type": "start"}},
				{"seq": 2, "timestamp": "2025-06-15T10:30:01Z", "data": map[string]any{"type": "output", "text": "hello"}},
				{"seq": 3, "timestamp": "2025-06-15T10:30:02Z", "data": map[string]any{"type": "end"}},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	events, err := client.GetEvents(context.Background(), "run-3")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].Seq != 1 {
		t.Fatalf("events[0].Seq = %d, want 1", events[0].Seq)
	}
	if events[2].Seq != 3 {
		t.Fatalf("events[2].Seq = %d, want 3", events[2].Seq)
	}
}

func TestStartAgent(t *testing.T) {
	var (
		createdWorkspaceSession bool
		createdHarnessRun       bool
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Create workspace session
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			createdWorkspaceSession = true
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["repoURL"] != "https://github.com/example/repo" {
				t.Fatalf("repoURL = %v", req["repoURL"])
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "ws-new",
				"repoURL": "https://github.com/example/repo",
				"phase":   "Active",
			})

		// Create harness run — verify agentSession is included
		case r.URL.Path == "/api/v1/workspace-sessions/ws-new/harness-runs" && r.Method == http.MethodPost:
			createdHarnessRun = true
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["repoURL"] != "https://github.com/example/repo" {
				t.Fatalf("run repoURL = %v", req["repoURL"])
			}
			if req["image"] != "ghcr.io/example/harness:latest" {
				t.Fatalf("run image = %v", req["image"])
			}
			if req["repoRevision"] != "main" {
				t.Fatalf("run repoRevision = %v", req["repoRevision"])
			}
			agentSession, ok := req["agentSession"].(map[string]any)
			if !ok {
				t.Fatal("expected agentSession in harness run request")
			}
			if agentSession["agent"] != "claude" {
				t.Fatalf("agentSession.agent = %v, want claude", agentSession["agent"])
			}
			if agentSession["runtime"] != "sandbox-agent" {
				t.Fatalf("agentSession.runtime = %v, want sandbox-agent", agentSession["runtime"])
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "run-new",
				"workspaceSessionID": "ws-new",
				"repoURL":            "https://github.com/example/repo",
				"image":              "ghcr.io/example/harness:latest",
				"phase":              "Starting",
			})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	runID, err := client.StartAgent(context.Background(), "", "https://github.com/example/repo", "main", "claude", "ghcr.io/example/harness:latest", nil, nil, "")
	if err != nil {
		t.Fatalf("StartAgent: %v", err)
	}
	if runID != "run-new" {
		t.Fatalf("runID = %q, want run-new", runID)
	}
	if !createdWorkspaceSession {
		t.Fatal("expected workspace session creation")
	}
	if !createdHarnessRun {
		t.Fatal("expected harness run creation")
	}
}

func TestStartAgentWithExistingWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Should NOT create workspace session — skip straight to harness run
		case r.URL.Path == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
			t.Fatal("should not create workspace session when ID is provided")

		case r.URL.Path == "/api/v1/workspace-sessions/ws-existing/harness-runs" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                 "run-existing",
				"workspaceSessionID": "ws-existing",
				"repoURL":            "https://github.com/example/repo",
				"image":              "ghcr.io/example/harness:latest",
				"phase":              "Starting",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	runID, err := client.StartAgent(context.Background(), "ws-existing", "https://github.com/example/repo", "main", "claude", "ghcr.io/example/harness:latest", nil, nil, "")
	if err != nil {
		t.Fatalf("StartAgent: %v", err)
	}
	if runID != "run-existing" {
		t.Fatalf("runID = %q, want run-existing", runID)
	}
}

func TestStreamEventsReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.StreamEvents(context.Background(), "run-missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	var apiErr *APIError
	if !isAPIError(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

// isAPIError is a test helper that checks if err is an *APIError.
func isAPIError(err error, target **APIError) bool {
	return errors.As(err, target)
}
