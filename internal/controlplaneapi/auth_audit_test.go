package controlplaneapi

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseScopesIncludesSymphonyScopes(t *testing.T) {
	scopes := parseScopes(ScopeSymphonyProjectRead + ", " + ScopeSymphonyProjectWrite + " " + ScopeSymphonyProjectControl)

	for _, want := range []string{ScopeSymphonyProjectRead, ScopeSymphonyProjectWrite, ScopeSymphonyProjectControl} {
		if !hasScope(scopes, want) {
			t.Fatalf("expected scope %q to be present", want)
		}
	}
}

func TestAuditStoreAppendRedactsSensitiveMetadata(t *testing.T) {
	audit := newAuditStore("")
	audit.Append(context.Background(), "operator", "symphony.sync", "symphony-project", "project-a", "allowed", map[string]any{
		"tokenSecretRef": map[string]any{
			"name": "github-project-token",
			"key":  "token",
		},
		"repository": "withakay/kocao",
		"nested": map[string]any{
			"authorization": "Bearer secret-value",
		},
	})

	events, err := audit.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}

	var metadata map[string]any
	if err := json.Unmarshal(events[0].Metadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["repository"] != "withakay/kocao" {
		t.Fatalf("expected repository to remain visible, got %#v", metadata["repository"])
	}
	if metadata["tokenSecretRef"] != "[redacted]" {
		t.Fatalf("expected tokenSecretRef to be redacted, got %#v", metadata["tokenSecretRef"])
	}
	nested, ok := metadata["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata object, got %#v", metadata["nested"])
	}
	if nested["authorization"] != "[redacted]" {
		t.Fatalf("expected nested authorization to be redacted, got %#v", nested["authorization"])
	}
}

func TestAppendSymphonyAuditUsesSymphonyResourceType(t *testing.T) {
	audit := newAuditStore("")
	appendSymphonyAudit(context.Background(), audit, "operator", "symphony.claim", "project-a", "allowed", map[string]any{"itemID": "PVT_item_1"})

	events, err := audit.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}
	if events[0].ResourceType != "symphony-project" {
		t.Fatalf("expected resource type symphony-project, got %q", events[0].ResourceType)
	}
}
