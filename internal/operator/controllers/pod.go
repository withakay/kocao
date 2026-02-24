package controllers

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildHarnessPod(run *operatorv1alpha1.HarnessRun, workspacePVCName string) *corev1.Pod {
	const (
		workspaceVolumeName = "workspace"
		workspaceMountPath  = "/workspace"
		gitAuthVolumeName   = "git-auth"
		gitAuthMountPath    = "/var/run/secrets/kocao/git"
	)

	// Hardened defaults: run as non-root with a restrictive security context.
	// Keep IDs in sync with build/Dockerfile.harness.
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	uid := int64(10001)
	gid := int64(10001)
	seccompProfile := corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}

	labels := map[string]string{
		"app.kubernetes.io/name":        "kocao-harness",
		"app.kubernetes.io/managed-by":  "kocao-control-plane-operator",
		"kocao.withakay.github.com/run": run.Name,
	}
	if run.Spec.WorkspaceSessionName != "" {
		labels[LabelWorkspaceSessionName] = run.Spec.WorkspaceSessionName
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

	env := make([]corev1.EnvVar, 0, len(run.Spec.Env)+6)
	env = append(env, corev1.EnvVar{Name: "KOCAO_REPO_URL", Value: run.Spec.RepoURL})
	if run.Spec.RepoRevision != "" {
		env = append(env, corev1.EnvVar{Name: "KOCAO_REPO_REVISION", Value: run.Spec.RepoRevision})
	}
	env = append(env,
		corev1.EnvVar{Name: "KOCAO_WORKSPACE_DIR", Value: workspaceMountPath},
		corev1.EnvVar{Name: "KOCAO_REPO_DIR", Value: workspaceMountPath + "/repo"},
		corev1.EnvVar{Name: "GIT_TERMINAL_PROMPT", Value: "0"},
	)
	for _, e := range run.Spec.Env {
		if strings.HasPrefix(strings.TrimSpace(e.Name), "KOCAO_") {
			// Reserved for operator/harness contract; do not allow user overrides.
			continue
		}
		env = append(env, corev1.EnvVar{Name: e.Name, Value: e.Value})
	}

	workspaceVolumeSource := corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}
	if strings.TrimSpace(workspacePVCName) != "" {
		workspaceVolumeSource = corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: workspacePVCName}}
	}
	volumes := []corev1.Volume{{Name: workspaceVolumeName, VolumeSource: workspaceVolumeSource}}
	volumeMounts := []corev1.VolumeMount{{Name: workspaceVolumeName, MountPath: workspaceMountPath}}

	if run.Spec.GitAuth != nil && strings.TrimSpace(run.Spec.GitAuth.SecretName) != "" {
		tokenKey := strings.TrimSpace(run.Spec.GitAuth.TokenKey)
		if tokenKey == "" {
			tokenKey = "token"
		}
		items := []corev1.KeyToPath{{Key: tokenKey, Path: "token"}}
		if uk := strings.TrimSpace(run.Spec.GitAuth.UsernameKey); uk != "" {
			items = append(items, corev1.KeyToPath{Key: uk, Path: "username"})
		}
		volumes = append(volumes, corev1.Volume{
			Name: gitAuthVolumeName,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName: run.Spec.GitAuth.SecretName,
				Items:      items,
			}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: gitAuthVolumeName, MountPath: gitAuthMountPath, ReadOnly: true})
		env = append(env,
			corev1.EnvVar{Name: "GIT_ASKPASS", Value: "/usr/local/bin/kocao-git-askpass"},
			corev1.EnvVar{Name: "KOCAO_GIT_TOKEN_FILE", Value: gitAuthMountPath + "/token"},
		)
		if strings.TrimSpace(run.Spec.GitAuth.UsernameKey) != "" {
			env = append(env, corev1.EnvVar{Name: "KOCAO_GIT_USERNAME_FILE", Value: gitAuthMountPath + "/username"})
		}
	}

	container := corev1.Container{
		Name:         "harness",
		Image:        run.Spec.Image,
		Command:      run.Spec.Command,
		Args:         run.Spec.Args,
		WorkingDir:   run.Spec.WorkingDir,
		Env:          env,
		VolumeMounts: volumeMounts,
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             &runAsNonRoot,
			RunAsUser:                &uid,
			RunAsGroup:               &gid,
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			SeccompProfile:           &seccompProfile,
		},
	}
	if container.WorkingDir == "" {
		container.WorkingDir = workspaceMountPath
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: run.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot:   &runAsNonRoot,
				FSGroup:        &gid,
				SeccompProfile: &seccompProfile,
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{container},
			Volumes:       volumes,
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
