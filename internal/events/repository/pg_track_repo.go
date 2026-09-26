package repository

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgTrackRepository struct {
	db *sqlx.DB
}

func NewPgTrackRepository(db *sqlx.DB) *PgTrackRepository {
	return &PgTrackRepository{db: db.Unsafe()}
}

func (r *PgTrackRepository) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.Track, error) {
	const q = `SELECT * FROM tracks WHERE event_id = $1 ORDER BY created_at ASC`
	var rows []struct {
		ID          uuid.UUID `db:"id"`
		EventID     uuid.UUID `db:"event_id"`
		Name        string    `db:"name"`
		Description *string   `db:"description"`
	}

	err := r.db.SelectContext(ctx, &rows, q, eventID)
	if err != nil {
		return nil, err
	}

	var tracks []*domain.Track
	for _, row := range rows {
		tracks = append(tracks, &domain.Track{
			ID:          row.ID,
			EventID:     row.EventID,
			Name:        row.Name,
			Description: row.Description,
		})
	}
	return tracks, nil
}

func (r *PgTrackRepository) Save(ctx context.Context, track *domain.Track) error {
	const q = `
		INSERT INTO tracks (id, event_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
	`
	_, err := r.db.ExecContext(ctx, q, track.ID, track.EventID, track.Name, track.Description, track.CreatedAt)
	return err
}
