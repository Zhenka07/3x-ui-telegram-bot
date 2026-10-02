package storage

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type AuditLog struct {
	AdminID     int64
	Action      string
	InboundID   int
	ClientEmail string
	Details     string
	CreatedAt   time.Time
}

type AuditRepo struct {
	db *DB
}

// NewAuditRepo creates a new AuditRepo instance.
func NewAuditRepo(db *DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Log records an administrator action in the audit log.
func (r *AuditRepo) Log(ctx context.Context, entry AuditLog) error {
	ctx, span := otel.Tracer("sqlite").Start(ctx, "db.audit_log",
		trace.WithAttributes(
			attribute.String("db.system", "sqlite"),
			attribute.String("db.operation", "INSERT"),
			attribute.Int64("admin.id", entry.AdminID),
			attribute.String("audit.action", entry.Action),
		),
	)
	defer span.End()

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	query := `INSERT INTO audit_logs (admin_id, action, inbound_id, client_email, details, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.sql.ExecContext(ctx, query,
		entry.AdminID,
		entry.Action,
		entry.InboundID,
		entry.ClientEmail,
		entry.Details,
		entry.CreatedAt.Unix(),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
