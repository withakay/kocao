package controllers

import (
	"context"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFakeClient_CanCreateAndGetHarnessRun(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := operatorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}).Build()

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-x", Namespace: "default"},
		Spec:       operatorv1alpha1.HarnessRunSpec{RepoURL: "https://example.com/repo", Image: "busybox"},
	}
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
}

func TestFakeClient_CanStatusUpdateHarnessRun(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := operatorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}).Build()

	run := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-status", Namespace: "default"},
		Spec:       operatorv1alpha1.HarnessRunSpec{RepoURL: "https://example.com/repo", Image: "busybox"},
	}
	if err := cl.Create(context.Background(), run); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	got.Status.PodName = "pod-x"
	if err := cl.Status().Update(context.Background(), &got); err != nil {
		t.Fatalf("status update: %v", err)
	}

	var got2 operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(run), &got2); err != nil {
		t.Fatalf("get2: %v", err)
	}
	if got2.Status.PodName != "pod-x" {
		t.Fatalf("expected status updated, got %q", got2.Status.PodName)
	}
}

func TestFakeClient_PatchMetaThenStatusUpdateHarnessRun(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := operatorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.HarnessRun{}).Build()

	orig := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-patch-status", Namespace: "default"},
		Spec:       operatorv1alpha1.HarnessRunSpec{RepoURL: "https://example.com/repo", Image: "busybox"},
	}
	if err := cl.Create(context.Background(), orig); err != nil {
		t.Fatalf("create: %v", err)
	}

	var base operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(orig), &base); err != nil {
		t.Fatalf("get base: %v", err)
	}

	updated := base.DeepCopy()
	updated.Finalizers = append(updated.Finalizers, FinalizerName)
	if err := cl.Patch(context.Background(), updated, client.MergeFrom(&base)); err != nil {
		t.Fatalf("patch: %v", err)
	}

	var latest operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(orig), &latest); err != nil {
		t.Fatalf("get latest: %v", err)
	}
	latest.Status.PodName = "pod-x"
	if err := cl.Status().Update(context.Background(), &latest); err != nil {
		t.Fatalf("status update: %v", err)
	}

	var got operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(orig), &got); err != nil {
		t.Fatalf("get got: %v", err)
	}
	if got.Status.PodName != "pod-x" {
		t.Fatalf("expected status updated, got %q", got.Status.PodName)
	}
}
