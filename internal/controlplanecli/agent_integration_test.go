package controlplanecli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// mockAgentServer — reusable stateful mock for the full agent session API
// ---------------------------------------------------------------------------

// mockWorkspace is the server-side state for a workspace session.
type mockWorkspace struct {
	WorkspaceSession
	HarnessRuns map[string]*mockHarnessRun
}

// mockHarnessRun is the server-side state for a harness run.
type mockHarnessRun struct {
	HarnessRun
	AgentSession *AgentSession
	Events       []AgentSessionEvent
}

// mockAgentServer maintains in-memory state so integration tests can verify
// full lifecycle flows (create → list → prompt → events → stop).
type mockAgentServer struct {
	mu         sync.Mutex
	workspaces map[string]*mockWorkspace  // keyed by workspace ID
	runs       map[string]*mockHarnessRun // keyed by run ID
	nextID     int
}

func newMockAgentServer() *mockAgentServer {
	return &mockAgentServer{
		workspaces: make(map[string]*mockWorkspace),
		runs:       make(map[string]*mockHarnessRun),
	}
}

func (m *mockAgentServer) genID(prefix string) string {
	m.nextID++
	return fmt.Sprintf("%s-%d", prefix, m.nextID)
}

// ServeHTTP implements http.Handler and routes to the correct endpoint.
func (m *mockAgentServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	switch {
	// POST /api/v1/workspace-sessions — create workspace
	case p == "/api/v1/workspace-sessions" && r.Method == http.MethodPost:
		m.handleCreateWorkspace(w, r)

	// GET /api/v1/workspace-sessions — list workspaces
	case p == "/api/v1/workspace-sessions" && r.Method == http.MethodGet:
		m.handleListWorkspaces(w, r)

	// GET /api/v1/workspace-sessions/{id}/agent-sessions — list sessions
	case r.Method == http.MethodGet && strings.HasSuffix(p, "/agent-sessions") && strings.HasPrefix(p, "/api/v1/workspace-sessions/"):
		m.handleListAgentSessions(w, r)

	// POST /api/v1/workspace-sessions/{id}/harness-runs — create run
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/harness-runs") && strings.HasPrefix(p, "/api/v1/workspace-sessions/"):
		m.handleCreateHarnessRun(w, r)

	// POST /api/v1/harness-runs/{id}/agent-session — create session
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/agent-session") && !strings.HasSuffix(p, "/stop") && !strings.HasSuffix(p, "/prompt") && strings.HasPrefix(p, "/api/v1/harness-runs/"):
		m.handleCreateAgentSession(w, r)

	// GET /api/v1/harness-runs/{id}/agent-session — get session
	case r.Method == http.MethodGet && strings.HasSuffix(p, "/agent-session") && strings.HasPrefix(p, "/api/v1/harness-runs/"):
		m.handleGetAgentSession(w, r)

	// POST /api/v1/harness-runs/{id}/agent-session/stop — stop session
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/agent-session/stop"):
		m.handleStopAgentSession(w, r)

	// POST /api/v1/harness-runs/{id}/agent-session/prompt — send prompt
	case r.Method == http.MethodPost && strings.HasSuffix(p, "/agent-session/prompt"):
		m.handleSendPrompt(w, r)

	// GET /api/v1/harness-runs/{id}/agent-session/events — get events
	case r.Method == http.MethodGet && strings.HasSuffix(p, "/agent-session/events") && !strings.HasSuffix(p, "/stream"):
		m.handleGetEvents(w, r)

	// GET /api/v1/harness-runs/{id}/agent-session/events/stream — SSE stream
	case r.Method == http.MethodGet && strings.HasSuffix(p, "/agent-session/events/stream"):
		m.handleStreamEvents(w, r)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf("not found: %s %s", r.Method, p)})
	}
}

// extractPathSegment extracts a path segment by position from a known prefix.
// e.g. "/api/v1/workspace-sessions/ws-1/harness-runs" → "ws-1" at segment index 4.
func extractPathSegment(path string, index int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if index < 0 || index >= len(parts) {
		return ""
	}
	return parts[index]
}

