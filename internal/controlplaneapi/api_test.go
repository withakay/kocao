package controlplaneapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestAPI(t *testing.T) (*API, func()) {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}, &operatorv1alpha1.Session{}).Build()

	api, err := New("test-ns", "", "", nil, k8s, Options{Env: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	cleanup := func() {}
	return api, cleanup
}

func newTestAPIWithAttachOptions(t *testing.T, opts Options) (*API, func()) {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}, &operatorv1alpha1.Session{}).Build()
	restCfg := &rest.Config{Host: "https://example.invalid"}

	api, err := New("test-ns", "", "", restCfg, k8s, opts)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return api, func() {}
}

func doJSON(t *testing.T, c *http.Client, method, url, token string, body any) (*http.Response, []byte) {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

type fakeAgentSessionTransport struct {
	mu         sync.Mutex
	writer     *io.PipeWriter
	sessionID  string
	deleted    bool
	postCalls  []string
	lastPrompt string
}

func newFakeAgentSessionTransport() *fakeAgentSessionTransport {
	return &fakeAgentSessionTransport{sessionID: "sas-123"}
}

func (f *fakeAgentSessionTransport) PostACP(_ context.Context, _ string, _ string, _ string, payload any) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	env, ok := payload.(jsonRPCEnvelope)
	if !ok {
		return nil, fmt.Errorf("unexpected payload type %T", payload)
	}
	f.postCalls = append(f.postCalls, env.Method)
	switch env.Method {
	case "initialize":
		return []byte(`{"jsonrpc":"2.0","id":1,"result":{"authMethods":[]}}`), nil
	case "session/new":
		return []byte(`{"jsonrpc":"2.0","id":2,"result":{"sessionId":"sas-123"}}`), nil
	case "session/prompt":
		params, _ := env.Params.(map[string]any)
		prompt, _ := params["prompt"].([]map[string]any)
		if len(prompt) != 0 {
			if text, ok := prompt[0]["text"].(string); ok {
				f.lastPrompt = text
			}
		}
		if f.writer != nil {
			_, _ = fmt.Fprintf(f.writer, "data: {\"jsonrpc\":\"2.0\",\"method\":\"session/update\",\"params\":{\"sessionId\":\"%s\",\"sessionUpdate\":\"user_message_chunk\",\"content\":{\"type\":\"text\",\"text\":%q}}}\n\n", f.sessionID, f.lastPrompt)
			_, _ = fmt.Fprintf(f.writer, "data: {\"jsonrpc\":\"2.0\",\"method\":\"session/update\",\"params\":{\"sessionId\":\"%s\",\"sessionUpdate\":\"agent_message_chunk\",\"content\":{\"type\":\"text\",\"text\":\"ack\"}}}\n\n", f.sessionID)
		}
		return []byte(`{"jsonrpc":"2.0","id":3,"result":{"stopReason":"completed"}}`), nil
	default:
		return []byte(`{"jsonrpc":"2.0","id":99,"result":{}}`), nil
	}
}

func (f *fakeAgentSessionTransport) StreamACP(_ context.Context, _ string, _ string) (io.ReadCloser, error) {
	reader, writer := io.Pipe()
	f.mu.Lock()
	f.writer = writer
	f.mu.Unlock()
	return reader, nil
}

func (f *fakeAgentSessionTransport) waitWriter(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		f.mu.Lock()
		w := f.writer
		f.mu.Unlock()
		if w != nil {
			return
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for fake transport writer to be established")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (f *fakeAgentSessionTransport) DeleteACP(_ context.Context, _ string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = true
	if f.writer != nil {
		_ = f.writer.Close()
	}
	return nil
}

func TestOpenAPI_Unauthenticated(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi.json")
	if err != nil {
		t.Fatalf("GET openapi: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestCaddyEdgeConfig_HasScalarAndProxyRoutes(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "deploy", "base", "caddy", "Caddyfile"))
	if err != nil {
		t.Fatalf("read Caddyfile: %v", err)
	}
	content := string(b)
	required := []string{
		":8081",
		"/api/v1/scalar",
		"/api/v1/openapi.json",
		"/api/v1/*",
		"path_regexp attach ^/api/v1/workspace-sessions/[^/]+/attach$",
		"header Upgrade websocket",
		"reverse_proxy 127.0.0.1:8080",
	}
	for _, token := range required {
		if !strings.Contains(content, token) {
			t.Fatalf("Caddyfile missing %q", token)
		}
	}
}

func TestAuth_MissingToken_DeniedAndAudited(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, _ := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	evs, err := api.Audit.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(evs) == 0 {
		t.Fatalf("expected audit events")
	}
	if evs[0].Outcome != "denied" {
		t.Fatalf("audit outcome = %q, want denied", evs[0].Outcome)
	}
}

func TestRBAC_MissingScope_DeniedAndAudited(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-readonly", "readonly", []string{"workspace-session:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, _ := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "readonly", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}

	evs, err := api.Audit.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(evs) == 0 {
		t.Fatalf("expected audit events")
	}
	if evs[0].Outcome != "denied" {
		t.Fatalf("audit outcome = %q, want denied", evs[0].Outcome)
	}
}

func TestJSON_BodyTooLarge_Returns413(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-writer", "writer", []string{"workspace-session:write", "control:write"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	big := strings.Repeat("a", int(maxJSONBodyBytes)+256)
	var buf bytes.Buffer
	buf.Grow(len(big) + 32)
	buf.WriteString("{\"repoURL\":\"")
	buf.WriteString(big)
	buf.WriteString("\"}")

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/workspace-sessions", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer writer")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 413 (body=%s)", resp.StatusCode, string(b))
	}
}

