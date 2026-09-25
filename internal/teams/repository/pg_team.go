package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgTeamReader struct {
	db *sqlx.DB
}

func NewPgTeamReader(db *sqlx.DB) *PgTeamReader {
	return &PgTeamReader{db: db}
}

func (r *PgTeamReader) FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*port.TeamSummary, error) {
	const q = `
		SELECT t.id, t.name 
		FROM team_members tm
		JOIN teams t ON t.id = tm.team_id
		WHERE tm.event_id = $1 AND tm.user_id = $2
	`
	var row struct {
		ID   uuid.UUID `db:"id"`
		Name string    `db:"name"`
	}
	err := r.db.GetContext(ctx, &row, q, eventID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &port.TeamSummary{
		ID:   row.ID,
		Name: row.Name,
	}, nil
}
