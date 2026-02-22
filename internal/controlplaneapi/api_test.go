package controlplaneapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
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

	api, err := New("test-ns", "", "", nil, k8s)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	cleanup := func() {}
	return api, cleanup
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

func TestAuth_MissingToken_DeniedAndAudited(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, _ := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/sessions", "", map[string]any{"repoURL": "https://example.com/repo"})
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

	if err := api.Tokens.Create(context.Background(), "t-readonly", "readonly", []string{"session:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, _ := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/sessions", "readonly", map[string]any{"repoURL": "https://example.com/repo"})
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

func TestLifecycle_SessionRunControlsAndAudit(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"session:write", "session:read", "run:write", "run:read", "control:write", "audit:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/sessions", "full", map[string]any{"repoURL": "https://example.com/repo"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var sess sessionResponse
	_ = json.Unmarshal(b, &sess)
	if sess.ID == "" {
		t.Fatalf("expected session id")
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/sessions/"+sess.ID+"/attach-control", "full", map[string]any{"enabled": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("attach control status = %d, want 200", resp.StatusCode)
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPatch, srv.URL+"/api/v1/sessions/"+sess.ID+"/egress-override", "full", map[string]any{"mode": "allowlist", "allowedHosts": []string{"github.com"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("egress override status = %d, want 200", resp.StatusCode)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/sessions/"+sess.ID+"/runs", "full", map[string]any{
		"repoURL":      "https://example.com/repo",
		"repoRevision": "main",
		"image":        "alpine:3",
		"env":          []map[string]any{{"name": "GITHUB_TOKEN", "value": "redacted"}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("start run status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var run runResponse
	_ = json.Unmarshal(b, &run)
	if run.ID == "" || run.SessionID != sess.ID {
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
	stored.Annotations["kocao.withakay.github.com/github-branch"] = "feature/mvp-ui"
	stored.Annotations["kocao.withakay.github.com/pull-request-url"] = "https://github.com/withakay/kocao/pull/123"
	stored.Annotations["kocao.withakay.github.com/pull-request-status"] = "merged"
	if err := api.K8s.Update(context.Background(), &stored); err != nil {
		t.Fatalf("update stored run: %v", err)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/runs/"+run.ID, "full", nil)
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

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/runs/"+run.ID+"/resume", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("resume run status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	var resumed runResponse
	_ = json.Unmarshal(b, &resumed)
	if resumed.ID == "" || resumed.ID == run.ID {
		t.Fatalf("expected new run id")
	}

	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/runs/"+run.ID+"/stop", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop run status = %d, want 200", resp.StatusCode)
	}
	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/runs/"+resumed.ID+"/stop", "full", nil)
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
