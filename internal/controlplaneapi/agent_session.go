package controlplaneapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	agentSessionBlockerProvisioning          = "provisioning"
	agentSessionBlockerSandboxAgentReadiness = "sandbox-agent-readiness"
	agentSessionBlockerAuth                  = "auth"
	agentSessionBlockerRepoAccess            = "repo-access"
	agentSessionBlockerNetwork               = "network"
	agentSessionBlockerImagePull             = "image-pull"
)

type jsonRPCEnvelope struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  any             `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type agentSessionTransport interface {
	PostACP(ctx context.Context, podName, serverID, bootstrapAgent string, payload any) ([]byte, error)
	StreamACP(ctx context.Context, podName, serverID string) (io.ReadCloser, error)
	DeleteACP(ctx context.Context, podName, serverID string) error
}

type podProxyAgentSessionTransport struct {
	namespace string
	clientset *http.Client
	baseURL   *url.URL
	token     string
}

func newPodProxyAgentSessionTransport(namespace string, restClient *http.Client, baseURL *url.URL, token string) *podProxyAgentSessionTransport {
	return &podProxyAgentSessionTransport{namespace: namespace, clientset: restClient, baseURL: baseURL, token: token}
}

func (t *podProxyAgentSessionTransport) acpURL(podName, serverID string, bootstrapAgent string) string {
	u := *t.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/namespaces/" + url.PathEscape(t.namespace) + "/pods/" + url.PathEscape(podName) + ":2468/proxy/v1/acp/" + url.PathEscape(serverID)
	q := url.Values{}
	if bootstrapAgent != "" {
		q.Set("agent", bootstrapAgent)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (t *podProxyAgentSessionTransport) do(ctx context.Context, method, urlStr string, accept string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := t.clientset.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("sandbox-agent proxy %s %s returned %d: %s", method, urlStr, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return bodyBytes, nil
}

func (t *podProxyAgentSessionTransport) PostACP(ctx context.Context, podName, serverID, bootstrapAgent string, payload any) ([]byte, error) {
	return t.do(ctx, http.MethodPost, t.acpURL(podName, serverID, bootstrapAgent), "application/json", payload)
}

func (t *podProxyAgentSessionTransport) StreamACP(ctx context.Context, podName, serverID string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.acpURL(podName, serverID, ""), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := t.clientset.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer func() { _ = resp.Body.Close() }()
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		return nil, fmt.Errorf("sandbox-agent stream GET returned %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return resp.Body, nil
}

func (t *podProxyAgentSessionTransport) DeleteACP(ctx context.Context, podName, serverID string) error {
	_, err := t.do(ctx, http.MethodDelete, t.acpURL(podName, serverID, ""), "application/json", nil)
	return err
}

type agentSessionEvent struct {
	Sequence int64           `json:"seq"`
	At       time.Time       `json:"timestamp"`
	Envelope json.RawMessage `json:"data"`
}

type agentSessionState struct {
	HarnessRunID string                             `json:"harnessRunID"`
	PodName      string                             `json:"podName,omitempty"`
	ServerID     string                             `json:"serverID,omitempty"`
	Runtime      operatorv1alpha1.AgentRuntime      `json:"runtime,omitempty"`
	Agent        operatorv1alpha1.AgentKind         `json:"agent,omitempty"`
	SessionID    string                             `json:"sessionId,omitempty"`
	Phase        operatorv1alpha1.AgentSessionPhase `json:"phase,omitempty"`
	LastSequence int64                              `json:"lastSequence,omitempty"`
}

type agentSessionOperationError struct {
	statusCode int
	message    string
}

func (e *agentSessionOperationError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

type agentSessionBridge struct {
	mu           sync.Mutex
	createMu     sync.Mutex
	runID        string
	podName      string
	serverID     string
	runtime      operatorv1alpha1.AgentRuntime
	agent        operatorv1alpha1.AgentKind
	sessionID    string
	phase        operatorv1alpha1.AgentSessionPhase
	events       []agentSessionEvent
	nextSeq      int64
	subscribers  map[chan agentSessionEvent]struct{}
	streaming    bool
	streamCancel context.CancelFunc
	streamBody   io.Closer
	streamDone   chan struct{}
	promptSeq    atomic.Int64
}

func normalizeAgentSessionState(state agentSessionState) agentSessionState {
	state.Phase = operatorv1alpha1.NormalizeAgentSessionPhase(string(state.Phase))
	return state
}

func resolveAgentSessionPhase(current, candidate operatorv1alpha1.AgentSessionPhase) operatorv1alpha1.AgentSessionPhase {
	current = operatorv1alpha1.NormalizeAgentSessionPhase(string(current))
	candidate = operatorv1alpha1.NormalizeAgentSessionPhase(string(candidate))
	switch {
	case candidate == "":
		return current
	case current == "":
		return candidate
	case current == candidate:
		return current
	case current.IsTerminal():
		return current
	case candidate.IsTerminal():
		return candidate
	case current.CanTransitionTo(candidate):
		return candidate
	case candidate.CanTransitionTo(current):
		return current
	default:
		return current
	}
}

func (b *agentSessionBridge) transitionLocked(next operatorv1alpha1.AgentSessionPhase) bool {
	next = operatorv1alpha1.NormalizeAgentSessionPhase(string(next))
	current := operatorv1alpha1.NormalizeAgentSessionPhase(string(b.phase))
	if !current.CanTransitionTo(next) {
		return false
	}
	b.phase = next
	return true
}

func (b *agentSessionBridge) snapshot() agentSessionState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return normalizeAgentSessionState(agentSessionState{
		HarnessRunID: b.runID,
		PodName:      b.podName,
		ServerID:     b.serverID,
		Runtime:      b.runtime,
		Agent:        b.agent,
		SessionID:    b.sessionID,
		Phase:        b.phase,
		LastSequence: b.nextSeq,
	})
}

func (b *agentSessionBridge) appendEvent(raw json.RawMessage) agentSessionEvent {
	sanitized := sanitizeAgentSessionEnvelope(raw)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextSeq++
	event := agentSessionEvent{Sequence: b.nextSeq, At: time.Now().UTC(), Envelope: sanitized}
	b.events = append(b.events, event)
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	return event
}

func (b *agentSessionBridge) list(offset int64, limit int) ([]agentSessionEvent, int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	out := make([]agentSessionEvent, 0, limit)
	for _, event := range b.events {
		if event.Sequence <= offset {
			continue
		}
		out = append(out, event)
		if len(out) >= limit {
			break
		}
	}
	return out, b.nextSeq
}

func (b *agentSessionBridge) subscribe() (chan agentSessionEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan agentSessionEvent, 32)
	if b.subscribers == nil {
		b.subscribers = map[chan agentSessionEvent]struct{}{}
	}
	b.subscribers[ch] = struct{}{}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers, ch)
		close(ch)
	}
}

func agentSessionStateFromHarnessRun(run *operatorv1alpha1.HarnessRun) (agentSessionState, bool) {
	state := agentSessionState{
		HarnessRunID: run.Name,
		PodName:      run.Status.PodName,
		ServerID:     run.Name,
	}
	if run.Spec.AgentSession != nil {
		state.Runtime = run.Spec.AgentSession.Runtime
		state.Agent = run.Spec.AgentSession.Agent
	}
	statusPresent := run.Status.AgentSession != nil
	if !statusPresent {
		return normalizeAgentSessionState(state), false
	}
	if run.Status.AgentSession.Runtime != "" {
		state.Runtime = run.Status.AgentSession.Runtime
	}
	if run.Status.AgentSession.Agent != "" {
		state.Agent = run.Status.AgentSession.Agent
	}
	state.SessionID = strings.TrimSpace(run.Status.AgentSession.SessionID)
	state.Phase = operatorv1alpha1.NormalizeAgentSessionPhase(string(run.Status.AgentSession.Phase))
	return normalizeAgentSessionState(state), true
}

func mergePersistedAgentSessionState(base, persisted agentSessionState, statusPresent bool) agentSessionState {
	persisted = normalizeAgentSessionState(persisted)
	if base.Runtime == "" {
		base.Runtime = persisted.Runtime
	}
	if base.Agent == "" {
		base.Agent = persisted.Agent
	}
	if base.SessionID == "" {
		base.SessionID = persisted.SessionID
	}
	if !statusPresent {
		base.Phase = resolveAgentSessionPhase(base.Phase, persisted.Phase)
	}
	if persisted.LastSequence > base.LastSequence {
		base.LastSequence = persisted.LastSequence
	}
	return normalizeAgentSessionState(base)
}

type AgentSessionService struct {
	transport  agentSessionTransport
	store      *AgentSessionStore
	serviceCtx context.Context

	mu      sync.Mutex
	bridges map[string]*agentSessionBridge
}

func newAgentSessionService(transport agentSessionTransport, store *AgentSessionStore) *AgentSessionService {
	if store == nil {
		store = newAgentSessionStore("")
	}
	return &AgentSessionService{transport: transport, store: store, serviceCtx: context.Background(), bridges: map[string]*agentSessionBridge{}}
}

func newAgentSessionServiceWithContext(ctx context.Context, transport agentSessionTransport, store *AgentSessionStore) *AgentSessionService {
	if store == nil {
		store = newAgentSessionStore("")
	}
	return &AgentSessionService{transport: transport, store: store, serviceCtx: ctx, bridges: map[string]*agentSessionBridge{}}
}

func (s *AgentSessionService) bridgeFor(run *operatorv1alpha1.HarnessRun) *agentSessionBridge {
	s.mu.Lock()
	defer s.mu.Unlock()
	statusState, statusPresent := agentSessionStateFromHarnessRun(run)
	bridge, ok := s.bridges[run.Name]
	if ok {
		bridge.mu.Lock()
		bridge.podName = statusState.PodName
		bridge.runtime = statusState.Runtime
		bridge.agent = statusState.Agent
		if statusState.SessionID != "" {
			bridge.sessionID = statusState.SessionID
		}
		if statusPresent {
			bridge.phase = statusState.Phase
		} else if statusState.Phase != "" {
			bridge.phase = resolveAgentSessionPhase(bridge.phase, statusState.Phase)
		}
		bridge.mu.Unlock()
		return bridge
	}

	phase := statusState.Phase
	if phase == "" {
		phase = operatorv1alpha1.AgentSessionPhaseProvisioning
	}
	bridge = &agentSessionBridge{
		runID:       run.Name,
		podName:     statusState.PodName,
		serverID:    statusState.ServerID,
		runtime:     statusState.Runtime,
		agent:       statusState.Agent,
		sessionID:   statusState.SessionID,
		phase:       phase,
		subscribers: map[chan agentSessionEvent]struct{}{},
	}

	if persisted, ok := s.store.LoadState(run.Name); ok {
		merged := mergePersistedAgentSessionState(bridge.snapshot(), persisted, statusPresent)
		bridge.podName = merged.PodName
		bridge.serverID = merged.ServerID
		bridge.runtime = merged.Runtime
		bridge.agent = merged.Agent
		bridge.sessionID = merged.SessionID
		bridge.phase = merged.Phase
		if persisted.LastSequence > bridge.nextSeq {
			bridge.nextSeq = persisted.LastSequence
		}
	}

	resumedFrom := strings.TrimSpace(run.Labels["kocao.withakay.github.com/resumed-from"])
	if _, next, ok := s.store.ListEvents(0, 1, run.Name, resumedFrom); ok && next > bridge.nextSeq {
		bridge.nextSeq = next
	}

	s.bridges[run.Name] = bridge
	return bridge
}

func (s *AgentSessionService) EnsureSession(ctx context.Context, run *operatorv1alpha1.HarnessRun) (agentSessionState, error) {
	if run.Spec.AgentSession == nil || !run.Spec.AgentSession.Enabled() {
		return agentSessionState{}, fmt.Errorf("harness run is not configured for agentSession")
	}
	if strings.TrimSpace(run.Status.PodName) == "" {
		return agentSessionState{}, fmt.Errorf("harness run pod is not ready yet")
	}
	bridge := s.bridgeFor(run)

	// Serialize session creation to prevent two concurrent callers from both
	// seeing SessionID=="" and racing to create duplicate sessions.
	bridge.createMu.Lock()
	defer bridge.createMu.Unlock()

	state := bridge.snapshot()
	if state.SessionID != "" {
		if state.Phase.IsTerminal() {
			return state, nil
		}
		s.ensureStreaming(ctx, bridge)
		return state, nil
	}
	bridge.mu.Lock()
	if !bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseProvisioning) {
		bridge.phase = resolveAgentSessionPhase(bridge.phase, operatorv1alpha1.AgentSessionPhaseProvisioning)
	}
	bridge.mu.Unlock()
	state = bridge.snapshot()
	s.store.SaveState(state)

	initEnv := jsonRPCEnvelope{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": 1,
			"clientInfo": map[string]any{
				"name":    "kocao-control-plane",
				"version": "dev",
			},
		},
	}
	body, err := s.transport.PostACP(ctx, run.Status.PodName, bridge.serverID, string(run.Spec.AgentSession.Agent), initEnv)
	if err != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, err
	}
	var initResp struct {
		Result struct {
			AuthMethods []struct {
				ID string `json:"id"`
			} `json:"authMethods"`
		} `json:"result"`
		Error *jsonRPCError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &initResp); err != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, err
	}
	if initResp.Error != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, fmt.Errorf("sandbox-agent initialize failed: %s", initResp.Error.Message)
	}
	for _, method := range initResp.Result.AuthMethods {
		switch method.ID {
		case "anthropic-api-key", "codex-api-key", "openai-api-key":
			_, _ = s.transport.PostACP(ctx, run.Status.PodName, bridge.serverID, "", jsonRPCEnvelope{
				JSONRPC: "2.0",
				ID:      100 + time.Now().UnixNano(),
				Method:  "authenticate",
				Params:  map[string]any{"methodId": method.ID},
			})
		}
	}

	cwd := strings.TrimSpace(run.Spec.WorkingDir)
	if cwd == "" {
		cwd = "/workspace/repo"
	}
	newSessionEnv := jsonRPCEnvelope{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "session/new",
		Params: map[string]any{
			"cwd":        cwd,
			"mcpServers": []any{},
		},
	}
	body, err = s.transport.PostACP(ctx, run.Status.PodName, bridge.serverID, "", newSessionEnv)
	if err != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, err
	}
	var newSessionResp struct {
		Result struct {
			SessionID string `json:"sessionId"`
		} `json:"result"`
		Error *jsonRPCError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &newSessionResp); err != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, err
	}
	if newSessionResp.Error != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		state = bridge.snapshot()
		s.store.SaveState(state)
		return state, fmt.Errorf("sandbox-agent session/new failed: %s", newSessionResp.Error.Message)
	}
	bridge.mu.Lock()
	bridge.sessionID = strings.TrimSpace(newSessionResp.Result.SessionID)
	bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseReady)
	bridge.mu.Unlock()
	state = bridge.snapshot()
	s.store.SaveState(state)
	s.ensureStreaming(ctx, bridge)
	return state, nil
}

func (s *AgentSessionService) ensureStreaming(ctx context.Context, bridge *agentSessionBridge) {
	bridge.mu.Lock()
	if bridge.streaming {
		bridge.mu.Unlock()
		return
	}
	bridge.streaming = true
	podName := bridge.podName
	serverID := bridge.serverID
	streamCtx, cancel := context.WithCancel(s.serviceCtx)
	bridge.streamCancel = cancel
	done := make(chan struct{})
	bridge.streamDone = done
	bridge.mu.Unlock()
	go func() {
		defer close(done)
		defer cancel()
		stream, err := s.transport.StreamACP(streamCtx, podName, serverID)
		if err != nil {
			bridge.mu.Lock()
			bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
			bridge.streaming = false
			bridge.mu.Unlock()
			s.store.SaveState(bridge.snapshot())
			return
		}
		bridge.mu.Lock()
		bridge.streamBody = stream
		bridge.mu.Unlock()
		defer func() { _ = stream.Close() }()
		s.consumeSSE(stream, bridge)
	}()
}

func (s *AgentSessionService) consumeSSE(stream io.Reader, bridge *agentSessionBridge) {
	scanner := bufio.NewScanner(stream)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(dataLines) != 0 {
				payload := strings.Join(dataLines, "\n")
				if json.Valid([]byte(payload)) {
					event := bridge.appendEvent(json.RawMessage(append([]byte(nil), payload...)))
					s.store.AppendEvent(bridge.runID, event)
				}
				dataLines = dataLines[:0]
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	bridge.mu.Lock()
	bridge.streaming = false
	switch bridge.phase {
	case operatorv1alpha1.AgentSessionPhaseStopping:
		// Stop owns the terminal transition after it confirms whether DELETE
		// succeeded or failed, so the stream closure alone must not mark the
		// session completed.
	case operatorv1alpha1.AgentSessionPhaseCompleted:
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseCompleted)
	default:
		if scanner.Err() != nil {
			bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		} else {
			bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseCompleted)
		}
	}
	state := normalizeAgentSessionState(agentSessionState{
		HarnessRunID: bridge.runID,
		PodName:      bridge.podName,
		ServerID:     bridge.serverID,
		Runtime:      bridge.runtime,
		Agent:        bridge.agent,
		SessionID:    bridge.sessionID,
		Phase:        bridge.phase,
		LastSequence: bridge.nextSeq,
	})
	bridge.mu.Unlock()
	s.store.SaveState(state)
}

func (s *AgentSessionService) Prompt(ctx context.Context, run *operatorv1alpha1.HarnessRun, text string) (json.RawMessage, agentSessionState, error) {
	state, err := s.EnsureSession(ctx, run)
	if err != nil {
		return nil, agentSessionState{}, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, agentSessionState{}, fmt.Errorf("prompt required")
	}
	if state.Phase.IsTerminal() {
		return nil, state, &agentSessionOperationError{
			statusCode: http.StatusConflict,
			message:    fmt.Sprintf("agent session is %s", strings.ToLower(string(state.Phase))),
		}
	}
	bridge := s.bridgeFor(run)
	bridge.mu.Lock()
	bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseRunning)
	bridge.mu.Unlock()
	s.store.SaveState(bridge.snapshot())
	body, err := s.transport.PostACP(ctx, run.Status.PodName, bridge.serverID, "", jsonRPCEnvelope{
		JSONRPC: "2.0",
		ID:      bridge.promptSeq.Add(1),
		Method:  "session/prompt",
		Params: map[string]any{
			"sessionId": state.SessionID,
			"prompt":    []map[string]any{{"type": "text", "text": text}},
		},
	})
	if err != nil {
		bridge.mu.Lock()
		bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
		bridge.mu.Unlock()
		s.store.SaveState(bridge.snapshot())
		return nil, bridge.snapshot(), err
	}
	state = bridge.snapshot()
	s.store.SaveState(state)
	return json.RawMessage(append([]byte(nil), body...)), state, nil
}

func (s *AgentSessionService) ListEvents(offset int64, limit int, runIDs ...string) ([]agentSessionEvent, int64, bool) {
	return s.store.ListEvents(offset, limit, runIDs...)
}

func (s *AgentSessionService) GetState(run *operatorv1alpha1.HarnessRun) agentSessionState {
	if run.Spec.AgentSession == nil || !run.Spec.AgentSession.Enabled() {
		return agentSessionState{}
	}
	s.mu.Lock()
	bridge, ok := s.bridges[run.Name]
	s.mu.Unlock()
	if ok {
		return bridge.snapshot()
	}
	state, statusPresent := agentSessionStateFromHarnessRun(run)
	if persisted, ok := s.store.LoadState(run.Name, run.Labels["kocao.withakay.github.com/resumed-from"]); ok {
		state = mergePersistedAgentSessionState(state, persisted, statusPresent)
		state.HarnessRunID = run.Name
		state.PodName = run.Status.PodName
		state.ServerID = run.Name
		return normalizeAgentSessionState(state)
	}
	if state.Phase == "" {
		state.Phase = operatorv1alpha1.AgentSessionPhaseProvisioning
	}
	return normalizeAgentSessionState(state)
}

// isStalePodProxyError returns true when the error looks like a stale K8s pod
// proxy / HTTP/2 timeout that commonly occurs after closing a long-lived SSE
// stream. These are soft failures — the stream is already closed and the
// operator will clean up the pod.
//
// The matcher is intentionally narrow: only errors that are clearly caused by
// the TCP connection being torn down after the SSE stream close are tolerated.
// Generic "stream error" or bare "http2:" prefixes are NOT matched because
// they can indicate real transport failures (e.g. REFUSED_STREAM, GOAWAY with
// an error code, or protocol violations) that should surface to the caller.
func isStalePodProxyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "use of closed network connection")
}

func (s *AgentSessionService) Stop(ctx context.Context, run *operatorv1alpha1.HarnessRun) (agentSessionState, error) {
	s.mu.Lock()
	bridge, ok := s.bridges[run.Name]
	s.mu.Unlock()
	if !ok {
		state := s.GetState(run)
		if state.Phase.IsTerminal() {
			return state, nil
		}
		bridge = s.bridgeFor(run)
	}
	bridge.mu.Lock()
	if operatorv1alpha1.NormalizeAgentSessionPhase(string(bridge.phase)).IsTerminal() {
		bridge.mu.Unlock()
		return bridge.snapshot(), nil
	}
	bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseStopping)
	cancelFn := bridge.streamCancel
	doneCh := bridge.streamDone
	bridge.mu.Unlock()
	s.store.SaveState(bridge.snapshot())

	// Close the SSE stream body and cancel the context, then wait for the
	// goroutine to finish. The sandbox-agent serializes requests per server
	// ID, so DELETE blocks while the SSE GET connection is still open. We
	// must fully close the TCP connection before sending DELETE.
	bridge.mu.Lock()
	streamBody := bridge.streamBody
	bridge.mu.Unlock()
	if streamBody != nil {
		_ = streamBody.Close()
	}
	if cancelFn != nil {
		cancelFn()
	}
	if doneCh != nil {
		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
		}
	}

	// Send DELETE to the sandbox-agent. If the K8s pod proxy connection is
	// stale (common with HTTP/2 after closing a long-lived SSE stream), the
	// DELETE may time out. We tolerate that specific class of failure since
	// the stream is already closed and the operator will clean up the pod.
	// Other failures are returned to the caller.
	deleteCtx, deleteCancel := context.WithTimeout(ctx, 10*time.Second)
	defer deleteCancel()
	deleteErr := s.transport.DeleteACP(deleteCtx, run.Status.PodName, bridge.serverID)

	if deleteErr != nil {
		if isStalePodProxyError(deleteErr) {
			slog.Warn("agent session stop: DELETE failed with stale proxy error (session marked completed)", "run", run.Name, "error", deleteErr)
		} else {
			slog.Error("agent session stop: DELETE failed with unexpected error", "run", run.Name, "error", deleteErr)
			bridge.mu.Lock()
			bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseFailed)
			bridge.mu.Unlock()
			state := bridge.snapshot()
			s.store.SaveState(state)
			return state, fmt.Errorf("agent session stop: delete failed: %w", deleteErr)
		}
	}
	bridge.mu.Lock()
	bridge.transitionLocked(operatorv1alpha1.AgentSessionPhaseCompleted)
	bridge.mu.Unlock()
	state := bridge.snapshot()
	s.store.SaveState(state)
	return state, nil
}

func (a *API) getHarnessRun(ctx context.Context, id string) (*operatorv1alpha1.HarnessRun, error) {
	var run operatorv1alpha1.HarnessRun
	if err := a.K8s.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: id}, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (a *API) updateHarnessRunAgentSessionStatus(ctx context.Context, run *operatorv1alpha1.HarnessRun, state agentSessionState) {
	updated := run.DeepCopy()
	updated.Status.AgentSession = &operatorv1alpha1.AgentSessionStatus{
		Runtime:   state.Runtime,
		Agent:     state.Agent,
		SessionID: state.SessionID,
		Phase:     state.Phase,
	}
	if err := a.K8s.Status().Patch(ctx, updated, client.MergeFrom(run)); err != nil {
		slog.Error("failed to update agent session status", "run", run.Name, "error", err)
		return
	}
	run.Status.AgentSession = updated.Status.AgentSession
}

// agentSessionDTO is the response shape returned by the status and stop
// endpoints. It includes the fields the CLI/docs promise: workspaceSessionId,
// createdAt, and displayName, in addition to the core session state.
//
// IMPORTANT: The primary identifier field is "runId" (not "harnessRunID") to
// match the CLI AgentSession contract. The CLI must be able to unmarshal this
// payload without client-side backfill.
type agentSessionDTO struct {
	RunID              string                             `json:"runId"`
	PodName            string                             `json:"podName,omitempty"`
	ServerID           string                             `json:"serverID,omitempty"`
	Runtime            operatorv1alpha1.AgentRuntime      `json:"runtime,omitempty"`
	Agent              operatorv1alpha1.AgentKind         `json:"agent,omitempty"`
	SessionID          string                             `json:"sessionId,omitempty"`
	Phase              operatorv1alpha1.AgentSessionPhase `json:"phase,omitempty"`
	LastSequence       int64                              `json:"lastSequence,omitempty"`
	WorkspaceSessionID string                             `json:"workspaceSessionId,omitempty"`
	DisplayName        string                             `json:"displayName,omitempty"`
	CreatedAt          string                             `json:"createdAt,omitempty"`
	Diagnostic         *agentSessionDiagnosticDTO         `json:"diagnostic,omitempty"`
}

type agentSessionDiagnosticDTO struct {
	Class   string `json:"class"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
}