func TestLifecycle_SessionRunControlsAndAudit(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read", "harness-run:write", "harness-run:read", "control:write", "audit:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)
	if sess.ID == "" {
		t.Fatalf("expected session id")
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-control", "full", map[string]any{"enabled": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("attach control status = %d, want 200", resp.StatusCode)
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/egress-override", "full", map[string]any{"mode": "restricted"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("egress override status = %d, want 200", resp.StatusCode)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL":      "https://example.com/repo",
		"repoRevision": "main",
		"image":        "alpine:3",
		"args":         []string{"bash", "-lc", "make ci"},
		"env":          []map[string]any{{"name": "GITHUB_TOKEN", "value": "redacted"}},
		"agentSession": map[string]any{"agent": "codex"},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("start run status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var run runResponse
	_ = json.Unmarshal(b, &run)
	if run.ID == "" || run.WorkspaceSessionID != sess.ID {
		t.Fatalf("unexpected run response: %+v", run)
	}
	if run.RepoRevision != "main" {
		t.Fatalf("repoRevision = %q, want main", run.RepoRevision)
	}
	if run.AgentSession == nil {
		t.Fatalf("expected agentSession in run response")
	}
	if run.AgentSession.Runtime != operatorv1alpha1.AgentRuntimeSandboxAgent {
		t.Fatalf("agent runtime = %q, want %q", run.AgentSession.Runtime, operatorv1alpha1.AgentRuntimeSandboxAgent)
	}
	if run.AgentSession.Agent != operatorv1alpha1.AgentKindCodex {
		t.Fatalf("agent = %q, want %q", run.AgentSession.Agent, operatorv1alpha1.AgentKindCodex)
	}
	if run.AgentSession.Phase != operatorv1alpha1.AgentSessionPhaseProvisioning {
		t.Fatalf("agent phase = %q, want %q", run.AgentSession.Phase, operatorv1alpha1.AgentSessionPhaseProvisioning)
	}

	// Simulate harness/outcome reporter adding GitHub metadata.
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: run.ID}, &stored); err != nil {
		t.Fatalf("get stored run: %v", err)
	}
	if stored.Annotations == nil {
		stored.Annotations = map[string]string{}
	}
	if len(stored.Spec.Args) != 3 || stored.Spec.Args[0] != "bash" || stored.Spec.Args[1] != "-lc" || stored.Spec.Args[2] != "make ci" {
		t.Fatalf("run args not persisted: %#v", stored.Spec.Args)
	}
	if stored.Spec.AgentSession == nil || stored.Spec.AgentSession.Agent != operatorv1alpha1.AgentKindCodex {
		t.Fatalf("agent session spec not persisted: %#v", stored.Spec.AgentSession)
	}
	if stored.Status.AgentSession == nil || stored.Status.AgentSession.Phase != operatorv1alpha1.AgentSessionPhaseProvisioning {
		t.Fatalf("agent session status not initialized: %#v", stored.Status.AgentSession)
	}
	stored.Annotations["kocao.withakay.github.com/github-branch"] = "feature/mvp-ui"
	stored.Annotations["kocao.withakay.github.com/pull-request-url"] = "https://github.com/withakay/kocao/pull/123"
	stored.Annotations["kocao.withakay.github.com/pull-request-status"] = "merged"
	if err := api.K8s.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update stored run: %v", err)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+run.ID, "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get run status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	var got runResponse
	_ = json.Unmarshal(b, &got)
	if got.GitHubBranch != "feature/mvp-ui" {
		t.Fatalf("gitHubBranch = %q, want feature/mvp-ui", got.GitHubBranch)
	}
	if got.PullRequestURL != "https://github.com/withakay/kocao/pull/123" {
		t.Fatalf("pullRequestURL = %q, want PR url", got.PullRequestURL)
	}
	if got.PullRequestStatus != "merged" {
		t.Fatalf("pullRequestStatus = %q, want merged", got.PullRequestStatus)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.ID+"/resume", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("resume run status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var resumed runResponse
	_ = json.Unmarshal(b, &resumed)
	if resumed.ID == "" || resumed.ID == run.ID {
		t.Fatalf("expected new run id")
	}
	if resumed.AgentSession == nil || resumed.AgentSession.Agent != operatorv1alpha1.AgentKindCodex {
		t.Fatalf("expected resumed run to retain agent session, got %+v", resumed.AgentSession)
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.ID+"/stop", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop run status = %d, want 200", resp.StatusCode)
	}
	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+resumed.ID+"/stop", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop resumed run status = %d, want 200", resp.StatusCode)
	}

	evs, err := api.Audit.List(context.Background(), 200)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	var sawCred bool
	for _, e := range evs {
		if e.Action == "credential.use" {
			sawCred = true
			break
		}
	}
	if !sawCred {
		t.Fatalf("expected credential.use audit event")
	}
}

