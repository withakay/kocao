package controlplaneapi

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	"github.com/withakay/kocao/internal/operator/controllers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	attachClaimSessionID = "attach.sessionID"
	attachClaimClientID  = "attach.clientID"
	attachClaimRole      = "attach.role"
)

var (
	attachTokenTTL               = 2 * time.Minute
	attachDriverLease            = 30 * time.Second
	attachCleanupGrace           = 5 * time.Second
	attachInitialTermCols uint16 = 80
	attachInitialTermRows uint16 = 24
)

type AttachRole string

const (
	AttachRoleViewer AttachRole = "viewer"
	AttachRoleDriver AttachRole = "driver"
)

func normalizeAttachRole(v string) (AttachRole, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "viewer", "read", "readonly", "read-only":
		return AttachRoleViewer, true
	case "driver", "write", "interactive":
		return AttachRoleDriver, true
	default:
		return "", false
	}
}

type attachTokenRequest struct {
	Role     string `json:"role,omitempty"`
	ClientID string `json:"clientID,omitempty"`
}

type attachTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	SessionID string    `json:"sessionID"`
	ClientID  string    `json:"clientID"`
	Role      string    `json:"role"`
}

type attachMsg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`

	Message   string `json:"message,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
	ClientID  string `json:"clientID,omitempty"`
	Role      string `json:"role,omitempty"`
	DriverID  string `json:"driverID,omitempty"`
	LeaseMS   int64  `json:"leaseMS,omitempty"`
}

type attachClient struct {
	conn     *websocket.Conn
	clientID string
	maxRole  AttachRole
	role     AttachRole
	send     chan attachMsg
}

type attachSession struct {
	namespace string
	sessionID string
	restCfg   *rest.Config
	clientset kubernetes.Interface

	mu      sync.Mutex
	clients map[string]*attachClient

	driverClientID   string
	driverLeaseUntil time.Time

	stdinW *io.PipeWriter
	sizeCh chan remotecommand.TerminalSize

	backendCancel context.CancelFunc

	cleanupTimer *time.Timer
}

func newAttachSession(ns, sessionID string, restCfg *rest.Config) (*attachSession, error) {
	if restCfg == nil {
		return nil, errors.New("rest config required")
	}
	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return &attachSession{
		namespace: ns,
		sessionID: sessionID,
		restCfg:   restCfg,
		clientset: cs,
		clients:   map[string]*attachClient{},
		sizeCh:    make(chan remotecommand.TerminalSize, 8),
	}, nil
}

func (s *attachSession) Next() *remotecommand.TerminalSize {
	sz, ok := <-s.sizeCh
	if !ok {
		return nil
	}
	return &sz
}

func (s *attachSession) broadcastLocked(msg attachMsg) {
	for _, c := range s.clients {
		select {
		case c.send <- msg:
		default:
			// Drop if slow.
		}
	}
}

func (s *attachSession) currentDriverLocked(now time.Time) (string, time.Duration) {
	if s.driverClientID == "" {
		return "", 0
	}
	if now.After(s.driverLeaseUntil) {
		return "", 0
	}
	return s.driverClientID, time.Until(s.driverLeaseUntil)
}

func (s *attachSession) refreshLeaseLocked(now time.Time, clientID string) {
	s.driverClientID = clientID
	s.driverLeaseUntil = now.Add(attachDriverLease)
}

