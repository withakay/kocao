package controlplaneapi

import (
	"context"
	"errors"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	"github.com/withakay/kocao/internal/operator/controllers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const (
	annotationAttachEnabled = "kocao.withakay.github.com/attach-enabled"
	annotationEgressMode    = "kocao.withakay.github.com/egress-mode"
	annotationEgressHosts   = "kocao.withakay.github.com/egress-allowed-hosts"
)

type API struct {
	Namespace string
	K8s       client.Client
	Auth      *Authenticator
	Tokens    *TokenStore
	Audit     *AuditStore
	Attach    *AttachService
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/openapi.json", openAPIHandler)

	// Health endpoints stay unauthenticated.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	var api http.Handler = http.HandlerFunc(a.serveAPI)
	api = a.Auth.Authenticate(api, a.Audit)
	mux.Handle("/api/", api)

	return mux
}

func (a *API) serveAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	segs := strings.Split(path, "/")
	if len(segs) < 2 || segs[0] != "api" || segs[1] != "v1" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	segs = segs[2:]

	switch {
	case len(segs) == 1 && segs[0] == "sessions":
		a.handleSessions(w, r)
		return
	case len(segs) == 2 && segs[0] == "sessions":
		a.handleSession(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "sessions" && segs[2] == "runs":
		a.handleSessionRuns(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "sessions" && segs[2] == "attach-control":
		a.handleAttachControl(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "sessions" && segs[2] == "attach-token":
		a.handleAttachToken(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "sessions" && segs[2] == "attach":
		a.handleAttachWS(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "sessions" && segs[2] == "egress-override":
		a.handleEgressOverride(w, r, segs[1])
		return
	case len(segs) == 1 && segs[0] == "runs":
		a.handleRuns(w, r)
		return
	case len(segs) == 2 && segs[0] == "runs":
		a.handleRun(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "runs" && segs[2] == "stop":
		a.handleRunStop(w, r, segs[1])
		return
	case len(segs) == 3 && segs[0] == "runs" && segs[2] == "resume":
		a.handleRunResume(w, r, segs[1])
		return
	case len(segs) == 1 && segs[0] == "audit":
		a.handleAudit(w, r)
		return
	default:
		writeError(w, http.StatusNotFound, "not found")
		return
	}
}

type sessionCreateRequest struct {
	RepoURL string `json:"repoURL,omitempty"`
}

type sessionResponse struct {
	ID      string                        `json:"id"`
	RepoURL string                        `json:"repoURL,omitempty"`
	Phase   operatorv1alpha1.SessionPhase `json:"phase,omitempty"`
}

func sessionToResponse(s *operatorv1alpha1.Session) sessionResponse {
	return sessionResponse{ID: s.Name, RepoURL: s.Spec.RepoURL, Phase: s.Status.Phase}
}

func (a *API) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h := requireScopes([]string{"session:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
			return "session.list", "session", "*"
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var list operatorv1alpha1.SessionList
			if err := a.K8s.List(r.Context(), &list, client.InNamespace(a.Namespace)); err != nil {
				writeError(w, http.StatusInternalServerError, "list sessions failed")
				return
			}
			out := make([]sessionResponse, 0, len(list.Items))
			for i := range list.Items {
				out = append(out, sessionToResponse(&list.Items[i]))
			}
			writeJSON(w, http.StatusOK, map[string]any{"sessions": out})
		}))
		h.ServeHTTP(w, r)
		return
	}

	if r.Method == http.MethodPost {
		h := requireScopes([]string{"session:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
			return "session.create", "session", "(new)"
		})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req sessionCreateRequest
			if err := readJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json")
				return
			}
			id := newID()
			sess := &operatorv1alpha1.Session{
				TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
				ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: a.Namespace},
				Spec:       operatorv1alpha1.SessionSpec{RepoURL: req.RepoURL},
			}
			if err := a.K8s.Create(r.Context(), sess); err != nil {
				writeError(w, http.StatusInternalServerError, "create session failed")
				return
			}
			writeJSON(w, http.StatusCreated, sessionToResponse(sess))
		}))
		h.ServeHTTP(w, r)
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (a *API) handleSession(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"session:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "session.get", "session", id
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sess operatorv1alpha1.Session
		err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: id}, &sess)
		if err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get session failed")
			return
		}
		writeJSON(w, http.StatusOK, sessionToResponse(&sess))
	}))
	h.ServeHTTP(w, r)
}