// extractRunID extracts the run ID from paths like /api/v1/harness-runs/{id}/...
func extractRunID(path string) string {
	return extractPathSegment(path, 3) // api/v1/harness-runs/{id}
}

// extractWorkspaceID extracts the workspace ID from paths like /api/v1/workspace-sessions/{id}/...
func extractWorkspaceID(path string) string {
	return extractPathSegment(path, 3) // api/v1/workspace-sessions/{id}
}

func (m *mockAgentServer) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var req struct {
		DisplayName string `json:"displayName"`
		RepoURL     string `json:"repoURL"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	id := m.genID("ws")
	ws := &mockWorkspace{
		WorkspaceSession: WorkspaceSession{
			ID:          id,
			DisplayName: req.DisplayName,
			RepoURL:     req.RepoURL,
			Phase:       "Active",
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		HarnessRuns: make(map[string]*mockHarnessRun),
	}
	m.workspaces[id] = ws

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ws.WorkspaceSession)
}

func (m *mockAgentServer) handleListWorkspaces(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sessions []WorkspaceSession
	for _, ws := range m.workspaces {
		sessions = append(sessions, ws.WorkspaceSession)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"workspaceSessions": sessions})
}

func (m *mockAgentServer) handleListAgentSessions(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wsID := extractWorkspaceID(r.URL.Path)
	ws, ok := m.workspaces[wsID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "workspace not found"})
		return
	}

	var sessions []AgentSession
	for _, run := range ws.HarnessRuns {
		if run.AgentSession != nil {
			sessions = append(sessions, *run.AgentSession)
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"agentSessions": sessions})
}

func (m *mockAgentServer) handleCreateHarnessRun(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wsID := extractWorkspaceID(r.URL.Path)
	ws, ok := m.workspaces[wsID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "workspace not found"})
		return
	}

	var req struct {
		RepoURL      string `json:"repoURL"`
		RepoRevision string `json:"repoRevision"`
		Image        string `json:"image"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	id := m.genID("run")
	run := &mockHarnessRun{
		HarnessRun: HarnessRun{
			ID:                 id,
			WorkspaceSessionID: wsID,
			RepoURL:            req.RepoURL,
			RepoRevision:       req.RepoRevision,
			Image:              req.Image,
			Phase:              "Starting",
		},
	}
	ws.HarnessRuns[id] = run
	m.runs[id] = run

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run.HarnessRun)
}