func (a *API) agentSessionToDTO(ctx context.Context, run *operatorv1alpha1.HarnessRun, state agentSessionState) agentSessionDTO {
	dto := agentSessionDTO{
		RunID:              run.Name,
		PodName:            state.PodName,
		ServerID:           state.ServerID,
		Runtime:            state.Runtime,
		Agent:              state.Agent,
		SessionID:          state.SessionID,
		Phase:              state.Phase,
		LastSequence:       state.LastSequence,
		WorkspaceSessionID: run.Spec.WorkspaceSessionName,
	}
	if !run.CreationTimestamp.IsZero() {
		dto.CreatedAt = run.CreationTimestamp.Time.UTC().Format(time.RFC3339)
	}
	dto.Diagnostic = a.agentSessionDiagnostic(ctx, run, state)
	if run.Spec.WorkspaceSessionName != "" {
		displayName := a.sessionDisplayNameFor(ctx, run)
		if displayName != "" {
			suffix := run.Name
			if len(suffix) > 5 {
				suffix = suffix[len(suffix)-5:]
			}
			dto.DisplayName = displayName + "-" + suffix
		}
	}
	return dto
}

func (a *API) agentSessionDiagnostic(ctx context.Context, run *operatorv1alpha1.HarnessRun, state agentSessionState) *agentSessionDiagnosticDTO {
	phase := operatorv1alpha1.NormalizeAgentSessionPhase(string(state.Phase))
	switch phase {
	case operatorv1alpha1.AgentSessionPhaseReady, operatorv1alpha1.AgentSessionPhaseRunning, operatorv1alpha1.AgentSessionPhaseCompleted:
		return nil
	}

	if strings.TrimSpace(state.PodName) == "" {
		return &agentSessionDiagnosticDTO{
			Class:   agentSessionBlockerProvisioning,
			Summary: "Harness pod has not been assigned yet.",
			Detail:  "Waiting for the operator to create and schedule the run pod.",
		}
	}

	var pod corev1.Pod
	if err := a.K8s.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: state.PodName}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			return &agentSessionDiagnosticDTO{
				Class:   agentSessionBlockerProvisioning,
				Summary: "Harness pod is still provisioning.",
				Detail:  fmt.Sprintf("Pod %q is not visible yet.", state.PodName),
			}
		}
		return &agentSessionDiagnosticDTO{
			Class:   agentSessionBlockerProvisioning,
			Summary: "Unable to inspect harness pod state.",
			Detail:  err.Error(),
		}
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
			return &agentSessionDiagnosticDTO{
				Class:   agentSessionBlockerProvisioning,
				Summary: "Pod scheduling is blocking session readiness.",
				Detail:  diagnosticDetail(cond.Reason, cond.Message),
			}
		}
	}

	for _, status := range pod.Status.InitContainerStatuses {
		if diag := diagnosticFromContainerStatus(status, true); diag != nil {
			return diag
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if diag := diagnosticFromContainerStatus(status, false); diag != nil {
			return diag
		}
	}

	if pod.Status.Phase == corev1.PodRunning && phase == operatorv1alpha1.AgentSessionPhaseProvisioning {
		return &agentSessionDiagnosticDTO{
			Class:   agentSessionBlockerSandboxAgentReadiness,
			Summary: "Sandbox-agent is not ready yet.",
			Detail:  fmt.Sprintf("Pod %q is running, but the sandbox-agent health path has not produced a ready session.", pod.Name),
		}
	}

	if pod.Status.Phase == corev1.PodPending {
		return &agentSessionDiagnosticDTO{
			Class:   agentSessionBlockerProvisioning,
			Summary: "Harness pod is still pending.",
			Detail:  fmt.Sprintf("Pod %q is pending while the session remains in %s.", pod.Name, valueOrUnknown(string(phase))),
		}
	}

	return nil
}

