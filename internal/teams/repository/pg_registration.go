package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgParticipantRepository struct {
	db *sqlx.DB
}

func NewPgParticipantRepository(db *sqlx.DB) *PgParticipantRepository {
	return &PgParticipantRepository{db: db}
}

func (r *PgParticipantRepository) HasRole(ctx context.Context, userID, eventID uuid.UUID, role string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM event_roles WHERE user_id = $1 AND event_id = $2 AND role = $3)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, q, userID, eventID, role)
	return exists, err
}

func (r *PgParticipantRepository) GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error) {
	const q = `SELECT role FROM event_roles WHERE user_id = $1 AND event_id = $2 LIMIT 1`
	var role string
	err := r.db.GetContext(ctx, &role, q, userID, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return "unregistered", nil
	}
	return role, err
}

func (r *PgParticipantRepository) GrantParticipantRole(ctx context.Context, userID, eventID uuid.UUID) (time.Time, error) {
	const q = `
		INSERT INTO event_roles (user_id, event_id, role)
		VALUES ($1, $2, 'participant')
		ON CONFLICT (event_id, user_id, role) DO NOTHING
		RETURNING granted_at
	`
	var registeredAt time.Time
	err := r.db.GetContext(ctx, &registeredAt, q, userID, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, response.ErrDuplicate
	}
	return registeredAt, err
}

func (r *PgParticipantRepository) RevokeParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error {
	const q = `DELETE FROM event_roles WHERE user_id = $1 AND event_id = $2 AND role = 'participant'`
	result, err := r.db.ExecContext(ctx, q, userID, eventID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return response.ErrNotFound
	}
	return nil
}
