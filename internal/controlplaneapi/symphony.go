package controlplaneapi

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var symphonySecretNameCleaner = regexp.MustCompile(`[^a-z0-9-]+`)

type symphonyProjectSourceRequest struct {
	Project         operatorv1alpha1.GitHubProjectRef `json:"project"`
	TokenSecretRef  operatorv1alpha1.SecretKeyRef     `json:"tokenSecretRef"`
	GitHubToken     string                            `json:"githubToken,omitempty"`
	ActiveStates    []string                          `json:"activeStates,omitempty"`
	TerminalStates  []string                          `json:"terminalStates,omitempty"`
	FieldName       string                            `json:"fieldName,omitempty"`
	PollIntervalSec int32                             `json:"pollIntervalSeconds,omitempty"`
}

type symphonyProjectSpecRequest struct {
	Paused       bool                                             `json:"paused,omitempty"`
	Source       symphonyProjectSourceRequest                     `json:"source"`
	Repositories []operatorv1alpha1.SymphonyProjectRepositorySpec `json:"repositories"`
	Runtime      operatorv1alpha1.SymphonyProjectRuntimeSpec      `json:"runtime"`
}

type symphonyProjectRequest struct {
	Name string                     `json:"name,omitempty"`
	Spec symphonyProjectSpecRequest `json:"spec"`
}

type symphonyProjectResponse struct {
	Name       string                                 `json:"name"`
	Namespace  string                                 `json:"namespace,omitempty"`
	CreatedAt  string                                 `json:"createdAt,omitempty"`
	Generation int64                                  `json:"generation,omitempty"`
	Paused     bool                                   `json:"paused"`
	Spec       operatorv1alpha1.SymphonyProjectSpec   `json:"spec"`
	Status     operatorv1alpha1.SymphonyProjectStatus `json:"status"`
}

func symphonyProjectToResponse(project *operatorv1alpha1.SymphonyProject) symphonyProjectResponse {
	createdAt := ""
	if !project.CreationTimestamp.IsZero() {
		createdAt = project.CreationTimestamp.UTC().Format(time.RFC3339)
	}
	return symphonyProjectResponse{
		Name:       project.Name,
		Namespace:  project.Namespace,
		CreatedAt:  createdAt,
		Generation: project.Generation,
		Paused:     project.Spec.Paused,
		Spec:       project.Spec,
		Status:     project.Status,
	}
}

func (a *API) handleSymphonyProjectsList(w http.ResponseWriter, r *http.Request) {
	var list operatorv1alpha1.SymphonyProjectList
	if err := a.K8s.List(r.Context(), &list, client.InNamespace(a.Namespace)); err != nil {
		writeError(w, http.StatusInternalServerError, "list symphony projects failed")
		return
	}
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Name < list.Items[j].Name
	})
	out := make([]symphonyProjectResponse, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, symphonyProjectToResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"symphonyProjects": out})
}

func (a *API) handleSymphonyProjectsCreate(w http.ResponseWriter, r *http.Request) {
	var req symphonyProjectRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	spec := req.Spec.toSpec()
	if err := a.prepareSymphonySourceSecret(r.Context(), name, &req.Spec.Source, &spec.Source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	project := &operatorv1alpha1.SymphonyProject{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "SymphonyProject"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: a.Namespace},
		Spec:       spec,
	}
	project.ApplyDefaults()
	if err := project.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.K8s.Create(r.Context(), project); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, "symphony project already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create symphony project failed")
		return
	}
	appendSymphonyAudit(r.Context(), a.Audit, "api", "symphony.create", project.Name, "allowed", map[string]any{"repositoryCount": len(project.Spec.Repositories), "project": map[string]any{"owner": project.Spec.Source.Project.Owner, "number": project.Spec.Source.Project.Number}})
	writeJSON(w, http.StatusCreated, symphonyProjectToResponse(project))
}

func (a *API) handleSymphonyProjectGet(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.getSymphonyProject(r.Context(), name)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "get symphony project failed")
		return
	}
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(project))
}

func (a *API) handleSymphonyProjectPatch(w http.ResponseWriter, r *http.Request, name string) {
	var req symphonyProjectRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	project, err := a.getSymphonyProject(r.Context(), name)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "get symphony project failed")
		return
	}
	updated := project.DeepCopy()
	updated.Spec = req.Spec.toSpec()
	if err := a.prepareSymphonySourceSecret(r.Context(), updated.Name, &req.Spec.Source, &updated.Spec.Source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated.ApplyDefaults()
	if err := updated.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(project)); err != nil {
		writeError(w, http.StatusInternalServerError, "update symphony project failed")
		return
	}
	appendSymphonyAudit(r.Context(), a.Audit, "api", "symphony.update", updated.Name, "allowed", map[string]any{"paused": updated.Spec.Paused, "repositoryCount": len(updated.Spec.Repositories)})
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(updated))
}

func (req symphonyProjectSpecRequest) toSpec() operatorv1alpha1.SymphonyProjectSpec {
	return operatorv1alpha1.SymphonyProjectSpec{
		Paused: req.Paused,
		Source: operatorv1alpha1.SymphonyProjectSourceSpec{
			Project:         req.Source.Project,
			TokenSecretRef:  req.Source.TokenSecretRef,
			ActiveStates:    append([]string(nil), req.Source.ActiveStates...),
			TerminalStates:  append([]string(nil), req.Source.TerminalStates...),
			FieldName:       req.Source.FieldName,
			PollIntervalSec: req.Source.PollIntervalSec,
		},
		Repositories: append([]operatorv1alpha1.SymphonyProjectRepositorySpec(nil), req.Repositories...),
		Runtime:      req.Runtime,
	}
}

