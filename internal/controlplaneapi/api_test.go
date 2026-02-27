package controlplaneapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()

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
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
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