func (s *attachSession) ensureBackendLocked(ctx context.Context, podName string) error {
	if s.backendCancel != nil {
		return nil
	}
	backendCtx, cancel := context.WithCancel(ctx)
	s.backendCancel = cancel

	stdinR, stdinW := io.Pipe()
	s.stdinW = stdinW

	// Seed initial size.
	select {
	case s.sizeCh <- remotecommand.TerminalSize{Width: attachInitialTermCols, Height: attachInitialTermRows}:
	default:
	}

	go func() {
		defer func() {
			_ = stdinR.Close()
			s.mu.Lock()
			s.backendCancel = nil
			s.stdinW = nil
			s.mu.Unlock()
		}()

		req := s.clientset.CoreV1().RESTClient().Post().
			Namespace(s.namespace).
			Resource("pods").
			Name(podName).
			SubResource("exec")
		opts := &corev1.PodExecOptions{
			Container: "harness",
			Command:   []string{"sh"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}
		req.VersionedParams(opts, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(s.restCfg, http.MethodPost, req.URL())
		if err != nil {
			s.mu.Lock()
			s.broadcastLocked(attachMsg{Type: "error", Message: "backend exec init failed"})
			s.mu.Unlock()
			return
		}
		w := &attachBackendWriter{sess: s}
		_ = exec.StreamWithContext(backendCtx, remotecommand.StreamOptions{
			Stdin:             stdinR,
			Stdout:            w,
			Stderr:            w,
			Tty:               true,
			TerminalSizeQueue: s,
		})

		s.mu.Lock()
		s.broadcastLocked(attachMsg{Type: "backend_closed"})
		s.mu.Unlock()
	}()

	return nil
}

type attachBackendWriter struct {
	sess *attachSession
}

func (w *attachBackendWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	msg := attachMsg{Type: "stdout", Data: base64.StdEncoding.EncodeToString(p)}
	w.sess.mu.Lock()
	w.sess.broadcastLocked(msg)
	w.sess.mu.Unlock()
	return len(p), nil
}

type AttachService struct {
	namespace string
	restCfg   *rest.Config
	k8s       client.Client
	tokens    *TokenStore
	audit     *AuditStore

	mu       sync.Mutex
	sessions map[string]*attachSession
}

func newAttachService(ns string, restCfg *rest.Config, k8s client.Client, tokens *TokenStore, audit *AuditStore) *AttachService {
	return &AttachService{namespace: ns, restCfg: restCfg, k8s: k8s, tokens: tokens, audit: audit, sessions: map[string]*attachSession{}}
}

func (s *AttachService) issueToken(ctx context.Context, principalID string, sessionID string, role AttachRole, clientID string) (attachTokenResponse, error) {
	if strings.TrimSpace(clientID) == "" {
		clientID = newID()
	}
	raw := newID() + newID()
	exp := time.Now().Add(attachTokenTTL)
	claims := map[string]string{
		attachClaimSessionID: sessionID,
		attachClaimClientID:  clientID,
		attachClaimRole:      string(role),
	}
	if err := s.tokens.CreateWithClaims(ctx, "attach-"+principalID, raw, []string{"attach:connect"}, exp, claims); err != nil {
		return attachTokenResponse{}, err
	}
	return attachTokenResponse{Token: raw, ExpiresAt: exp, SessionID: sessionID, ClientID: clientID, Role: string(role)}, nil
}

func (s *AttachService) claimsFromToken(ctx context.Context, raw string) (string, string, AttachRole, error) {
	rec, err := s.tokens.Lookup(ctx, raw)
	if err != nil {
		return "", "", "", err
	}
	if rec == nil || rec.Claims == nil {
		return "", "", "", errors.New("invalid token")
	}
	sessionID := strings.TrimSpace(rec.Claims[attachClaimSessionID])
	clientID := strings.TrimSpace(rec.Claims[attachClaimClientID])
	role, ok := normalizeAttachRole(rec.Claims[attachClaimRole])
	if sessionID == "" || clientID == "" || !ok {
		return "", "", "", errors.New("invalid token claims")
	}
	return sessionID, clientID, role, nil
}

func (s *AttachService) findAttachPod(ctx context.Context, sessionID string) (string, error) {
	var runs operatorv1alpha1.HarnessRunList
	if err := s.k8s.List(ctx, &runs, client.InNamespace(s.namespace), client.MatchingLabels{controllers.LabelSessionName: sessionID}); err != nil {
		return "", err
	}
	var startingPod string
	for i := range runs.Items {
		run := &runs.Items[i]
		if strings.TrimSpace(run.Status.PodName) == "" {
			continue
		}
		switch run.Status.Phase {
		case operatorv1alpha1.HarnessRunPhaseRunning:
			return run.Status.PodName, nil
		case operatorv1alpha1.HarnessRunPhaseStarting:
			startingPod = run.Status.PodName
		}
	}
	if startingPod != "" {
		return startingPod, nil
	}
	return "", errors.New("no active run pod")
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func (a *API) handleAttachToken(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "attach.token.issue", "session", sessionID
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Attach == nil {
			writeError(w, http.StatusInternalServerError, "attach service not configured")
			return
		}
		var sess operatorv1alpha1.Session
		if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: sessionID}, &sess); err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get session failed")
			return
		}
		enabled := false
		if sess.Annotations != nil {
			v := strings.TrimSpace(sess.Annotations[annotationAttachEnabled])
			enabled = strings.EqualFold(v, "true")
		}
		if !enabled {
			writeError(w, http.StatusForbidden, "attach disabled")
			return
		}
		var req attachTokenRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		role, ok := normalizeAttachRole(req.Role)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid role")
			return
		}
		if role == AttachRoleDriver {
			p, _ := principalFrom(r.Context())
			if p == nil || !hasScope(p.Scopes, "control:write") {
				writeError(w, http.StatusForbidden, "driver role requires control:write")
				return
			}
		}
		p := principal(r.Context())
		resp, err := a.Attach.issueToken(r.Context(), p, sessionID, role, req.ClientID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "issue attach token failed")
			return
		}
		a.Audit.Append(r.Context(), p, "attach.token.issued", "session", sessionID, "allowed", map[string]any{"role": resp.Role})
		writeJSON(w, http.StatusCreated, resp)
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleAttachWS(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if a.Attach == nil {
		writeError(w, http.StatusInternalServerError, "attach service not configured")
		return
	}

	tok := strings.TrimSpace(r.URL.Query().Get("token"))
	if tok == "" {
		tok = bearerToken(r)
	}
	if tok == "" {
		writeError(w, http.StatusUnauthorized, "missing attach token")
		return
	}

	claimedSessionID, clientID, role, err := a.Attach.claimsFromToken(r.Context(), tok)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid attach token")
		return
	}
	if claimedSessionID != sessionID {
		writeError(w, http.StatusForbidden, "attach token session mismatch")
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	a.Attach.handleConn(r.Context(), sessionID, clientID, role, conn)
}

