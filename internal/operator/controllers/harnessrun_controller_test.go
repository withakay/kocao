package controllers

import (
	"context"
	"testing"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clocktesting "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHarnessRunReconcile_CreatesPodAndInitializesStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}
	if err := operatorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add operator scheme: %v", err)
	}

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: operatorv1alpha1.HarnessRunSpec{
			RepoURL: "https://github.com/withakay/kocao",
			Image:   "busybox:latest",
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}, &corev1.Pod{}).Build()
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	{
		var got operatorv1alpha1.HarnessRun
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got); err != nil {
			t.Fatalf("precondition get run: %v", err)
		}
	}
	clk := clocktesting.NewFakeClock(time.Unix(1, 0))
	r := &HarnessRunReconciler{Client: cl, Scheme: scheme, Clock: clk}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var podList corev1.PodList
	if err := cl.List(context.Background(), &podList, client.InNamespace("default")); err != nil {
		t.Fatalf("list pods: %v", err)
	}
	if len(podList.Items) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(podList.Items))
	}
	pod := podList.Items[0]
	if pod.Spec.Containers[0].Image != "busybox:latest" {
		t.Fatalf("expected pod image busybox:latest, got %q", pod.Spec.Containers[0].Image)
	}
	// Pod contract: always mounts a workspace volume at /workspace.
	volOK := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == "workspace" {
			volOK = v.EmptyDir != nil
			break
		}
	}
	if !volOK {
		t.Fatalf("expected workspace emptyDir volume, got volumes=%#v", pod.Spec.Volumes)
	}
	mountOK := false
	for _, m := range pod.Spec.Containers[0].VolumeMounts {
		if m.Name == "workspace" && m.MountPath == "/workspace" {
			mountOK = true
			break
		}
	}
	if !mountOK {
		t.Fatalf("expected workspace mount at /workspace, got mounts=%#v", pod.Spec.Containers[0].VolumeMounts)
	}

	env := map[string]string{}
	for _, ev := range pod.Spec.Containers[0].Env {
		env[ev.Name] = ev.Value
	}
	if env["KOCAO_WORKSPACE_DIR"] != "/workspace" {
		t.Fatalf("expected KOCAO_WORKSPACE_DIR=/workspace, got %q", env["KOCAO_WORKSPACE_DIR"])
	}
	if env["KOCAO_REPO_DIR"] == "" {
		t.Fatalf("expected KOCAO_REPO_DIR to be set")
	}

	var updated operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status.PodName == "" {
		podName := ""
		if len(podList.Items) != 0 {
			podName = podList.Items[0].Name
		}
		t.Fatalf("expected status.podName to be set (podList[0].name=%q status=%#v)", podName, updated.Status)
	}
	if updated.Status.Phase != operatorv1alpha1.HarnessRunPhaseStarting {
		t.Fatalf("expected phase Starting, got %q", updated.Status.Phase)
	}
}

func TestHarnessRunReconcile_GitAuthAddsSecretVolumeAndAskpassEnv(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = operatorv1alpha1.AddToScheme(scheme)

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-auth", Namespace: "default"},
		Spec: operatorv1alpha1.HarnessRunSpec{
			RepoURL: "https://example.com/repo",
			Image:   "busybox",
			GitAuth: &operatorv1alpha1.GitAuthSpec{SecretName: "repo-creds", TokenKey: "token", UsernameKey: "username"},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}, &corev1.Pod{}).Build()
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	r := &HarnessRunReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(1, 0))}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var updated operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated); err != nil {
		t.Fatalf("get run: %v", err)
	}
	var pod corev1.Pod
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: updated.Status.PodName}, &pod); err != nil {
		t.Fatalf("get pod: %v", err)
	}

	secretVol := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == "git-auth" && v.Secret != nil && v.Secret.SecretName == "repo-creds" {
			secretVol = true
			break
		}
	}
	if !secretVol {
		t.Fatalf("expected git-auth secret volume, got volumes=%#v", pod.Spec.Volumes)
	}

	env := map[string]string{}
	for _, ev := range pod.Spec.Containers[0].Env {
		env[ev.Name] = ev.Value
	}
	if env["GIT_ASKPASS"] != "/usr/local/bin/kocao-git-askpass" {
		t.Fatalf("expected GIT_ASKPASS to be set, got %q", env["GIT_ASKPASS"])
	}
	if env["KOCAO_GIT_TOKEN_FILE"] != "/var/run/secrets/kocao/git/token" {
		t.Fatalf("expected KOCAO_GIT_TOKEN_FILE to be set, got %q", env["KOCAO_GIT_TOKEN_FILE"])
	}
	if env["KOCAO_GIT_USERNAME_FILE"] != "/var/run/secrets/kocao/git/username" {
		t.Fatalf("expected KOCAO_GIT_USERNAME_FILE to be set, got %q", env["KOCAO_GIT_USERNAME_FILE"])
	}
}