func diagnosticFromContainerStatus(status corev1.ContainerStatus, initContainer bool) *agentSessionDiagnosticDTO {
	containerType := "container"
	if initContainer {
		containerType = "initContainer"
	}
	name := strings.TrimSpace(status.Name)
	if waiting := status.State.Waiting; waiting != nil {
		if class, summary, ok := classifyAgentSessionBlocker(name, waiting.Reason, waiting.Message); ok {
			return &agentSessionDiagnosticDTO{Class: class, Summary: summary, Detail: fmt.Sprintf("%s %q waiting: %s", containerType, name, diagnosticDetail(waiting.Reason, waiting.Message))}
		}
	}
	if terminated := status.State.Terminated; terminated != nil && terminated.ExitCode != 0 {
		if class, summary, ok := classifyAgentSessionBlocker(name, terminated.Reason, terminated.Message); ok {
			return &agentSessionDiagnosticDTO{Class: class, Summary: summary, Detail: fmt.Sprintf("%s %q terminated: %s", containerType, name, diagnosticDetail(terminated.Reason, terminated.Message))}
		}
	}
	return nil
}

func classifyAgentSessionBlocker(containerName, reason, message string) (string, string, bool) {
	combined := strings.ToLower(strings.Join([]string{containerName, reason, message}, " "))
	switch {
	case containsAny(combined, "imagepullbackoff", "errimagepull", "failed to pull image", "pull access denied", "back-off pulling image", "invalidimagename"):
		return agentSessionBlockerImagePull, "Image pull is blocking session readiness.", true
	case containsAny(combined, "dial tcp", "i/o timeout", "no such host", "network is unreachable", "connection refused", "tls handshake timeout", "temporary failure in name resolution", "egress"):
		return agentSessionBlockerNetwork, "Network reachability is blocking session readiness.", true
	case containsAny(combined, "repository", "repo", "git clone", "could not read from remote repository", "access denied", "remote: repository not found", "git ls-remote"):
		return agentSessionBlockerRepoAccess, "Repository access is blocking session readiness.", true
	case containsAny(combined, "oauth", "auth.json", "credential", "api key", "secret", "token", "unauthorized", "forbidden", "authentication"):
		return agentSessionBlockerAuth, "Credential setup is blocking session readiness.", true
	case containsAny(combined, "sandbox-agent"):
		return agentSessionBlockerSandboxAgentReadiness, "Sandbox-agent is not ready yet.", true
	default:
		return "", "", false
	}
}