func TestCreateHarnessRunRejectsUnsupportedAgentSession(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read", "harness-run:write", "harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL":      "https://example.com/repo",
		"repoRevision": "main",
		"image":        "alpine:3",
		"agentSession": map[string]any{"runtime": "sandbox-agent", "agent": "cursor"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), "agentSession.agent") {
		t.Fatalf("expected validation error mentioning agentSession.agent, got %s", string(b))
	}
}

func TestAgentSessionLifecycle_API(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()
	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, newAgentSessionStore(""))

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"harness-run:write", "harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-agent-session", Namespace: api.Namespace},
		Spec: operatorv1alpha1.HarnessRunSpec{
			WorkspaceSessionName: "sess-1",
			RepoURL:              "https://example.com/repo",
			Image:                "kocao/harness-runtime:dev",
			WorkingDir:           "/workspace/repo",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindCodex,
			},
		},
	}
	if err := api.K8s.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(run), &stored); err != nil {
		t.Fatalf("get run: %v", err)
	}
	stored.Status.PodName = "pod-agent-session"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var created agentSessionState
	if err := json.Unmarshal(b, &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	if created.SessionID != "sas-123" || created.Phase != operatorv1alpha1.AgentSessionPhaseReady {
		t.Fatalf("unexpected agent session create response: %+v", created)
	}
	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get agent session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}

	streamReq, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/events/stream?offset=0", nil)
	if err != nil {
		t.Fatalf("new stream request: %v", err)
	}
	streamReq.Header.Set("Authorization", "Bearer full")
	streamResp, err := srv.Client().Do(streamReq)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer func() { _ = streamResp.Body.Close() }()
	streamCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(streamResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				streamCh <- line
				return
			}
		}
		streamCh <- ""
	}()

	// Wait for the streaming goroutine to connect and establish the pipe
	// writer before sending the prompt. Without this, on slow CI runners the
	// prompt can fire before the writer exists, causing events to be lost.
	transport.waitWriter(t, 5*time.Second)

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/prompt", "full", map[string]any{"prompt": "hello sandbox"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prompt status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	select {
	case line := <-streamCh:
		if !strings.Contains(line, "user_message_chunk") {
			t.Fatalf("expected streamed event, got %q", line)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for stream event")
	}

	var eventsPayload struct {
		Events []agentSessionEvent `json:"events"`
	}
	for i := 0; i < 20; i++ {
		resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/events?offset=0&limit=10", "full", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("events status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
		}
		_ = json.Unmarshal(b, &eventsPayload)
		if len(eventsPayload.Events) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(eventsPayload.Events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(eventsPayload.Events))
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/stop", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop agent session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	if !transport.deleted {
		t.Fatal("expected fake transport delete to be called")
	}
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("restart agent session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}

	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(run), &stored); err != nil {
		t.Fatalf("get updated run: %v", err)
	}
	if stored.Status.AgentSession == nil || stored.Status.AgentSession.SessionID != "sas-123" {
		t.Fatalf("expected stored agent session status to be updated, got %#v", stored.Status.AgentSession)
	}
}

