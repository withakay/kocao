package controllers

import (
	"context"
	"os"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	envSessionStorageSize  = "CP_SESSION_STORAGE_SIZE"
	envSessionStorageClass = "CP_SESSION_STORAGE_CLASS"
)

func sessionWorkspacePVCName(sessionName string) string {
	base := sanitizeDNSLabel(sessionName)
	if base == "" {
		base = "session"
	}
	// base is capped at 50 chars by sanitizeDNSLabel. Keep the suffix short.
	return base + "-workspace"
}

func sessionWorkspacePVCSize() resource.Quantity {
	// Use a conservative default; admin can override via env.
	s := strings.TrimSpace(os.Getenv(envSessionStorageSize))
	if s == "" {
		s = "10Gi"
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return resource.MustParse("10Gi")
	}
	return q
}

func sessionWorkspacePVCStorageClass() *string {
	s := strings.TrimSpace(os.Getenv(envSessionStorageClass))
	if s == "" {
		return nil
	}
	return &s
}

func desiredSessionWorkspacePVC(sess *operatorv1alpha1.Session) *corev1.PersistentVolumeClaim {
	size := sessionWorkspacePVCSize()
	storageClassName := sessionWorkspacePVCStorageClass()

	labels := map[string]string{
		LabelSessionName:               sess.Name,
		"app.kubernetes.io/managed-by": "kocao-control-plane-operator",
		"app.kubernetes.io/name":       "kocao-session",
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sessionWorkspacePVCName(sess.Name),
			Namespace: sess.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceStorage: size,
			}},
			StorageClassName: storageClassName,
		},
	}
}

func ensureSessionWorkspacePVC(ctx context.Context, c client.Client, scheme *runtime.Scheme, sess *operatorv1alpha1.Session) error {
	if sess == nil {
		return nil
	}
	desired := desiredSessionWorkspacePVC(sess)
	if err := controllerutil.SetControllerReference(sess, desired, scheme); err != nil {
		return err
	}

	var existing corev1.PersistentVolumeClaim
	err := c.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if apierrors.IsNotFound(err) {
		return c.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// For MVP, avoid mutating PVC sizing/class after creation.
	return nil
}