func TestHarnessRunReconcile_MapsPodRunningToPhaseRunning(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = operatorv1alpha1.AddToScheme(scheme)

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-2", Namespace: "default"},
		Spec:       operatorv1alpha1.HarnessRunSpec{RepoURL: "https://example.com/repo", Image: "busybox"},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}, &corev1.Pod{}).Build()
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	{
		var got operatorv1alpha1.HarnessRun
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got); err != nil {
			t.Fatalf("precondition get run: %v", err)
		}
	}
	r := &HarnessRunReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(10, 0))}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile 1: %v", err)
	}

	var updated operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status.PodName == "" {
		t.Fatalf("expected podName")
	}

	var pod corev1.Pod
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: updated.Status.PodName}, &pod); err != nil {
		t.Fatalf("get pod: %v", err)
	}
	pod.Status.Phase = corev1.PodRunning
	pod.Status.StartTime = &metav1.Time{Time: time.Unix(11, 0)}
	if err := cl.Status().Update(context.Background(), &pod); err != nil {
		t.Fatalf("update pod status: %v", err)
	}

	_, err = r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile 2: %v", err)
	}

	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated); err != nil {
		t.Fatalf("get run 2: %v", err)
	}
	if updated.Status.Phase != operatorv1alpha1.HarnessRunPhaseRunning {
		t.Fatalf("expected Running, got %q", updated.Status.Phase)
	}
	if updated.Status.StartTime == nil {
		t.Fatalf("expected startTime to be set")
	}
}

func TestHarnessRunReconcile_TTLDeletesAfterCompletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = operatorv1alpha1.AddToScheme(scheme)

	ttl := int32(1)
	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-3", Namespace: "default"},
		Spec:       operatorv1alpha1.HarnessRunSpec{RepoURL: "https://example.com/repo", Image: "busybox", TTLSecondsAfterFinished: &ttl},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}, &corev1.Pod{}).Build()
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	{
		var got operatorv1alpha1.HarnessRun
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got); err != nil {
			t.Fatalf("precondition get run: %v", err)
		}
	}
	clk := clocktesting.NewFakeClock(time.Unix(100, 0))
	r := &HarnessRunReconciler{Client: cl, Scheme: scheme, Clock: clk}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile 1: %v", err)
	}

	var updated operatorv1alpha1.HarnessRun
	_ = cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated)
	var pod corev1.Pod
	_ = cl.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: updated.Status.PodName}, &pod)
	pod.Status.Phase = corev1.PodSucceeded
	if err := cl.Status().Update(context.Background(), &pod); err != nil {
		t.Fatalf("update pod status: %v", err)
	}

	_, err = r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile 2: %v", err)
	}

	clk.SetTime(time.Unix(102, 0))
	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("reconcile 3: %v", err)
	}
	if res.RequeueAfter == 0 {
		// ok: either deleted or no requeue needed
	}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(run), &updated)
	if err == nil {
		if updated.DeletionTimestamp == nil {
			t.Fatalf("expected harnessrun to be deleting or gone after TTL")
		}
		return
	}
}