func TestSymphonyProjectLifecycle_API(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-symphony", "symphony", []string{ScopeSymphonyProjectRead, ScopeSymphonyProjectWrite, ScopeSymphonyProjectControl}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	body := map[string]any{
		"name": "demo",
		"spec": map[string]any{
			"source": map[string]any{
				"project":        map[string]any{"owner": "withakay", "number": 42},
				"githubToken":    "github_pat_demo_token_value",
				"activeStates":   []string{"Queued"},
				"terminalStates": []string{"Done"},
			},
			"repositories": []map[string]any{{"owner": "withakay", "name": "kocao", "repoURL": "https://github.com/withakay/kocao"}},
			"runtime":      map[string]any{"image": "ghcr.io/withakay/kocao-harness:latest"},
		},
	}

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/symphony-projects", "symphony", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create symphony project status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var created symphonyProjectResponse
	_ = json.Unmarshal(b, &created)
	if created.Name != "demo" {
		t.Fatalf("name = %q, want demo", created.Name)
	}
	if created.Spec.Runtime.MaxConcurrentItems != operatorv1alpha1.DefaultSymphonyMaxConcurrentItems {
		t.Fatalf("maxConcurrentItems = %d", created.Spec.Runtime.MaxConcurrentItems)
	}
	if created.Spec.Source.TokenSecretRef.Name != "symphony-demo-withakay-token" {
		t.Fatalf("token secret ref = %#v", created.Spec.Source.TokenSecretRef)
	}
	var secret corev1.Secret
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: "symphony-demo-withakay-token"}, &secret); err != nil {
		t.Fatalf("get symphony token secret: %v", err)
	}
	if string(secret.Data["token"]) != "github_pat_demo_token_value" {
		t.Fatalf("secret token = %q", string(secret.Data["token"]))
	}

	var stored operatorv1alpha1.SymphonyProject
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: "demo"}, &stored); err != nil {
		t.Fatalf("get symphony project: %v", err)
	}
	stored.Status.Phase = operatorv1alpha1.SymphonyProjectPhaseReady
	stored.Status.RunningItems = 1
	stored.Status.RetryingItems = 2
	stored.Status.TokenTotals = operatorv1alpha1.SymphonyProjectTokenTotalsStatus{InputTokens: 120, OutputTokens: 45, TotalTokens: 165, SecondsRunning: 9.5}
	stored.Status.RecentEvents = []operatorv1alpha1.SymphonyProjectEventStatus{{
		ItemID:         "PVT_item_1",
		Issue:          operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 101, Title: "First issue"},
		SessionID:      "thread-1-turn-1",
		ThreadID:       "thread-1",
		TurnID:         "turn-1",
		Event:          "turn_completed",
		Message:        "workflow execution completed",
		HarnessRunName: "sym-run-1",
	}}
	nextSync := metav1.Now()
	stored.Status.NextSyncTime = &nextSync
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update symphony status: %v", err)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/symphony-projects", "symphony", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list symphony projects status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), "symphonyProjects") {
		t.Fatalf("list response missing symphonyProjects: %s", string(b))
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/symphony-projects/demo", "symphony", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get symphony project status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	var fetched symphonyProjectResponse
	_ = json.Unmarshal(b, &fetched)
	if fetched.Status.RunningItems != 1 || fetched.Status.RetryingItems != 2 {
		t.Fatalf("unexpected runtime summary: %#v", fetched.Status)
	}
	if fetched.Status.TokenTotals.TotalTokens != 165 {
		t.Fatalf("unexpected token totals: %#v", fetched.Status.TokenTotals)
	}
	if len(fetched.Status.RecentEvents) != 1 || fetched.Status.RecentEvents[0].Event != "turn_completed" {
		t.Fatalf("unexpected recent events: %#v", fetched.Status.RecentEvents)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/symphony-projects/demo", "symphony", map[string]any{
		"spec": map[string]any{
			"paused": true,
			"source": map[string]any{
				"project":        map[string]any{"owner": "withakay", "number": 42},
				"githubToken":    "github_pat_rotated_token_value",
				"activeStates":   []string{"Queued"},
				"terminalStates": []string{"Done"},
			},
			"repositories": []map[string]any{{"owner": "withakay", "name": "kocao", "repoURL": "https://github.com/withakay/kocao", "branch": "main"}},
			"runtime":      map[string]any{"image": "ghcr.io/withakay/kocao-harness:latest"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch symphony project status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: "symphony-demo-withakay-token"}, &secret); err != nil {
		t.Fatalf("get updated symphony token secret: %v", err)
	}
	if string(secret.Data["token"]) != "github_pat_rotated_token_value" {
		t.Fatalf("updated secret token = %q", string(secret.Data["token"]))
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/symphony-projects/demo", "symphony", map[string]any{
		"spec": map[string]any{
			"paused": false,
			"source": map[string]any{
				"project":        map[string]any{"owner": "withakay", "number": 42},
				"tokenSecretRef": map[string]any{"name": "symphony-demo-withakay-token"},
				"activeStates":   []string{"Queued"},
				"terminalStates": []string{"Done"},
			},
			"repositories": []map[string]any{{"owner": "withakay", "name": "kocao", "repoURL": "https://github.com/withakay/kocao", "branch": "main"}},
			"runtime":      map[string]any{"image": "ghcr.io/withakay/kocao-harness:latest"},
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch without github token status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: "symphony-demo-withakay-token"}, &secret); err != nil {
		t.Fatalf("get preserved symphony token secret: %v", err)
	}
	if string(secret.Data["token"]) != "github_pat_rotated_token_value" {
		t.Fatalf("expected existing token to remain, got %q", string(secret.Data["token"]))
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/symphony-projects/demo/pause", "symphony", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pause symphony project status = %d, want 200", resp.StatusCode)
	}
	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/symphony-projects/demo/resume", "symphony", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resume symphony project status = %d, want 200", resp.StatusCode)
	}
	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/symphony-projects/demo/refresh", "symphony", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh symphony project status = %d, want 200", resp.StatusCode)
	}

	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: "demo"}, &stored); err != nil {
		t.Fatalf("get refreshed symphony project: %v", err)
	}
	if stored.Spec.Paused {
		t.Fatalf("expected project to be resumed")
	}
	if stored.Annotations[annotationSymphonyRefreshRequestedAt] == "" {
		t.Fatalf("expected refresh annotation, got %#v", stored.Annotations)
	}
}