type runCreateRequest struct {
	RepoURL                 string                        `json:"repoURL"`
	RepoRevision            string                        `json:"repoRevision,omitempty"`
	Image                   string                        `json:"image"`
	EgressMode              string                        `json:"egressMode,omitempty"`
	Command                 []string                      `json:"command,omitempty"`
	Args                    []string                      `json:"args,omitempty"`
	WorkingDir              string                        `json:"workingDir,omitempty"`
	Env                     []operatorv1alpha1.EnvVar     `json:"env,omitempty"`
	GitAuth                 *operatorv1alpha1.GitAuthSpec `json:"gitAuth,omitempty"`
	TTLSecondsAfterFinished *int32                        `json:"ttlSecondsAfterFinished,omitempty"`
}

func normalizeRunEgressMode(mode string) (string, bool) {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "":
		return "", true
	case "restricted", "github-only", "github":
		return "restricted", true
	case "full", "full-internet", "internet":
		return "full", true
	default:
		return "", false
	}
}

type runResponse struct {
	ID        string                           `json:"id"`
	SessionID string                           `json:"sessionID,omitempty"`
	RepoURL   string                           `json:"repoURL"`
	Image     string                           `json:"image"`
	Phase     operatorv1alpha1.HarnessRunPhase `json:"phase,omitempty"`
	PodName   string                           `json:"podName,omitempty"`
}

func runToResponse(run *operatorv1alpha1.HarnessRun) runResponse {
	return runResponse{
		ID:        run.Name,
		SessionID: run.Spec.SessionName,
		RepoURL:   run.Spec.RepoURL,
		Image:     run.Spec.Image,
		Phase:     run.Status.Phase,
		PodName:   run.Status.PodName,
	}
}

func (a *API) handleSessionRuns(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "run.start", "session", sessionID
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sess operatorv1alpha1.Session
		if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: sessionID}, &sess); err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get session failed")
			return
		}

		var req runCreateRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		if strings.TrimSpace(req.RepoURL) == "" {
			writeError(w, http.StatusBadRequest, "repoURL required")
			return
		}
		if strings.TrimSpace(req.Image) == "" {
			writeError(w, http.StatusBadRequest, "image required")
			return
		}
		egressMode, ok := normalizeRunEgressMode(req.EgressMode)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid egressMode")
			return
		}

		id := newID()
		run := &operatorv1alpha1.HarnessRun{
			TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
			ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: a.Namespace},
			Spec: operatorv1alpha1.HarnessRunSpec{
				SessionName:             sessionID,
				RepoURL:                 req.RepoURL,
				RepoRevision:            req.RepoRevision,
				Image:                   req.Image,
				EgressMode:              egressMode,
				Command:                 req.Command,
				Args:                    req.Args,
				WorkingDir:              req.WorkingDir,
				Env:                     req.Env,
				GitAuth:                 req.GitAuth,
				TTLSecondsAfterFinished: req.TTLSecondsAfterFinished,
			},
		}
		if err := a.K8s.Create(r.Context(), run); err != nil {
			writeError(w, http.StatusInternalServerError, "create run failed")
			return
		}
		if egressMode == "full" {
			a.Audit.Append(r.Context(), principal(r.Context()), "run.egress.override", "run", run.Name, "allowed", map[string]any{"mode": egressMode})
		}

		// Credential-use audit is derived from env var names and gitAuth presence only.
		var credNames []string
		for _, ev := range req.Env {
			name := strings.ToUpper(strings.TrimSpace(ev.Name))
			if name == "" {
				continue
			}
			if strings.Contains(name, "TOKEN") || strings.Contains(name, "SECRET") || strings.HasSuffix(name, "_KEY") {
				credNames = append(credNames, ev.Name)
			}
		}
		gitAuthUsed := req.GitAuth != nil && strings.TrimSpace(req.GitAuth.SecretName) != ""
		if len(credNames) != 0 || gitAuthUsed {
			meta := map[string]any{}
			if len(credNames) != 0 {
				meta["names"] = credNames
			}
			if gitAuthUsed {
				meta["gitAuth"] = true
			}
			a.Audit.Append(r.Context(), principal(r.Context()), "credential.use", "run", run.Name, "allowed", meta)
		}

		writeJSON(w, http.StatusCreated, runToResponse(run))
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "run.list", "run", "*"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var list operatorv1alpha1.HarnessRunList
		opts := []client.ListOption{client.InNamespace(a.Namespace)}
		if sessionID := strings.TrimSpace(r.URL.Query().Get("sessionID")); sessionID != "" {
			opts = append(opts, client.MatchingLabels{controllers.LabelSessionName: sessionID})
		}
		if err := a.K8s.List(r.Context(), &list, opts...); err != nil {
			writeError(w, http.StatusInternalServerError, "list runs failed")
			return
		}
		out := make([]runResponse, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, runToResponse(&list.Items[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": out})
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleRun(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "run.get", "run", id
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var run operatorv1alpha1.HarnessRun
		err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: id}, &run)
		if err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get run failed")
			return
		}
		writeJSON(w, http.StatusOK, runToResponse(&run))
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleRunStop(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "run.stop", "run", id
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var run operatorv1alpha1.HarnessRun
		if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: id}, &run); err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get run failed")
			return
		}
		if err := a.K8s.Delete(r.Context(), &run); err != nil {
			if apierrors.IsNotFound(err) {
				writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
				return
			}
			writeError(w, http.StatusInternalServerError, "delete run failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleRunResume(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"run:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "run.resume", "run", id
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var run operatorv1alpha1.HarnessRun
		if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: id}, &run); err != nil {
			if apierrors.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "get run failed")
			return
		}
		newID := newID()
		copy := &operatorv1alpha1.HarnessRun{
			TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
			ObjectMeta: metav1.ObjectMeta{Name: newID, Namespace: a.Namespace, Labels: map[string]string{"kocao.withakay.github.com/resumed-from": id}},
			Spec:       run.Spec,
		}
		copy.Spec.TTLSecondsAfterFinished = run.Spec.TTLSecondsAfterFinished
		if err := a.K8s.Create(r.Context(), copy); err != nil {
			writeError(w, http.StatusInternalServerError, "create resumed run failed")
			return
		}
		writeJSON(w, http.StatusCreated, runToResponse(copy))
	}))
	h.ServeHTTP(w, r)
}

