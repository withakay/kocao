package tokensync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// Patcher writes a single key into a Kubernetes Secret.
type Patcher interface {
	Patch(ctx context.Context, key string, data []byte) error
}

// K8sPatcher patches a named Secret using a JSON merge patch.
type K8sPatcher struct {
	client     kubernetes.Interface
	namespace  string
	secretName string
}

// NewK8sPatcher returns a Patcher that updates secretName in namespace.
func NewK8sPatcher(client kubernetes.Interface, namespace, secretName string) *K8sPatcher {
	return &K8sPatcher{
		client:     client,
		namespace:  namespace,
		secretName: secretName,
	}
}

// Patch applies a JSON merge patch that sets data[key] = base64(data).
func (p *K8sPatcher) Patch(ctx context.Context, key string, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)

	patch := map[string]any{
		"data": map[string]string{
			key: encoded,
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch: %w", err)
	}

	err = p.doPatch(ctx, patchBytes)
	if err == nil {
		slog.Info("synced secret key", "key", key, "secret", p.secretName, "namespace", p.namespace)
		return nil
	}

	// On 404 log a warning and return the error (don't crash).
	if apierrors.IsNotFound(err) {
		slog.Warn("secret not found", "secret", p.secretName, "namespace", p.namespace)
		return fmt.Errorf("secret %s/%s not found: %w", p.namespace, p.secretName, err)
	}

	// On 409 (conflict) retry once.
	if apierrors.IsConflict(err) {
		slog.Warn("conflict patching secret, retrying once", "secret", p.secretName)
		if retryErr := p.doPatch(ctx, patchBytes); retryErr != nil {
			return fmt.Errorf("retry patch: %w", retryErr)
		}
		slog.Info("synced secret key after retry", "key", key, "secret", p.secretName)
		return nil
	}

	return fmt.Errorf("patch secret: %w", err)
}

func (p *K8sPatcher) doPatch(ctx context.Context, patchBytes []byte) error {
	_, err := p.client.CoreV1().Secrets(p.namespace).Patch(
		ctx,
		p.secretName,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	return err
}