func TestSymphonyProjectCreate_RejectsPATInSecretName(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-symphony", "symphony", []string{ScopeSymphonyProjectWrite}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	body := map[string]any{
		"name": "demo",
		"spec": map[string]any{
			"source": map[string]any{
				"project":        map[string]any{"owner": "withakay", "number": 42},
				"tokenSecretRef": map[string]any{"name": "github_pat_should_not_be_here"},
				"activeStates":   []string{"Queued"},
				"terminalStates": []string{"Done"},
			},
			"repositories": []map[string]any{{"owner": "withakay", "name": "kocao", "repoURL": "https://github.com/withakay/kocao"}},
			"runtime":      map[string]any{"image": "ghcr.io/withakay/kocao-harness:latest"},
		},
	}

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/symphony-projects", "symphony", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", resp.StatusCode, string(b))
	}
	if strings.Contains(string(b), "github_pat_should_not_be_here") {
		t.Fatalf("response leaked raw token-like value: %s", string(b))
	}
}

func TestEgressOverride_RejectsAllowedHosts(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-writer", "writer", []string{"workspace-session:write", "control:write"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "writer", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)
	if sess.ID == "" {
		t.Fatalf("expected session id")
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/egress-override", "writer", map[string]any{"mode": "restricted", "allowedHosts": []string{"github.com"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("egress override status = %d, want 400 (body=%s)", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), "allowedHosts is not supported") {
		t.Fatalf("expected error to mention allowedHosts not supported (body=%s)", string(b))
	}
}

