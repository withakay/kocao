package controlplaneapi

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type clusterOverviewResponse struct {
	Namespace   string                    `json:"namespace"`
	CollectedAt time.Time                 `json:"collectedAt"`
	Summary     clusterSummary            `json:"summary"`
	Deployments []clusterDeploymentStatus `json:"deployments"`
	Pods        []clusterPodStatus        `json:"pods"`
	Config      clusterConfigSnapshot     `json:"config"`
}

type clusterSummary struct {
	SessionCount    int `json:"sessionCount"`
	HarnessRunCount int `json:"harnessRunCount"`
	PodCount        int `json:"podCount"`
	RunningPods     int `json:"runningPods"`
	PendingPods     int `json:"pendingPods"`
	FailedPods      int `json:"failedPods"`
}

type clusterDeploymentStatus struct {
	Name              string `json:"name"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	DesiredReplicas   int32  `json:"desiredReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	Unavailable       int32  `json:"unavailableReplicas"`
}

type clusterPodStatus struct {
	Name       string `json:"name"`
	Phase      string `json:"phase"`
	Ready      string `json:"ready"`
	Restarts   int32  `json:"restarts"`
	NodeName   string `json:"nodeName,omitempty"`
	AgeSeconds int64  `json:"ageSeconds"`
}

type clusterConfigSnapshot struct {
	Environment            string `json:"environment,omitempty"`
	AuditPathConfigured    bool   `json:"auditPathConfigured"`
	BootstrapTokenDetected bool   `json:"bootstrapTokenDetected"`
	GitHubCIDRsConfigured  bool   `json:"gitHubCIDRsConfigured"`
}

type podLogsResponse struct {
	PodName   string `json:"podName"`
	Container string `json:"container,omitempty"`
	TailLines int64  `json:"tailLines"`
	Logs      string `json:"logs"`
}