type attachControlRequest struct {
	Enabled bool `json:"enabled"`
}

func (a *API) handleAttachControl(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"control:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "attach-control.update", "session", sessionID
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req attachControlRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
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
		updated := sess.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		updated.Annotations[annotationAttachEnabled] = strconv.FormatBool(req.Enabled)
		if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(&sess)); err != nil {
			writeError(w, http.StatusInternalServerError, "update attach control failed")
			return
		}
		a.Audit.Append(r.Context(), principal(r.Context()), "attach-control.changed", "session", sessionID, "allowed", map[string]any{"enabled": req.Enabled})
		writeJSON(w, http.StatusOK, map[string]any{"updated": true})
	}))
	h.ServeHTTP(w, r)
}

type egressOverrideRequest struct {
	Mode         string   `json:"mode"`
	AllowedHosts []string `json:"allowedHosts,omitempty"`
}

func (a *API) handleEgressOverride(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"control:write"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "egress-override.update", "session", sessionID
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req egressOverrideRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		mode := strings.TrimSpace(req.Mode)
		if mode == "" {
			writeError(w, http.StatusBadRequest, "mode required")
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
		updated := sess.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		updated.Annotations[annotationEgressMode] = mode
		if len(req.AllowedHosts) != 0 {
			updated.Annotations[annotationEgressHosts] = strings.Join(req.AllowedHosts, ",")
		} else {
			delete(updated.Annotations, annotationEgressHosts)
		}
		if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(&sess)); err != nil {
			writeError(w, http.StatusInternalServerError, "update egress override failed")
			return
		}
		a.Audit.Append(r.Context(), principal(r.Context()), "egress-override.changed", "session", sessionID, "allowed", map[string]any{"mode": mode, "allowedHosts": req.AllowedHosts})
		writeJSON(w, http.StatusOK, map[string]any{"updated": true})
	}))
	h.ServeHTTP(w, r)
}

func (a *API) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h := requireScopes([]string{"audit:read"}, a.Audit, func(_ *http.Request) (string, string, string) {
		return "audit.list", "audit", "*"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				writeError(w, http.StatusBadRequest, "invalid limit")
				return
			}
			limit = n
		}
		events, err := a.Audit.List(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list audit failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	}))
	h.ServeHTTP(w, r)
}

func validateAPI(a *API) error {
	if a.K8s == nil {
		return errors.New("k8s client required")
	}
	if a.Auth == nil {
		return errors.New("authenticator required")
	}
	if a.Audit == nil {
		return errors.New("audit store required")
	}
	if strings.TrimSpace(a.Namespace) == "" {
		return errors.New("namespace required")
	}
	return nil
}

func New(namespace, auditPath, bootstrapToken string, restCfg *rest.Config, k8s client.Client) (*API, error) {
	tokens := newTokenStore()
	if err := tokens.EnsureBootstrapToken(context.Background(), bootstrapToken); err != nil {
		return nil, err
	}
	api := &API{
		Namespace: namespace,
		K8s:       k8s,
		Auth:      newAuthenticator(tokens),
		Tokens:    tokens,
		Audit:     newAuditStore(auditPath),
	}
	if restCfg != nil {
		api.Attach = newAttachService(namespace, restCfg, k8s, tokens, api.Audit)
	}
	if err := validateAPI(api); err != nil {
		return nil, err
	}
	return api, nil
}