func TestAttachWS_OriginAllowlist(t *testing.T) {
	api, cleanup := newTestAPIWithAttachOptions(t, Options{Env: "prod", AttachWSAllowedOrigins: []string{"https://allowed.example"}})
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read", "harness-run:read", "control:write"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)

	resp, _ = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-control", "full", map[string]any{"enabled": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("attach-control status = %d, want 200", resp.StatusCode)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "viewer"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var tok attachTokenResponse
	_ = json.Unmarshal(b, &tok)

	ws := wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{tok.Token}})

	_, respWS, err := websocket.DefaultDialer.Dial(ws, http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		_ = respWS.Body.Close()
		t.Fatalf("expected dial error")
	}
	if respWS == nil || respWS.StatusCode != http.StatusForbidden {
		code := 0
		if respWS != nil {
			code = respWS.StatusCode
			_ = respWS.Body.Close()
		}
		t.Fatalf("status=%d, want 403", code)
	}
	_ = respWS.Body.Close()

	conn, _, err := websocket.DefaultDialer.Dial(ws, http.Header{"Origin": []string{"https://allowed.example"}})
	if err != nil {
		t.Fatalf("dial allowed origin: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readMsgType(t, conn, "hello")
}

func TestSessionCreate_AutoGeneratesDisplayName(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)
	if sess.DisplayName == "" {
		t.Fatal("expected auto-generated display name, got empty")
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/"+sess.ID, "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	var got sessionResponse
	_ = json.Unmarshal(b, &got)
	if got.DisplayName != sess.DisplayName {
		t.Fatalf("expected display name %q, got %q", sess.DisplayName, got.DisplayName)
	}
}

func TestSessionCreate_DuplicateDisplayNameConflict(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{
		"displayName": "bold-tiger",
		"repoURL":     "https://example.com/repo",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{
		"displayName": "bold-tiger",
		"repoURL":     "https://example.com/repo2",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%s)", resp.StatusCode, string(b))
	}
}

func TestSessionDelete_DeletesSession(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)
	if sess.ID == "" {
		t.Fatal("expected session ID")
	}

	resp, b = doJSON(t, srv.Client(), http.MethodDelete, srv.URL+"/api/v1/workspace-sessions/"+sess.ID, "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/"+sess.ID, "full", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get deleted session status = %d, want 404", resp.StatusCode)
	}
}

func TestClusterOverview_ReturnsNamespaceState(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&operatorv1alpha1.Session{ObjectMeta: metav1.ObjectMeta{Name: "sess-1", Namespace: "test-ns"}, Spec: operatorv1alpha1.SessionSpec{DisplayName: "calm-morse", RepoURL: "https://example.com/repo"}},
		&operatorv1alpha1.HarnessRun{ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "test-ns"}, Spec: operatorv1alpha1.HarnessRunSpec{WorkspaceSessionName: "sess-1", RepoURL: "https://example.com/repo", Image: "kocao/harness-runtime:dev"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "control-plane-api", Namespace: "test-ns"}, Spec: appsv1.DeploymentSpec{Replicas: int32Ptr(1)}, Status: appsv1.DeploymentStatus{ReadyReplicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "control-plane-api-abc", Namespace: "test-ns"}, Spec: corev1.PodSpec{NodeName: "node-1", Containers: []corev1.Container{{Name: "api"}}}, Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "api", Ready: true, RestartCount: 2}}}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "control-plane-config", Namespace: "test-ns"}, Data: map[string]string{"CP_ENV": "dev", "CP_AUDIT_PATH": "/tmp/audit", "CP_BOOTSTRAP_TOKEN": "present"}},
	).Build()

	api, err := New("test-ns", "", "", nil, k8s, Options{Env: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := api.Tokens.Create(context.Background(), "t-cluster", "cluster", []string{"cluster:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/cluster-overview", "cluster", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}

	var out clusterOverviewResponse
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Namespace != "test-ns" {
		t.Fatalf("namespace = %q, want test-ns", out.Namespace)
	}
	if out.Summary.SessionCount != 1 || out.Summary.HarnessRunCount != 1 {
		t.Fatalf("unexpected summary: %+v", out.Summary)
	}
	if len(out.Pods) != 1 || len(out.Deployments) != 1 {
		t.Fatalf("unexpected payload counts: pods=%d deployments=%d", len(out.Pods), len(out.Deployments))
	}
	if !out.Config.BootstrapTokenDetected {
		t.Fatal("expected bootstrap token indicator")
	}
}

func TestPodLogs_NoClientset_Returns503(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-cluster", "cluster", []string{"cluster:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/pods/example/logs", "cluster", nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body=%s)", resp.StatusCode, string(b))
	}
}

