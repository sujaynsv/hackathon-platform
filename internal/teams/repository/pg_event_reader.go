package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgEventReader struct {
	db *sqlx.DB
}

func NewPgEventReader(db *sqlx.DB) *PgEventReader {
	return &PgEventReader{db: db}
}

func (r *PgEventReader) FindSummaryBySlug(ctx context.Context, slug string) (*port.EventSummary, error) {
	const q = `SELECT id, slug, status, registration_closes_at FROM events WHERE slug = $1`
	var event port.EventSummary
	var eventID uuid.UUID
	if err := r.db.QueryRowxContext(ctx, q, slug).Scan(&eventID, &event.Slug, &event.Status, &event.RegistrationClosesAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, response.ErrNotFound
		}
		return nil, err
	}
	event.ID = eventID
	return &event, nil
}
