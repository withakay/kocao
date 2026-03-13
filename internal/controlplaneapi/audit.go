package controlplaneapi

import (
	"context"

	"github.com/withakay/kocao/internal/auditlog"
)

type AuditEvent = auditlog.Event
type AuditStore = auditlog.Store

func newAuditStore(path string) *AuditStore {
	return auditlog.New(path, newID)
}

func appendSymphonyAudit(ctx context.Context, audit *AuditStore, actor, action, resourceID, outcome string, metadata map[string]any) {
	auditlog.AppendSymphony(ctx, audit, actor, action, resourceID, outcome, metadata)
}