func TestWorkspaceAgentSessionsList_ReturnsSessionsFromHarnessRuns(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, newAgentSessionStore(""))

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{
		"workspace-session:write", "workspace-session:read",
		"harness-run:write", "harness-run:read",
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create a workspace session.
	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{
		"displayName": "test-ws",
		"repoURL":     "https://example.com/repo",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)

	// Create a harness run with an agent session.
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL": "https://example.com/repo",
		"image":   "alpine:3",
		"agentSession": map[string]any{
			"agent": "codex",
		},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create run status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var run runResponse
	_ = json.Unmarshal(b, &run)

	// Create a second harness run WITHOUT an agent session.
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL": "https://example.com/repo",
		"image":   "alpine:3",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create non-agent run status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// GET /workspace-sessions/{id}/agent-sessions should return exactly one session.
	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/agent-sessions", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list agent sessions status = %d (body=%s)", resp.StatusCode, string(b))
	}

	var payload struct {
		AgentSessions []agentSessionListItem `json:"agentSessions"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.AgentSessions) != 1 {
		t.Fatalf("expected 1 agent session, got %d: %s", len(payload.AgentSessions), string(b))
	}
	as := payload.AgentSessions[0]
	if as.RunID != run.ID {
		t.Fatalf("runId = %q, want %q", as.RunID, run.ID)
	}
	if as.Agent != "codex" {
		t.Fatalf("agent = %q, want codex", as.Agent)
	}
	if as.Runtime != "sandbox-agent" {
		t.Fatalf("runtime = %q, want sandbox-agent", as.Runtime)
	}
	if as.WorkspaceID != sess.ID {
		t.Fatalf("workspaceSessionId = %q, want %q", as.WorkspaceID, sess.ID)
	}
	if as.Phase != "Provisioning" {
		t.Fatalf("phase = %q, want Provisioning", as.Phase)
	}
	if !strings.Contains(as.DisplayName, "test-ws") {
		t.Fatalf("displayName = %q, expected to contain test-ws", as.DisplayName)
	}
}

func TestWorkspaceAgentSessionsList_NotFound(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, _ := doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/nonexistent/agent-sessions", "full", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestWorkspaceAgentSessionsList_EmptyWhenNoAgentRuns(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{
		"workspace-session:write", "workspace-session:read",
		"harness-run:write", "harness-run:read",
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create a workspace session.
	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{
		"repoURL": "https://example.com/repo",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)

	// Create a harness run WITHOUT an agent session.
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL": "https://example.com/repo",
		"image":   "alpine:3",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create run status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// GET /workspace-sessions/{id}/agent-sessions should return empty list.
	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/agent-sessions", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list agent sessions status = %d (body=%s)", resp.StatusCode, string(b))
	}

	var payload struct {
		AgentSessions []agentSessionListItem `json:"agentSessions"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.AgentSessions) != 0 {
		t.Fatalf("expected 0 agent sessions, got %d", len(payload.AgentSessions))
	}
}