func (a *API) handleClusterOverview(w http.ResponseWriter, r *http.Request) {
	var sessions operatorv1alpha1.SessionList
	if err := a.K8s.List(r.Context(), &sessions, client.InNamespace(a.Namespace)); err != nil {
		writeError(w, http.StatusInternalServerError, "list sessions failed")
		return
	}

	var runs operatorv1alpha1.HarnessRunList
	if err := a.K8s.List(r.Context(), &runs, client.InNamespace(a.Namespace)); err != nil {
		writeError(w, http.StatusInternalServerError, "list harness runs failed")
		return
	}

	var pods corev1.PodList
	if err := a.K8s.List(r.Context(), &pods, client.InNamespace(a.Namespace)); err != nil {
		writeError(w, http.StatusInternalServerError, "list pods failed")
		return
	}

	var deployments appsv1.DeploymentList
	if err := a.K8s.List(r.Context(), &deployments, client.InNamespace(a.Namespace)); err != nil {
		writeError(w, http.StatusInternalServerError, "list deployments failed")
		return
	}

	config := clusterConfigSnapshot{}
	var cfg corev1.ConfigMap
	if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: "control-plane-config"}, &cfg); err == nil {
		config.Environment = strings.TrimSpace(cfg.Data["CP_ENV"])
		config.AuditPathConfigured = strings.TrimSpace(cfg.Data["CP_AUDIT_PATH"]) != ""
		config.BootstrapTokenDetected = strings.TrimSpace(cfg.Data["CP_BOOTSTRAP_TOKEN"]) != ""
		config.GitHubCIDRsConfigured = strings.TrimSpace(cfg.Data["CP_GITHUB_EGRESS_CIDRS"]) != ""
	} else if !apierrors.IsNotFound(err) {
		writeError(w, http.StatusInternalServerError, "read control-plane config failed")
		return
	}

	now := time.Now()
	outPods := make([]clusterPodStatus, 0, len(pods.Items))
	summary := clusterSummary{
		SessionCount:    len(sessions.Items),
		HarnessRunCount: len(runs.Items),
		PodCount:        len(pods.Items),
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		restarts := int32(0)
		ready := int32(0)
		total := int32(len(p.Status.ContainerStatuses))
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
			if cs.Ready {
				ready++
			}
		}
		if total == 0 {
			total = int32(len(p.Spec.Containers))
		}
		ageSeconds := int64(0)
		if !p.CreationTimestamp.IsZero() {
			ageSeconds = int64(now.Sub(p.CreationTimestamp.Time).Seconds())
			if ageSeconds < 0 {
				ageSeconds = 0
			}
		}

		switch p.Status.Phase {
		case corev1.PodRunning:
			summary.RunningPods++
		case corev1.PodPending:
			summary.PendingPods++
		case corev1.PodFailed:
			summary.FailedPods++
		}

		outPods = append(outPods, clusterPodStatus{
			Name:       p.Name,
			Phase:      string(p.Status.Phase),
			Ready:      strconv.FormatInt(int64(ready), 10) + "/" + strconv.FormatInt(int64(total), 10),
			Restarts:   restarts,
			NodeName:   p.Spec.NodeName,
			AgeSeconds: ageSeconds,
		})
	}
	sort.Slice(outPods, func(i, j int) bool { return outPods[i].Name < outPods[j].Name })

	outDeployments := make([]clusterDeploymentStatus, 0, len(deployments.Items))
	for i := range deployments.Items {
		d := &deployments.Items[i]
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		outDeployments = append(outDeployments, clusterDeploymentStatus{
			Name:              d.Name,
			ReadyReplicas:     d.Status.ReadyReplicas,
			AvailableReplicas: d.Status.AvailableReplicas,
			DesiredReplicas:   desired,
			UpdatedReplicas:   d.Status.UpdatedReplicas,
			Unavailable:       d.Status.UnavailableReplicas,
		})
	}
	sort.Slice(outDeployments, func(i, j int) bool { return outDeployments[i].Name < outDeployments[j].Name })

	writeJSON(w, http.StatusOK, clusterOverviewResponse{
		Namespace:   a.Namespace,
		CollectedAt: metav1.Now().Time,
		Summary:     summary,
		Deployments: outDeployments,
		Pods:        outPods,
		Config:      config,
	})
}

func (a *API) handlePodLogs(w http.ResponseWriter, r *http.Request, podName string) {
	podName = strings.TrimSpace(podName)
	if podName == "" || strings.Contains(podName, "/") {
		writeError(w, http.StatusBadRequest, "invalid pod name")
		return
	}
	if a.Clientset == nil {
		writeError(w, http.StatusServiceUnavailable, "pod logs unavailable")
		return
	}

	tailLines := int64(200)
	if raw := strings.TrimSpace(r.URL.Query().Get("tailLines")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid tailLines")
			return
		}
		if n > 2000 {
			n = 2000
		}
		tailLines = n
	}

	container := strings.TrimSpace(r.URL.Query().Get("container"))
	if strings.Contains(container, "/") {
		writeError(w, http.StatusBadRequest, "invalid container")
		return
	}

	var pod corev1.Pod
	if err := a.K8s.Get(r.Context(), client.ObjectKey{Namespace: a.Namespace, Name: podName}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "pod not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get pod failed")
		return
	}

	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}

	logReq := a.Clientset.CoreV1().Pods(a.Namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  container,
		TailLines:  &tailLines,
		Timestamps: true,
	})
	stream, err := logReq.Stream(r.Context())
	if err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "pod logs not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "read pod logs failed")
		return
	}
	defer func() { _ = stream.Close() }()

	b, err := io.ReadAll(io.LimitReader(stream, 2*1024*1024))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read pod logs failed")
		return
	}

	writeJSON(w, http.StatusOK, podLogsResponse{
		PodName:   podName,
		Container: container,
		TailLines: tailLines,
		Logs:      string(b),
	})
}
