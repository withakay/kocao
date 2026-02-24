package controlplaneapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func waitForAudit(t *testing.T, api *API, action string, outcome string) AuditEvent {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		evs, err := api.Audit.List(context.Background(), 500)
		if err != nil {
			t.Fatalf("audit list: %v", err)
		}
		for _, e := range evs {
			if e.Action == action && e.Outcome == outcome {
				return e
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for audit action=%q outcome=%q", action, outcome)
	return AuditEvent{}
}

func newTestAPIWithAttach(t *testing.T) (*API, func()) {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()

	// This host is never dialed in tests unless a backend exec is started.
	restCfg := &rest.Config{Host: "https://example.invalid"}

	api, err := New("test-ns", "", "", restCfg, k8s, Options{Env: "test"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return api, func() {}
}

func wsURL(httpURL string, path string, q url.Values) string {
	u, _ := url.Parse(httpURL)
	u.Scheme = "ws"
	u.Path = path
	u.RawQuery = q.Encode()
	return u.String()
}

func readMsgType(t *testing.T, c *websocket.Conn, want string) attachMsg {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	_ = c.SetReadDeadline(deadline)
	for {
		var m attachMsg
		if err := c.ReadJSON(&m); err != nil {
			t.Fatalf("ReadJSON error: %v", err)
		}
		if m.Type == want {
			return m
		}
	}
}

func readStateDriverIn(t *testing.T, c *websocket.Conn, a string, b string) attachMsg {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	_ = c.SetReadDeadline(deadline)
	for {
		var m attachMsg
		if err := c.ReadJSON(&m); err != nil {
			t.Fatalf("ReadJSON error: %v", err)
		}
		if m.Type != "state" {
			continue
		}
		if m.DriverID == a || m.DriverID == b {
			return m
		}
	}
}

func readStateDriverEquals(t *testing.T, c *websocket.Conn, want string) attachMsg {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	_ = c.SetReadDeadline(deadline)
	for {
		var m attachMsg
		if err := c.ReadJSON(&m); err != nil {
			t.Fatalf("ReadJSON error: %v", err)
		}
		if m.Type == "state" && m.DriverID == want {
			return m
		}
	}
}

func TestAttachToken_DisabledForbidden(t *testing.T) {
	api, cleanup := newTestAPIWithAttach(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read", "harness-run:write", "harness-run:read", "control:write"}); err != nil {
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

	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "viewer"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("attach-token status = %d, want 403", resp.StatusCode)
	}
}

func TestAttachToken_RoleEnforcementAndTakeControl(t *testing.T) {
	oldLease := attachDriverLease
	oldTTL := attachTokenTTL
	oldGrace := attachCleanupGrace
	attachDriverLease = 500 * time.Millisecond
	attachTokenTTL = 5 * time.Second
	attachCleanupGrace = 50 * time.Millisecond
	t.Cleanup(func() {
		attachDriverLease = oldLease
		attachTokenTTL = oldTTL
		attachCleanupGrace = oldGrace
	})

	api, cleanup := newTestAPIWithAttach(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "workspace-session:read", "harness-run:write", "harness-run:read", "control:write"}); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := api.Tokens.Create(context.Background(), "t-run", "run", []string{"harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	// Create session and enable attach.
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

	// Driver token requires control:write.
	resp, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "run", map[string]any{"role": "driver"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("attach-token(driver) status = %d, want 403", resp.StatusCode)
	}

	// Viewer token.
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "viewer"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token(viewer) status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var viewerTok attachTokenResponse
	_ = json.Unmarshal(b, &viewerTok)

	// Driver token.
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "driver"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token(driver) status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var driverTok1 attachTokenResponse
	_ = json.Unmarshal(b, &driverTok1)

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "driver"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token(driver2) status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var driverTok2 attachTokenResponse
	_ = json.Unmarshal(b, &driverTok2)

	// Viewer cannot take control.
	viewerConn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{viewerTok.Token}}), nil)
	if err != nil {
		t.Fatalf("dial viewer: %v", err)
	}
	defer func() { _ = viewerConn.Close() }()
	_ = readMsgType(t, viewerConn, "hello")
	_ = viewerConn.WriteJSON(attachMsg{Type: "take_control"})
	errMsg := readMsgType(t, viewerConn, "error")
	if errMsg.Message != "insufficient role" {
		t.Fatalf("error message = %q, want %q", errMsg.Message, "insufficient role")
	}

	// Driver lease blocks other drivers until expiry.
	d1, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{driverTok1.Token}}), nil)
	if err != nil {
		t.Fatalf("dial driver1: %v", err)
	}
	defer func() { _ = d1.Close() }()
	_ = readMsgType(t, d1, "hello")
	_ = d1.WriteJSON(attachMsg{Type: "take_control"})
	state := readStateDriverEquals(t, d1, driverTok1.ClientID)
	if state.DriverID != driverTok1.ClientID {
		t.Fatalf("driver after take_control = %q, want %q", state.DriverID, driverTok1.ClientID)
	}

	d2, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{driverTok2.Token}}), nil)
	if err != nil {
		t.Fatalf("dial driver2: %v", err)
	}
	defer func() { _ = d2.Close() }()
	_ = readMsgType(t, d2, "hello")
	_ = d2.WriteJSON(attachMsg{Type: "take_control"})
	state2 := readStateDriverIn(t, d2, driverTok1.ClientID, driverTok2.ClientID)
	if state2.DriverID != driverTok1.ClientID {
		t.Fatalf("driver while lease active = %q, want %q", state2.DriverID, driverTok1.ClientID)
	}

	// After lease expiry, driver2 can take control.
	time.Sleep(2 * attachDriverLease)
	_ = d2.WriteJSON(attachMsg{Type: "take_control"})
	state3 := readStateDriverEquals(t, d2, driverTok2.ClientID)
	if state3.DriverID != driverTok2.ClientID {
		t.Fatalf("driver after expiry = %q, want %q", state3.DriverID, driverTok2.ClientID)
	}
}

