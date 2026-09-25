package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogfood-platform/dogfood/internal/submissions/port"
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
	const q = `SELECT id, status, slug FROM events WHERE slug = $1`
	var row struct {
		ID     uuid.UUID `db:"id"`
		Status string    `db:"status"`
		Slug   string    `db:"slug"`
	}
	err := r.db.GetContext(ctx, &row, q, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("event not found")
	}
	if err != nil {
		return nil, err
	}
	return &port.EventSummary{
		ID:     row.ID,
		Status: row.Status,
		Slug:   row.Slug,
	}, nil
}

type PgTrackReader struct {
	db *sqlx.DB
}

func NewPgTrackReader(db *sqlx.DB) *PgTrackReader {
	return &PgTrackReader{db: db}
}

func (r *PgTrackReader) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*port.TrackSummary, error) {
	const q = `SELECT id, event_id, name FROM tracks WHERE event_id = $1`
	var rows []struct {
		ID      uuid.UUID `db:"id"`
		EventID uuid.UUID `db:"event_id"`
		Name    string    `db:"name"`
	}
	err := r.db.SelectContext(ctx, &rows, q, eventID)
	if err != nil {
		return nil, err
	}
	var res []*port.TrackSummary
	for _, row := range rows {
		res = append(res, &port.TrackSummary{
			ID:      row.ID,
			EventID: row.EventID,
			Name:    row.Name,
		})
	}
	return res, nil
}
