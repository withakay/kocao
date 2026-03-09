package controlplaneapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type symphonyProjectRequest struct {
	Name string                               `json:"name,omitempty"`
	Spec operatorv1alpha1.SymphonyProjectSpec `json:"spec"`
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
	project := &operatorv1alpha1.SymphonyProject{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "SymphonyProject"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: a.Namespace},
		Spec:       req.Spec,
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
	updated.Spec = req.Spec
	updated.Spec.Paused = req.Spec.Paused
	updated.ApplyDefaults()
	if err := updated.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.K8s.Patch(r.Context(), updated, client.MergeFrom(project)); err != nil {
		writeError(w, http.StatusInternalServerError, "update symphony project failed")
		return
	}
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(updated))
}

func (a *API) handleSymphonyProjectPause(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.setSymphonyProjectPaused(r, name, true)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "pause symphony project failed")
		return
	}
	writeJSON(w, http.StatusOK, symphonyProjectToResponse(project))
}

func (a *API) handleSymphonyProjectResume(w http.ResponseWriter, r *http.Request, name string) {
	project, err := a.setSymphonyProjectPaused(r, name, false)
	if err != nil {
		a.writeSymphonyProjectError(w, err, "resume symphony project failed")
		return
	}
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
