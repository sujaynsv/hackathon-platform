package repository

import (
	"context"
	"encoding/json"

	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgUserReader struct {
	db *sqlx.DB
}

func NewPgUserReader(db *sqlx.DB) *PgUserReader {
	return &PgUserReader{db: db.Unsafe()}
}

func (r *PgUserReader) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	const q = `SELECT is_admin FROM users WHERE id = $1`
	var isAdmin bool
	err := r.db.GetContext(ctx, &isAdmin, q, userID)
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

type PgAuditLogWriter struct {
	db *sqlx.DB
}

func NewPgAuditLogWriter(db *sqlx.DB) *PgAuditLogWriter {
	return &PgAuditLogWriter{db: db.Unsafe()}
}

func (r *PgAuditLogWriter) Write(ctx context.Context, entry *port.AuditEntry) error {
	const q = `
		INSERT INTO audit_log (actor_id, action, resource_type, resource_id, changes)
		VALUES ($1, $2, $3, $4, $5)
	`
	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, q, entry.ActorID, entry.Action, entry.ResourceType, entry.ResourceID, changesJSON)
	return err
}
