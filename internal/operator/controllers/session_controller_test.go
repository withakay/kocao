package controllers

import (
	"context"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSessionReconcile_ActiveAddsFinalizerAndStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := operatorv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add core scheme: %v", err)
	}

	sess := &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "s1",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://example.com/repo"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.Session{}).Build()
	if err := cl.Create(context.Background(), sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	r := &SessionReconciler{Client: cl, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sess)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.Session
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(sess), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Finalizers) != 1 || got.Finalizers[0] != FinalizerName {
		t.Fatalf("expected finalizer %q, got %v", FinalizerName, got.Finalizers)
	}
	if got.Status.Phase != operatorv1alpha1.SessionPhaseActive {
		t.Fatalf("expected Active, got %q", got.Status.Phase)
	}
}

func TestSessionReconcile_BackfillsDisplayName(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	sess := &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "s-noname",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://example.com/repo"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.Session{}).Build()
	if err := cl.Create(context.Background(), sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	r := &SessionReconciler{Client: cl, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sess)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.Session
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(sess), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Spec.DisplayName == "" {
		t.Fatal("expected display name to be backfilled, got empty")
	}
	// Format: adjective-noun
	if len(got.Spec.DisplayName) < 3 {
		t.Fatalf("display name too short: %q", got.Spec.DisplayName)
	}
}

func TestSessionReconcile_PreservesExistingDisplayName(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	sess := &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "s-named",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.SessionSpec{
			DisplayName: "elegant-galileo",
			RepoURL:     "https://example.com/repo",
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.Session{}).Build()
	if err := cl.Create(context.Background(), sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	r := &SessionReconciler{Client: cl, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sess)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.Session
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(sess), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Spec.DisplayName != "elegant-galileo" {
		t.Fatalf("expected display name to be preserved as %q, got %q", "elegant-galileo", got.Spec.DisplayName)
	}
}

func TestSessionReconcile_CreatesWorkspacePVC(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	sess := &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "s-pvc",
			Namespace: "default",
		},
		Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://example.com/repo"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.Session{}).Build()
	if err := cl.Create(context.Background(), sess); err != nil {
		t.Fatalf("create session: %v", err)
	}

	r := &SessionReconciler{Client: cl, Scheme: scheme}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(sess)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: sessionWorkspacePVCName(sess.Name)}, &pvc); err != nil {
		t.Fatalf("get pvc: %v", err)
	}
}