func diagnosticDetail(reason, message string) string {
	reason = strings.TrimSpace(reason)
	message = strings.TrimSpace(message)
	if reason == "" {
		return message
	}
	if message == "" {
		return reason
	}
	return reason + ": " + message
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown state"
	}
	return value
}

type agentSessionPromptRequest struct {
	Prompt string `json:"prompt"`
}

func (a *API) handleRunAgentSessionGet(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	state := a.AgentSessions.GetState(run)
	if state.Runtime == "" {
		writeError(w, http.StatusNotFound, "agent session not configured for harness run")
		return
	}
	writeJSON(w, http.StatusOK, a.agentSessionToDTO(r.Context(), run, state))
}

func (a *API) handleRunAgentSessionCreate(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	state, err := a.AgentSessions.EnsureSession(r.Context(), run)
	if err != nil {
		slog.Error("agent session create failed", "run", id, "error", err)
		a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
		writeError(w, http.StatusBadGateway, "agent session create failed")
		return
	}
	a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
	a.Audit.Append(r.Context(), principal(r.Context()), "agent-session.create", "harness-run", id, "allowed", map[string]any{"agent": state.Agent, "runtime": state.Runtime, "sessionId": state.SessionID})
	writeJSON(w, http.StatusCreated, a.agentSessionToDTO(r.Context(), run, state))
}

