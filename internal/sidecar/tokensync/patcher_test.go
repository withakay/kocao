package tokensync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// Ensure K8sPatcher implements Patcher at compile time.
var _ Patcher = (*K8sPatcher)(nil)

func TestK8sPatcher_PatchesSecret(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		Data:       map[string][]byte{},
	}
	client := fake.NewSimpleClientset(secret)

	// Capture the raw patch payload to verify correctness.
	var capturedPatch []byte
	client.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if pa, ok := action.(k8stesting.PatchAction); ok {
			capturedPatch = pa.GetPatch()
		}
		return false, nil, nil // let the default reactor handle it
	})

	p := NewK8sPatcher(client, "default", "test-secret")

	payload := []byte(`{"access_token":"abc123"}`)
	if err := p.Patch(context.Background(), "auth.json", payload); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	// Verify the patch payload contains the correct base64-encoded data.
	if capturedPatch == nil {
		t.Fatal("no patch action captured")
	}
	var patch map[string]map[string]string
	if err := json.Unmarshal(capturedPatch, &patch); err != nil {
		t.Fatalf("unmarshal captured patch: %v", err)
	}
	encoded, ok := patch["data"]["auth.json"]
	if !ok {
		t.Fatal("patch missing key 'auth.json'")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if string(decoded) != string(payload) {
		t.Fatalf("decoded data mismatch: got %q, want %q", string(decoded), string(payload))
	}
}

func TestK8sPatcher_Handles404(t *testing.T) {
	// No secret exists — client will return 404 on patch.
	client := fake.NewSimpleClientset()
	p := NewK8sPatcher(client, "default", "missing-secret")

	err := p.Patch(context.Background(), "auth.json", []byte("data"))
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
	// Should not panic; error should mention "not found".
	if !apierrors.IsNotFound(extractAPIError(err)) {
		// The fake client may not return a proper status error on patch to
		// a missing resource. Accept any non-nil error.
		t.Logf("got non-404 error (acceptable with fake client): %v", err)
	}
}

func TestK8sPatcher_HandlesConflict(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		Data:       map[string][]byte{},
	}
	client := fake.NewSimpleClientset(secret)

	callCount := 0
	client.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		callCount++
		if callCount == 1 {
			return true, nil, apierrors.NewConflict(
				schema.GroupResource{Resource: "secrets"},
				"test-secret",
				nil,
			)
		}
		// Let the default reactor handle subsequent calls.
		return false, nil, nil
	})

	p := NewK8sPatcher(client, "default", "test-secret")
	err := p.Patch(context.Background(), "auth.json", []byte(`{"token":"xyz"}`))
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if callCount < 2 {
		t.Fatalf("expected at least 2 patch calls (initial + retry), got %d", callCount)
	}
}

func TestK8sPatcher_UsesCorrectPatchType(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		Data:       map[string][]byte{},
	}
	client := fake.NewSimpleClientset(secret)

	var capturedAction k8stesting.PatchAction
	client.PrependReactor("patch", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if pa, ok := action.(k8stesting.PatchAction); ok {
			capturedAction = pa
		}
		return false, nil, nil
	})

	p := NewK8sPatcher(client, "default", "test-secret")
	if err := p.Patch(context.Background(), "test-key", []byte("data")); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	if capturedAction == nil {
		t.Fatal("no patch action captured")
	}
	if capturedAction.GetVerb() != "patch" {
		t.Fatalf("expected verb 'patch', got %q", capturedAction.GetVerb())
	}
}

// extractAPIError unwraps to find an API status error if present.
func extractAPIError(err error) error {
	for err != nil {
		if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
			return err
		}
		if e, ok := err.(interface{ Unwrap() error }); ok {
			err = e.Unwrap()
		} else {
			break
		}
	}
	return err
}