func TestAttachWS_ReadLimit_ClosesConnection(t *testing.T) {
	oldLimit := attachWSReadLimit
	oldPongWait := attachWSPongWait
	oldPingPeriod := attachWSPingPeriod
	oldWriteWait := attachWSWriteWait
	attachWSReadLimit = 64
	attachWSPongWait = 500 * time.Millisecond
	attachWSPingPeriod = 100 * time.Millisecond
	attachWSWriteWait = 200 * time.Millisecond
	t.Cleanup(func() {
		attachWSReadLimit = oldLimit
		attachWSPongWait = oldPongWait
		attachWSPingPeriod = oldPingPeriod
		attachWSWriteWait = oldWriteWait
	})

	api, cleanup := newTestAPIWithAttach(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "harness-run:read", "control:write", "harness-run:write", "workspace-session:read"}); err != nil {
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

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "driver"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var tok attachTokenResponse
	_ = json.Unmarshal(b, &tok)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{tok.Token}}), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readMsgType(t, conn, "hello")

	big := strings.Repeat("a", int(attachWSReadLimit)+1)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(big)); err != nil {
		t.Fatalf("WriteMessage error: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected connection close after oversized message")
	}
}

func TestAttachWS_IdleWithoutPong_ClosesConnection(t *testing.T) {
	oldPongWait := attachWSPongWait
	oldPingPeriod := attachWSPingPeriod
	oldWriteWait := attachWSWriteWait
	attachWSPongWait = 200 * time.Millisecond
	attachWSPingPeriod = 50 * time.Millisecond
	attachWSWriteWait = 200 * time.Millisecond
	t.Cleanup(func() {
		attachWSPongWait = oldPongWait
		attachWSPingPeriod = oldPingPeriod
		attachWSWriteWait = oldWriteWait
	})

	api, cleanup := newTestAPIWithAttach(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"workspace-session:write", "harness-run:read", "control:write", "harness-run:write", "workspace-session:read"}); err != nil {
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

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{tok.Token}}), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	conn.SetPingHandler(func(string) error { return nil })
	_ = readMsgType(t, conn, "hello")

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatalf("expected connection close when client does not pong")
	}
}

func TestAttachCookie_IssuesCookieAndAllowsWSWithoutQueryToken(t *testing.T) {
	api, cleanup := newTestAPIWithAttach(t)
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

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-cookie", "full", map[string]any{"role": "viewer"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-cookie status = %d (body=%s)", resp.StatusCode, string(b))
	}
	if strings.Contains(string(b), "token") {
		t.Fatalf("expected attach-cookie response to omit token")
	}
	var issued *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == attachCookieName {
			issued = c
			break
		}
	}
	if issued == nil {
		t.Fatalf("expected %s cookie", attachCookieName)
	}
	if issued.Value == "" {
		t.Fatalf("expected cookie value")
	}
	if issued.Path == "" {
		t.Fatalf("expected cookie path")
	}

	h := http.Header{}
	h.Add("Cookie", issued.Name+"="+issued.Value)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{}), h)
	if err != nil {
		t.Fatalf("dial with cookie: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = readMsgType(t, conn, "hello")
}

func TestAttachWS_AuditsLifecycleControlAndStdin(t *testing.T) {
	api, cleanup := newTestAPIWithAttach(t)
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

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/workspace-sessions/"+sess.ID+"/attach-token", "full", map[string]any{"role": "driver"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("attach-token status = %d (body=%s)", resp.StatusCode, string(b))
	}
	var tok attachTokenResponse
	_ = json.Unmarshal(b, &tok)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(srv.URL, "/api/v1/workspace-sessions/"+sess.ID+"/attach", url.Values{"token": []string{tok.Token}}), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = readMsgType(t, conn, "hello")

	connect := waitForAudit(t, api, "attach.connect", "allowed")
	if connect.Actor != "t-full" {
		t.Fatalf("connect actor=%q, want t-full", connect.Actor)
	}
	var meta map[string]any
	_ = json.Unmarshal(connect.Metadata, &meta)
	if meta["clientID"] != tok.ClientID {
		t.Fatalf("connect clientID=%v, want %q", meta["clientID"], tok.ClientID)
	}

	_ = conn.WriteJSON(attachMsg{Type: "take_control"})
	_ = readMsgType(t, conn, "state")
	ctrl := waitForAudit(t, api, "attach.control.acquired", "allowed")
	_ = json.Unmarshal(ctrl.Metadata, &meta)
	if meta["clientID"] != tok.ClientID {
		t.Fatalf("control clientID=%v, want %q", meta["clientID"], tok.ClientID)
	}

	_ = conn.WriteJSON(attachMsg{Type: "stdin", Data: base64.StdEncoding.EncodeToString([]byte("echo hi\n"))})
	stdin := waitForAudit(t, api, "attach.stdin", "allowed")
	_ = json.Unmarshal(stdin.Metadata, &meta)
	if meta["clientID"] != tok.ClientID {
		t.Fatalf("stdin clientID=%v, want %q", meta["clientID"], tok.ClientID)
	}

	_ = conn.Close()
	_ = waitForAudit(t, api, "attach.disconnect", "allowed")
}