func (a *API) handleRunAgentSessionPrompt(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	var req agentSessionPromptRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	result, state, err := a.AgentSessions.Prompt(r.Context(), run, req.Prompt)
	if err != nil {
		slog.Error("agent session prompt failed", "run", id, "error", err)
		a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
		var opErr *agentSessionOperationError
		if errors.As(err, &opErr) {
			writeError(w, opErr.statusCode, opErr.message)
			return
		}
		writeError(w, http.StatusBadGateway, "agent session prompt failed")
		return
	}
	a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
	a.Audit.Append(r.Context(), principal(r.Context()), "agent-session.prompt", "harness-run", id, "allowed", map[string]any{"sessionId": state.SessionID})

	// Build an event from the prompt result so the CLI receives a uniform
	// {events: [...]} envelope it can display immediately.
	promptEvent := agentSessionEvent{
		Sequence: state.LastSequence + 1,
		At:       time.Now().UTC(),
		Envelope: json.RawMessage(result),
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session": state,
		"result":  json.RawMessage(result),
		"events":  []agentSessionEvent{promptEvent},
	})
}

func (a *API) handleRunAgentSessionEvents(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	offset, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("offset")), 10, 64)
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	events, next, ok := a.AgentSessions.ListEvents(offset, limit, id, run.Labels["kocao.withakay.github.com/resumed-from"])
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"events": []agentSessionEvent{}, "nextOffset": 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "nextOffset": next})
}

