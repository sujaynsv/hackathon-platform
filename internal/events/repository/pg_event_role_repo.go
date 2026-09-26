package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgEventRoleRepository struct {
	db *sqlx.DB
}

func NewPgEventRoleRepository(db *sqlx.DB) *PgEventRoleRepository {
	return &PgEventRoleRepository{db: db}
}

func (r *PgEventRoleRepository) GrantRole(ctx context.Context, userID, eventID uuid.UUID, role string) error {
	const q = `
		INSERT INTO event_roles (user_id, event_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (event_id, user_id, role) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, userID, eventID, role)
	return err
}

func (r *PgEventRoleRepository) GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error) {
	const q = `SELECT role FROM event_roles WHERE user_id = $1 AND event_id = $2`
	var role string
	err := r.db.GetContext(ctx, &role, q, userID, eventID)
	return role, err
}
