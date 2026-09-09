package storage

import (
	"context"
	"time"
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
	return err
}