func (a *API) handleRunAgentSessionEventsStream(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	state := a.AgentSessions.GetState(run)
	if state.Runtime == "" {
		writeError(w, http.StatusNotFound, "agent session not configured for harness run")
		return
	}
	bridge := a.AgentSessions.bridgeFor(run)
	offset, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("offset")), 10, 64)
	backlog, _, _ := a.AgentSessions.ListEvents(offset, 500, id, run.Labels["kocao.withakay.github.com/resumed-from"])
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	for _, event := range backlog {
		b, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	}
	flusher.Flush()
	ch, unsubscribe := bridge.subscribe()
	defer unsubscribe()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			state := bridge.snapshot()
			if state.Phase == operatorv1alpha1.AgentSessionPhaseCompleted || state.Phase == operatorv1alpha1.AgentSessionPhaseFailed {
				return
			}
		case event := <-ch:
			if event.Sequence <= offset {
				continue
			}
			b, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (a *API) handleRunAgentSessionStop(w http.ResponseWriter, r *http.Request, id string) {
	if a.AgentSessions == nil {
		writeError(w, http.StatusNotImplemented, "agent session service not configured")
		return
	}
	run, err := a.getHarnessRun(r.Context(), id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "harness run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get harness run failed")
		return
	}
	state, err := a.AgentSessions.Stop(r.Context(), run)
	if err != nil {
		// Best-effort: persist the failed state to the HarnessRun CRD so
		// the operator and subsequent status queries reflect the failure,
		// even though we are about to return a 502 to the caller.
		a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	a.updateHarnessRunAgentSessionStatus(r.Context(), run, state)
	a.Audit.Append(r.Context(), principal(r.Context()), "agent-session.stop", "harness-run", id, "allowed", map[string]any{"sessionId": state.SessionID})
	writeJSON(w, http.StatusOK, a.agentSessionToDTO(r.Context(), run, state))
}
