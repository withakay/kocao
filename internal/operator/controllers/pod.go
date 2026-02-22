package controllers

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildHarnessPod(run *operatorv1alpha1.HarnessRun) *corev1.Pod {
	labels := map[string]string{
		"app.kubernetes.io/name":        "kocao-harness",
		"app.kubernetes.io/managed-by":  "kocao-control-plane-operator",
		"kocao.withakay.github.com/run": run.Name,
	}
	if run.Spec.SessionName != "" {
		labels[LabelSessionName] = run.Spec.SessionName
	}

	namePrefix := sanitizeDNSLabel(run.Name)
	name := namePrefix
	if len(name) > 59 {
		name = name[:59]
		name = strings.Trim(name, "-")
		if name == "" {
			name = "run"
		}
	}
	name += "-pod"

	env := make([]corev1.EnvVar, 0, len(run.Spec.Env)+2)
	env = append(env, corev1.EnvVar{Name: "KOCAO_REPO_URL", Value: run.Spec.RepoURL})
	if run.Spec.RepoRevision != "" {
		env = append(env, corev1.EnvVar{Name: "KOCAO_REPO_REVISION", Value: run.Spec.RepoRevision})
	}
	for _, e := range run.Spec.Env {
		env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
	}

	container := corev1.Container{
		Name:       "harness",
		Image:      run.Spec.Image,
		Command:    run.Spec.Command,
		Args:       run.Spec.Args,
		WorkingDir: run.Spec.WorkingDir,
		Env:        env,
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: run.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{container},
		},
	}
}

func sanitizeDNSLabel(s string) string {
	// Kubernetes object names must be valid DNS labels. Keep this lightweight
	// since it is only used as a GenerateName prefix.
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "run"
	}
	b := strings.Builder{}
	b.Grow(len(s))
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "run"
	}
	if len(out) > 50 {
		out = out[:50]
		out = strings.Trim(out, "-")
	}
	return out
}

func invalidSpecError(field string) error {
	return fmt.Errorf("invalid spec: %s", field)
}