func TestWorkspaceAgentSessionsList_WithLiveBridgeState(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, newAgentSessionStore(""))

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{
		"workspace-session:write", "workspace-session:read",
		"harness-run:write", "harness-run:read",
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create workspace + run with agent session.
	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions", "full", map[string]any{
		"repoURL": "https://example.com/repo",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/harness-runs", "full", map[string]any{
		"repoURL":      "https://example.com/repo",
		"image":        "alpine:3",
		"agentSession": map[string]any{"agent": "codex"},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create run status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var run runResponse
	_ = json.Unmarshal(b, &run)

	// Simulate the pod being ready and create an agent session via the API.
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: run.ID}, &stored); err != nil {
		t.Fatalf("get run: %v", err)
	}
	stored.Status.PodName = "pod-test"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.ID+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent session status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// Now list agent sessions — should show the live bridge state with sessionId.
	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/agent-sessions", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list agent sessions status = %d (body=%s)", resp.StatusCode, string(b))
	}

	var payload struct {
		AgentSessions []agentSessionListItem `json:"agentSessions"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.AgentSessions) != 1 {
		t.Fatalf("expected 1 agent session, got %d", len(payload.AgentSessions))
	}
	as := payload.AgentSessions[0]
	if as.SessionID != "sas-123" {
		t.Fatalf("sessionId = %q, want sas-123", as.SessionID)
	}
	if as.Phase != "Ready" {
		t.Fatalf("phase = %q, want Ready", as.Phase)
	}
}

func int32Ptr(v int32) *int32 { return &v }

// TestAgentSessionWireFormat_EventFieldNames verifies that the events endpoint
// returns JSON with field names that match the CLI's AgentSessionEvent struct
// (seq, timestamp, data) rather than the old internal names (sequence, at, envelope).
func TestAgentSessionWireFormat_EventFieldNames(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()
	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, newAgentSessionStore(""))

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"harness-run:write", "harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-wire", Namespace: api.Namespace},
		Spec: operatorv1alpha1.HarnessRunSpec{
			RepoURL:    "https://example.com/repo",
			Image:      "kocao/harness-runtime:dev",
			WorkingDir: "/workspace/repo",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindClaude,
			},
		},
	}
	if err := api.K8s.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(run), &stored); err != nil {
		t.Fatalf("get run: %v", err)
	}
	stored.Status.PodName = "pod-wire"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create agent session
	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// Wait for stream writer to be established
	transport.waitWriter(t, 5*time.Second)

	// Send prompt
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/prompt", "full", map[string]any{"prompt": "wire format test"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prompt status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// Verify prompt response contains events array with CLI-compatible field names
	var promptPayload map[string]json.RawMessage
	if err := json.Unmarshal(b, &promptPayload); err != nil {
		t.Fatalf("unmarshal prompt response: %v", err)
	}
	if _, ok := promptPayload["events"]; !ok {
		t.Fatalf("prompt response missing 'events' key; keys: %v", keysOf(promptPayload))
	}

	// Verify the events use CLI-compatible field names (seq, timestamp, data)
	var events []map[string]json.RawMessage
	if err := json.Unmarshal(promptPayload["events"], &events); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least one event in prompt response")
	}
	for _, ev := range events {
		if _, ok := ev["seq"]; !ok {
			t.Fatalf("event missing 'seq' field; keys: %v", keysOf(ev))
		}
		if _, ok := ev["timestamp"]; !ok {
			t.Fatalf("event missing 'timestamp' field; keys: %v", keysOf(ev))
		}
		if _, ok := ev["data"]; !ok {
			t.Fatalf("event missing 'data' field; keys: %v", keysOf(ev))
		}
		// Verify old field names are NOT present
		if _, ok := ev["sequence"]; ok {
			t.Fatal("event should not have old 'sequence' field")
		}
		if _, ok := ev["at"]; ok {
			t.Fatal("event should not have old 'at' field")
		}
		if _, ok := ev["envelope"]; ok {
			t.Fatal("event should not have old 'envelope' field")
		}
	}

	// Wait for SSE events to propagate to the store
	var eventsPayload struct {
		Events []json.RawMessage `json:"events"`
	}
	for i := 0; i < 20; i++ {
		resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/events?offset=0&limit=10", "full", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("events status = %d (body=%s)", resp.StatusCode, string(b))
		}
		_ = json.Unmarshal(b, &eventsPayload)
		if len(eventsPayload.Events) >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(eventsPayload.Events) == 0 {
		t.Fatal("expected at least one event from events endpoint")
	}

	// Verify events endpoint also uses CLI-compatible field names
	for _, raw := range eventsPayload.Events {
		var ev map[string]json.RawMessage
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if _, ok := ev["seq"]; !ok {
			t.Fatalf("events endpoint: event missing 'seq' field; keys: %v", keysOf(ev))
		}
		if _, ok := ev["timestamp"]; !ok {
			t.Fatalf("events endpoint: event missing 'timestamp' field; keys: %v", keysOf(ev))
		}
		if _, ok := ev["data"]; !ok {
			t.Fatalf("events endpoint: event missing 'data' field; keys: %v", keysOf(ev))
		}
	}
}

func keysOf[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestAgentSessionWireFormat_StopDeleteBeforeStreamCancel verifies that the
// Stop method sends the DELETE to the sandbox-agent before cancelling the SSE
// stream, preventing the timeout that occurs when the stream is torn down first.
func TestAgentSessionWireFormat_StopDeleteBeforeStreamCancel(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()
	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, newAgentSessionStore(""))

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"harness-run:write", "harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-stop-order", Namespace: api.Namespace},
		Spec: operatorv1alpha1.HarnessRunSpec{
			RepoURL:    "https://example.com/repo",
			Image:      "kocao/harness-runtime:dev",
			WorkingDir: "/workspace/repo",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindClaude,
			},
		},
	}
	if err := api.K8s.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(run), &stored); err != nil {
		t.Fatalf("get run: %v", err)
	}
	stored.Status.PodName = "pod-stop"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update run status: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create session
	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// Wait for stream to be established
	transport.waitWriter(t, 5*time.Second)

	// Stop should complete within a reasonable time (not hang)
	stopDone := make(chan struct{})
	go func() {
		resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+run.Name+"/agent-session/stop", "full", nil)
		close(stopDone)
	}()

	select {
	case <-stopDone:
		// Good - stop completed
	case <-time.After(5 * time.Second):
		t.Fatal("stop timed out - DELETE may be blocked by stream cancellation")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d (body=%s)", resp.StatusCode, string(b))
	}

	// Verify the transport's delete was called
	transport.mu.Lock()
	deleted := transport.deleted
	transport.mu.Unlock()
	if !deleted {
		t.Fatal("expected DeleteACP to be called")
	}
}