func (s *AttachService) handleConn(ctx context.Context, sessionID, clientID string, role AttachRole, conn *websocket.Conn) {
	defer func() { _ = conn.Close() }()

	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		created, err := newAttachSession(s.namespace, sessionID, s.restCfg)
		if err != nil {
			s.mu.Unlock()
			_ = conn.WriteJSON(attachMsg{Type: "error", Message: "attach not available"})
			return
		}
		sess = created
		s.sessions[sessionID] = sess
	}
	s.mu.Unlock()

	cli := &attachClient{conn: conn, clientID: clientID, maxRole: role, role: role, send: make(chan attachMsg, 64)}

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for msg := range cli.send {
			_ = conn.WriteJSON(msg)
		}
	}()

	now := time.Now()
	sess.mu.Lock()
	if sess.cleanupTimer != nil {
		sess.cleanupTimer.Stop()
		sess.cleanupTimer = nil
	}
	curDriver, _ := sess.currentDriverLocked(now)
	// Driver reconnect gets the role even if token is viewer.
	if sess.driverClientID == clientID && curDriver == clientID {
		cli.role = AttachRoleDriver
		sess.refreshLeaseLocked(now, clientID)
	} else if cli.maxRole == AttachRoleDriver && curDriver == "" {
		// No active driver; driver-capable token can claim the lease.
		cli.role = AttachRoleDriver
		sess.refreshLeaseLocked(now, clientID)
	} else {
		// Active lease held by someone else; join as viewer until lease transfer.
		cli.role = AttachRoleViewer
	}
	sess.clients[clientID] = cli
	if sess.backendCancel == nil {
		podName, err := s.findAttachPod(ctx, sessionID)
		if err == nil {
			_ = sess.ensureBackendLocked(ctx, podName)
		}
	}
	driverID, lease := sess.currentDriverLocked(now)
	sess.broadcastLocked(attachMsg{Type: "state", DriverID: driverID, LeaseMS: lease.Milliseconds()})
	sess.mu.Unlock()

	cli.send <- attachMsg{Type: "hello", SessionID: sessionID, ClientID: clientID, Role: string(cli.role), DriverID: driverID, LeaseMS: lease.Milliseconds()}

	for {
		var m attachMsg
		err := conn.ReadJSON(&m)
		if err != nil {
			break
		}
		switch m.Type {
		case "keepalive":
			now := time.Now()
			sess.mu.Lock()
			if sess.driverClientID == clientID {
				sess.refreshLeaseLocked(now, clientID)
			}
			driverID, lease := sess.currentDriverLocked(now)
			sess.mu.Unlock()
			cli.send <- attachMsg{Type: "state", DriverID: driverID, LeaseMS: lease.Milliseconds()}
		case "resize":
			if m.Cols > 0 && m.Rows > 0 {
				select {
				case sess.sizeCh <- remotecommand.TerminalSize{Width: uint16(m.Cols), Height: uint16(m.Rows)}:
				default:
				}
			}
		case "take_control":
			if cli.maxRole != AttachRoleDriver {
				cli.send <- attachMsg{Type: "error", Message: "insufficient role"}
				continue
			}
			now := time.Now()
			sess.mu.Lock()
			cur, _ := sess.currentDriverLocked(now)
			if cur == "" || cur == clientID {
				sess.refreshLeaseLocked(now, clientID)
				cli.role = AttachRoleDriver
				cur = clientID
			}
			lease := time.Until(sess.driverLeaseUntil)
			sess.broadcastLocked(attachMsg{Type: "state", DriverID: cur, LeaseMS: lease.Milliseconds()})
			sess.mu.Unlock()
		case "stdin":
			payload, err := base64.StdEncoding.DecodeString(m.Data)
			if err != nil {
				cli.send <- attachMsg{Type: "error", Message: "invalid stdin payload"}
				continue
			}
			now := time.Now()
			sess.mu.Lock()
			cur, _ := sess.currentDriverLocked(now)
			w := sess.stdinW
			if cur == clientID {
				sess.refreshLeaseLocked(now, clientID)
			}
			lease := time.Until(sess.driverLeaseUntil)
			sess.broadcastLocked(attachMsg{Type: "state", DriverID: sess.driverClientID, LeaseMS: lease.Milliseconds()})
			sess.mu.Unlock()
			if cur != clientID {
				cli.send <- attachMsg{Type: "error", Message: "read-only"}
				continue
			}
			if w == nil {
				podName, err := s.findAttachPod(ctx, sessionID)
				if err != nil {
					cli.send <- attachMsg{Type: "error", Message: "no active run pod"}
					continue
				}
				sess.mu.Lock()
				err = sess.ensureBackendLocked(ctx, podName)
				w = sess.stdinW
				sess.mu.Unlock()
				if err != nil {
					cli.send <- attachMsg{Type: "error", Message: "backend unavailable"}
					continue
				}
			}
			_, _ = w.Write(payload)
		default:
			cli.send <- attachMsg{Type: "error", Message: "unknown message type"}
		}
	}

	sess.mu.Lock()
	delete(sess.clients, clientID)
	close(cli.send)
	noClients := len(sess.clients) == 0
	leaseExpires := sess.driverLeaseUntil
	stillDriver := sess.driverClientID == clientID
	if stillDriver {
		// Keep the lease for reconnect; do not clear driverClientID.
	}
	// Schedule cleanup when last client disconnects.
	if noClients {
		delay := time.Until(leaseExpires) + attachCleanupGrace
		if delay < attachCleanupGrace {
			delay = attachCleanupGrace
		}
		sess.cleanupTimer = time.AfterFunc(delay, func() {
			s.mu.Lock()
			cur := s.sessions[sessionID]
			if cur == nil {
				s.mu.Unlock()
				return
			}
			cur.mu.Lock()
			should := len(cur.clients) == 0 && (cur.driverClientID == "" || time.Now().After(cur.driverLeaseUntil))
			cancel := cur.backendCancel
			cur.mu.Unlock()
			if should {
				if cancel != nil {
					cancel()
				}
				delete(s.sessions, sessionID)
			}
			s.mu.Unlock()
		})
	}
	sess.mu.Unlock()

	<-writeDone
}