func (m *mockAgentServer) handleCreateAgentSession(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}
	if run.AgentSession != nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session already exists"})
		return
	}

	sessID := m.genID("as")
	session := &AgentSession{
		SessionID:   sessID,
		RunID:       runID,
		DisplayName: "agent-" + sessID,
		Runtime:     "opencode",
		Agent:       "claude",
		Phase:       "Ready",
		WorkspaceID: run.WorkspaceSessionID,
		CreatedAt:   time.Now().UTC(),
	}
	run.AgentSession = session
	run.Phase = "Running"

	// Seed an initial event.
	run.Events = append(run.Events, AgentSessionEvent{
		Seq:       1,
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"type":"session_started"}`),
	})

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(session)
}

func (m *mockAgentServer) handleGetAgentSession(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}
	if run.AgentSession == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "no agent session for this run"})
		return
	}
	_ = json.NewEncoder(w).Encode(run.AgentSession)
}

func (m *mockAgentServer) handleStopAgentSession(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}
	if run.AgentSession == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "no agent session for this run"})
		return
	}
	if run.AgentSession.Phase == "Stopped" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session already stopped"})
		return
	}

	run.AgentSession.Phase = "Stopped"
	run.Phase = "Succeeded"

	run.Events = append(run.Events, AgentSessionEvent{
		Seq:       len(run.Events) + 1,
		Timestamp: time.Now().UTC(),
		Data:      json.RawMessage(`{"type":"session_stopped"}`),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (m *mockAgentServer) handleSendPrompt(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}
	if run.AgentSession == nil || run.AgentSession.Phase == "Stopped" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session not running"})
		return
	}

	var req PromptRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Generate a response event.
	now := time.Now().UTC()
	respEvent := AgentSessionEvent{
		Seq:       len(run.Events) + 1,
		Timestamp: now,
		Data:      json.RawMessage(fmt.Sprintf(`{"type":"response","text":"echo: %s"}`, req.Prompt)),
	}
	run.Events = append(run.Events, respEvent)

	_ = json.NewEncoder(w).Encode(PromptResponse{Events: []AgentSessionEvent{respEvent}})
}

func (m *mockAgentServer) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}

	events := run.Events
	if events == nil {
		events = []AgentSessionEvent{}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
}

func (m *mockAgentServer) handleStreamEvents(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	runID := extractRunID(r.URL.Path)
	run, ok := m.runs[runID]
	if !ok {
		m.mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "harness run not found"})
		return
	}

	// Snapshot current events under lock.
	events := make([]AgentSessionEvent, len(run.Events))
	copy(events, run.Events)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	for _, ev := range events {
		b, _ := json.Marshal(ev)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	}

	// Flush if possible.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// startMockServer creates an httptest.Server backed by a mockAgentServer.
func startMockServer(t *testing.T) (*httptest.Server, *mockAgentServer) {
	t.Helper()
	mock := newMockAgentServer()
	srv := httptest.NewServer(mock)
	t.Cleanup(srv.Close)
	return srv, mock
}

// runCLI is a test helper that invokes the CLI Main function with the given
// args prepended by the server URL and a test token.
func runCLI(t *testing.T, serverURL string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	t.Setenv(EnvToken, "")

	fullArgs := append([]string{"--api-url", serverURL, "--token", "test-token"}, args...)
	var outBuf, errBuf bytes.Buffer
	exitCode = Main(fullArgs, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), exitCode
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

// TestFullLifecycle_StartExecStop exercises the full agent lifecycle:
// start an agent, send a prompt, verify events, then stop it.
func TestFullLifecycle_StartExecStop(t *testing.T) {
	srv, mock := startMockServer(t)

	// --- Step 1: Start an agent ---
	stdout, stderr, code := runCLI(t, srv.URL,
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "claude",
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent start: exit=%d stderr=%s", code, stderr)
	}

	var startResult agentStartResult
	if err := json.Unmarshal([]byte(stdout), &startResult); err != nil {
		t.Fatalf("parse start result: %v\nraw: %s", err, stdout)
	}
	if startResult.RunID == "" {
		t.Fatal("start result missing RunID")
	}
	if startResult.SessionID == "" {
		t.Fatal("start result missing SessionID")
	}
	if startResult.Phase != "Ready" {
		t.Fatalf("start result Phase = %q, want Ready", startResult.Phase)
	}

	runID := startResult.RunID

	// Verify mock state: one workspace, one run, one session.
	mock.mu.Lock()
	if len(mock.workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(mock.workspaces))
	}
	if len(mock.runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(mock.runs))
	}
	run := mock.runs[runID]
	if run == nil {
		t.Fatalf("run %q not found in mock state", runID)
	}
	if run.AgentSession == nil {
		t.Fatal("expected agent session to exist")
	}
	if run.AgentSession.Phase != "Ready" {
		t.Fatalf("agent session Phase = %q, want Ready", run.AgentSession.Phase)
	}
	initialEventCount := len(run.Events)
	mock.mu.Unlock()

	// --- Step 2: Send a prompt ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "exec", runID, "--prompt", "hello world", "--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent exec: exit=%d stderr=%s", code, stderr)
	}

	var promptResp PromptResponse
	if err := json.Unmarshal([]byte(stdout), &promptResp); err != nil {
		t.Fatalf("parse exec result: %v\nraw: %s", err, stdout)
	}
	if len(promptResp.Events) != 1 {
		t.Fatalf("expected 1 response event, got %d", len(promptResp.Events))
	}
	if !strings.Contains(string(promptResp.Events[0].Data), "echo: hello world") {
		t.Fatalf("response event missing echo, got: %s", string(promptResp.Events[0].Data))
	}

	// --- Step 3: Verify events accumulated ---
	mock.mu.Lock()
	currentEventCount := len(run.Events)
	mock.mu.Unlock()
	if currentEventCount <= initialEventCount {
		t.Fatalf("expected events to grow: initial=%d current=%d", initialEventCount, currentEventCount)
	}

	// Verify events via the CLI logs command.
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "logs", runID, "--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent logs: exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "session_started") {
		t.Errorf("logs output missing session_started event, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "echo: hello world") {
		t.Errorf("logs output missing prompt response event, got:\n%s", stdout)
	}

	// --- Step 4: Stop the agent ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "stop", runID,
	)
	if code != 0 {
		t.Fatalf("agent stop: exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Agent session stopped") {
		t.Errorf("expected 'Agent session stopped' in output, got:\n%s", stdout)
	}

	// Verify mock state: session is stopped.
	mock.mu.Lock()
	if run.AgentSession.Phase != "Stopped" {
		t.Fatalf("agent session Phase = %q, want Stopped", run.AgentSession.Phase)
	}
	mock.mu.Unlock()

	// Verify status shows Stopped.
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "status", runID,
	)
	if code != 0 {
		t.Fatalf("agent status after stop: exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Stopped") {
		t.Errorf("expected Stopped in status output, got:\n%s", stdout)
	}
}

// TestFullLifecycle_StartListStatus starts an agent, verifies it appears in
// the list output, and checks that status returns the correct details.
func TestFullLifecycle_StartListStatus(t *testing.T) {
	srv, _ := startMockServer(t)

	// --- Start an agent ---
	stdout, stderr, code := runCLI(t, srv.URL,
		"agent", "start",
		"--repo", "https://github.com/example/repo",
		"--agent", "codex",
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent start: exit=%d stderr=%s", code, stderr)
	}

	var startResult agentStartResult
	if err := json.Unmarshal([]byte(stdout), &startResult); err != nil {
		t.Fatalf("parse start result: %v\nraw: %s", err, stdout)
	}
	runID := startResult.RunID
	sessionID := startResult.SessionID

	// --- List agents ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "list", "--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent list: exit=%d stderr=%s", code, stderr)
	}

	var sessions []AgentSession
	if err := json.Unmarshal([]byte(stdout), &sessions); err != nil {
		t.Fatalf("parse list result: %v\nraw: %s", err, stdout)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session in list, got %d", len(sessions))
	}
	if sessions[0].SessionID != sessionID {
		t.Fatalf("list session ID = %q, want %q", sessions[0].SessionID, sessionID)
	}
	if sessions[0].Phase != "Ready" {
		t.Fatalf("list session Phase = %q, want Ready", sessions[0].Phase)
	}

	// --- Check status ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "status", runID, "--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent status: exit=%d stderr=%s", code, stderr)
	}

	var statusSession AgentSession
	if err := json.Unmarshal([]byte(stdout), &statusSession); err != nil {
		t.Fatalf("parse status result: %v\nraw: %s", err, stdout)
	}
	if statusSession.SessionID != sessionID {
		t.Fatalf("status SessionID = %q, want %q", statusSession.SessionID, sessionID)
	}
	if statusSession.RunID != runID {
		t.Fatalf("status RunID = %q, want %q", statusSession.RunID, runID)
	}
	if statusSession.Phase != "Ready" {
		t.Fatalf("status Phase = %q, want Ready", statusSession.Phase)
	}

	// --- Verify table output too ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "list",
	)
	if code != 0 {
		t.Fatalf("agent list table: exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, sessionID) {
		t.Errorf("table output missing session ID %q, got:\n%s", sessionID, stdout)
	}
	if !strings.Contains(stdout, "Ready") {
		t.Errorf("table output missing Ready phase, got:\n%s", stdout)
	}
}

// TestFullLifecycle_MultipleAgents starts two agents in different workspaces
// and verifies that listing shows both.
func TestFullLifecycle_MultipleAgents(t *testing.T) {
	srv, mock := startMockServer(t)

	// --- Start agent 1 ---
	stdout, stderr, code := runCLI(t, srv.URL,
		"agent", "start",
		"--repo", "https://github.com/example/repo-alpha",
		"--agent", "claude",
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent start 1: exit=%d stderr=%s", code, stderr)
	}

	var start1 agentStartResult
	if err := json.Unmarshal([]byte(stdout), &start1); err != nil {
		t.Fatalf("parse start1: %v\nraw: %s", err, stdout)
	}

	// --- Start agent 2 ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "start",
		"--repo", "https://github.com/example/repo-beta",
		"--agent", "codex",
		"--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent start 2: exit=%d stderr=%s", code, stderr)
	}

	var start2 agentStartResult
	if err := json.Unmarshal([]byte(stdout), &start2); err != nil {
		t.Fatalf("parse start2: %v\nraw: %s", err, stdout)
	}

	// Verify we have 2 distinct workspaces and 2 runs.
	mock.mu.Lock()
	wsCount := len(mock.workspaces)
	runCount := len(mock.runs)
	mock.mu.Unlock()

	if wsCount != 2 {
		t.Fatalf("expected 2 workspaces, got %d", wsCount)
	}
	if runCount != 2 {
		t.Fatalf("expected 2 runs, got %d", runCount)
	}

	// Verify session IDs are distinct.
	if start1.SessionID == start2.SessionID {
		t.Fatalf("expected distinct session IDs, both are %q", start1.SessionID)
	}
	if start1.RunID == start2.RunID {
		t.Fatalf("expected distinct run IDs, both are %q", start1.RunID)
	}

	// --- List all agents ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "list", "--output", "json",
	)
	if code != 0 {
		t.Fatalf("agent list: exit=%d stderr=%s", code, stderr)
	}

	var sessions []AgentSession
	if err := json.Unmarshal([]byte(stdout), &sessions); err != nil {
		t.Fatalf("parse list: %v\nraw: %s", err, stdout)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions in list, got %d", len(sessions))
	}

	// Collect session IDs from the list.
	listedIDs := make(map[string]bool)
	for _, s := range sessions {
		listedIDs[s.SessionID] = true
	}
	if !listedIDs[start1.SessionID] {
		t.Errorf("list missing session %q", start1.SessionID)
	}
	if !listedIDs[start2.SessionID] {
		t.Errorf("list missing session %q", start2.SessionID)
	}

	// --- Verify table output contains both ---
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "list",
	)
	if code != 0 {
		t.Fatalf("agent list table: exit=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, start1.SessionID) {
		t.Errorf("table output missing session %q, got:\n%s", start1.SessionID, stdout)
	}
	if !strings.Contains(stdout, start2.SessionID) {
		t.Errorf("table output missing session %q, got:\n%s", start2.SessionID, stdout)
	}

	// --- Stop agent 1, verify agent 2 still Running ---
	_, stderr, code = runCLI(t, srv.URL,
		"agent", "stop", start1.RunID,
	)
	if code != 0 {
		t.Fatalf("agent stop 1: exit=%d stderr=%s", code, stderr)
	}

	// Status of agent 1 should be Stopped.
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "status", start1.RunID, "--output", "json",
	)
	if code != 0 {
		t.Fatalf("status agent 1: exit=%d stderr=%s", code, stderr)
	}
	var status1 AgentSession
	if err := json.Unmarshal([]byte(stdout), &status1); err != nil {
		t.Fatalf("parse status1: %v", err)
	}
	if status1.Phase != "Stopped" {
		t.Fatalf("agent 1 Phase = %q, want Stopped", status1.Phase)
	}

	// Status of agent 2 should still be Ready.
	stdout, stderr, code = runCLI(t, srv.URL,
		"agent", "status", start2.RunID, "--output", "json",
	)
	if code != 0 {
		t.Fatalf("status agent 2: exit=%d stderr=%s", code, stderr)
	}
	var status2 AgentSession
	if err := json.Unmarshal([]byte(stdout), &status2); err != nil {
		t.Fatalf("parse status2: %v", err)
	}
	if status2.Phase != "Ready" {
		t.Fatalf("agent 2 Phase = %q, want Ready", status2.Phase)
	}
}