func (a *API) prepareSymphonySourceSecret(ctx context.Context, projectName string, req *symphonyProjectSourceRequest, spec *operatorv1alpha1.SymphonyProjectSourceSpec) error {
	if req == nil || spec == nil {
		return nil
	}
	secretKey := strings.TrimSpace(req.TokenSecretRef.Key)
	if secretKey == "" {
		secretKey = "token"
	}
	githubToken := strings.TrimSpace(req.GitHubToken)
	if githubToken != "" {
		secretName := deriveSymphonySecretName(projectName, req.Project.Owner)
		if err := a.upsertSymphonyTokenSecret(ctx, secretName, secretKey, githubToken, projectName, req.Project.Owner); err != nil {
			return fmt.Errorf("create symphony github token secret failed")
		}
		spec.TokenSecretRef = operatorv1alpha1.SecretKeyRef{Name: secretName, Key: secretKey}
		return nil
	}
	if looksLikeGitHubPAT(req.TokenSecretRef.Name) {
		return fmt.Errorf("spec.source.tokenSecretRef.name must reference a Kubernetes Secret name; use spec.source.githubToken for raw PAT input")
	}
	if spec.TokenSecretRef.Key == "" {
		spec.TokenSecretRef.Key = secretKey
	}
	return nil
}

func (a *API) upsertSymphonyTokenSecret(ctx context.Context, secretName, secretKey, token, projectName, owner string) error {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: a.Namespace, Name: secretName}
	err := a.K8s.Get(ctx, key, secret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	labels := map[string]string{
		"kocao.withakay.github.com/managed-by":       "control-plane-api",
		"kocao.withakay.github.com/symphony-project": projectName,
		"kocao.withakay.github.com/github-owner":     strings.TrimSpace(owner),
	}
	if apierrors.IsNotFound(err) {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: a.Namespace, Labels: labels},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{secretKey: []byte(token)},
		}
		return a.K8s.Create(ctx, secret)
	}
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	for k, v := range labels {
		secret.Labels[k] = v
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[secretKey] = []byte(token)
	secret.Type = corev1.SecretTypeOpaque
	return a.K8s.Update(ctx, secret)
}

func deriveSymphonySecretName(projectName, owner string) string {
	parts := []string{"symphony", sanitizeSecretNamePart(projectName), sanitizeSecretNamePart(owner), "token"}
	joined := strings.Join(parts, "-")
	joined = strings.Trim(joined, "-")
	if joined == "" {
		joined = "symphony-token"
	}
	if len(joined) > 63 {
		joined = strings.Trim(joined[:63], "-")
	}
	if joined == "" {
		return "symphony-token"
	}
	return joined
}

func sanitizeSecretNamePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = symphonySecretNameCleaner.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return value
}

func looksLikeGitHubPAT(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	return strings.HasPrefix(trimmed, "github_pat_") || strings.HasPrefix(trimmed, "ghp_") || strings.HasPrefix(trimmed, "gho_") || strings.HasPrefix(trimmed, "ghu_") || strings.HasPrefix(trimmed, "ghs_") || strings.HasPrefix(trimmed, "ghr_")
}

func (a *API) handleSymphonyProjectPause(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.setSymphonyProjectPaused(r, name, true)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "pause symphony project failed")
		return
	}
	appendSymphonyAudit(r.Context(), a.Audit, "api", "symphony.pause", project.Name, "allowed", map[string]any{"paused": true})
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(project))
}

func (a *API) handleSymphonyProjectResume(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.setSymphonyProjectPaused(r, name, false)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "resume symphony project failed")
		return
	}
	appendSymphonyAudit(r.Context(), a.Audit, "api", "symphony.resume", project.Name, "allowed", map[string]any{"paused": false})
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(project))
}

func (a *API) handleSymphonyProjectRefresh(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.getSymphonyProject(r.Context(), name)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "get symphony project failed")
		return
	}
	updated := project.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[annotationSymphonyRefreshRequestedAt] = time.Now().UTC().Format(time.RFC3339Nano)
	if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(project)); err != nil {
		writeError(w, http.StatusInternalServerError, "refresh symphony project failed")
		return
	}
	appendSymphonyAudit(r.Context(), a.Audit, "api", "symphony.refresh", updated.Name, "allowed", map[string]any{"requestedAt": updated.Annotations[annotationSymphonyRefreshRequestedAt]})
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(updated))
}

func (a *API) getSymphonyProject(ctx context.Context, name string) (*operatorv1alpha1.SymphonyProject, error) {
	project := &operatorv1alpha1.SymphonyProject{}
	err := a.K8s.Get(ctx, client.ObjectKey{Namespace: a.Namespace, Name: strings.TrimSpace(name)}, project)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func (a *API) setSymphonyProjectPaused(r *http.Request, name string, paused bool) (*operatorv1alpha1.SymphonyProject, error) {
	project, err := a.getSymphonyProject(r.Context(), name)
	if err != nil {
		return nil, err
	}
	updated := project.DeepCopy()
	updated.Spec.Paused = paused
	if !paused {
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		updated.Annotations[annotationSymphonyRefreshRequestedAt] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(project)); err != nil {
		return nil, err
	}
	return updated, nil
}

func (a *API) writeSymphonyProjectError(w http.ResponseWriter, err error, message string) {
	if apierrors.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "symphony project not found")
		return
	}
	writeError(w, http.StatusInternalServerError, message)
}
