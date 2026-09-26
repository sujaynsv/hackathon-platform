package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgTeamRepository struct {
	db *sqlx.DB
}

func NewPgTeamRepository(db *sqlx.DB) *PgTeamRepository {
	return &PgTeamRepository{db: db}
}

func (r *PgTeamRepository) FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*domain.Team, error) {
	const q = `
		SELECT t.id, t.event_id, t.name, t.invite_code, t.created_by, t.is_locked, t.locked_at, t.created_at, t.updated_at
		FROM team_members tm
		JOIN teams t ON t.id = tm.team_id
		WHERE tm.event_id = $1 AND tm.user_id = $2
	`
	var row struct {
		ID         uuid.UUID     `db:"id"`
		EventID    uuid.UUID     `db:"event_id"`
		Name       string        `db:"name"`
		InviteCode string        `db:"invite_code"`
		CreatedBy  uuid.UUID     `db:"created_by"`
		IsLocked   bool          `db:"is_locked"`
		LockedAt   sql.NullTime  `db:"locked_at"`
		CreatedAt  sql.NullTime  `db:"created_at"`
		UpdatedAt  sql.NullTime  `db:"updated_at"`
	}
	err := r.db.GetContext(ctx, &row, q, eventID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	team := &domain.Team{
		ID:         row.ID,
		EventID:    row.EventID,
		Name:       row.Name,
		InviteCode: row.InviteCode,
		CreatedBy:  row.CreatedBy,
		IsLocked:   row.IsLocked,
	}
	if row.LockedAt.Valid {
		team.LockedAt = &row.LockedAt.Time
	}
	if row.CreatedAt.Valid {
		team.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		team.UpdatedAt = row.UpdatedAt.Time
	}

	return team, nil
}

func (r *PgTeamRepository) ExistsByName(ctx context.Context, eventID uuid.UUID, name string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM teams WHERE event_id = $1 AND name = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, q, eventID, name)
	return exists, err
}

func (r *PgTeamRepository) FindByInviteCode(ctx context.Context, code string) (*domain.Team, error) {
	const q = `
		SELECT id, event_id, name, invite_code, created_by, is_locked, locked_at, created_at, updated_at
		FROM teams
		WHERE invite_code = $1
	`
	var row struct {
		ID         uuid.UUID     `db:"id"`
		EventID    uuid.UUID     `db:"event_id"`
		Name       string        `db:"name"`
		InviteCode string        `db:"invite_code"`
		CreatedBy  uuid.UUID     `db:"created_by"`
		IsLocked   bool          `db:"is_locked"`
		LockedAt   sql.NullTime  `db:"locked_at"`
		CreatedAt  sql.NullTime  `db:"created_at"`
		UpdatedAt  sql.NullTime  `db:"updated_at"`
	}
	err := r.db.GetContext(ctx, &row, q, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	team := &domain.Team{
		ID:         row.ID,
		EventID:    row.EventID,
		Name:       row.Name,
		InviteCode: row.InviteCode,
		CreatedBy:  row.CreatedBy,
		IsLocked:   row.IsLocked,
	}
	if row.LockedAt.Valid {
		team.LockedAt = &row.LockedAt.Time
	}
	if row.CreatedAt.Valid {
		team.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		team.UpdatedAt = row.UpdatedAt.Time
	}

	return team, nil
}

func (r *PgTeamRepository) Save(ctx context.Context, team *domain.Team) error {
	const q = `
		INSERT INTO teams (id, event_id, name, invite_code, created_by, is_locked, locked_at, created_at, updated_at)
		VALUES (:id, :event_id, :name, :invite_code, :created_by, :is_locked, :locked_at, :created_at, :updated_at)
	`
	type teamRow struct {
		ID         uuid.UUID     `db:"id"`
		EventID    uuid.UUID     `db:"event_id"`
		Name       string        `db:"name"`
		InviteCode string        `db:"invite_code"`
		CreatedBy  uuid.UUID     `db:"created_by"`
		IsLocked   bool          `db:"is_locked"`
		LockedAt   sql.NullTime  `db:"locked_at"`
		CreatedAt  sql.NullTime  `db:"created_at"`
		UpdatedAt  sql.NullTime  `db:"updated_at"`
	}
	row := teamRow{
		ID:         team.ID,
		EventID:    team.EventID,
		Name:       team.Name,
		InviteCode: team.InviteCode,
		CreatedBy:  team.CreatedBy,
		IsLocked:   team.IsLocked,
	}
	if team.LockedAt != nil {
		row.LockedAt = sql.NullTime{Time: *team.LockedAt, Valid: true}
	}
	row.CreatedAt = sql.NullTime{Time: team.CreatedAt, Valid: !team.CreatedAt.IsZero()}
	row.UpdatedAt = sql.NullTime{Time: team.UpdatedAt, Valid: !team.UpdatedAt.IsZero()}

	_, err := r.db.NamedExecContext(ctx, q, row)
	return err
}

func (r *PgTeamRepository) AddMember(ctx context.Context, member *domain.TeamMember) error {
	const q = `
		INSERT INTO team_members (id, team_id, user_id, event_id, role, joined_at)
		VALUES (:id, :team_id, :user_id, :event_id, :role, :joined_at)
	`
	type memberRow struct {
		ID       uuid.UUID `db:"id"`
		TeamID   uuid.UUID `db:"team_id"`
		UserID   uuid.UUID `db:"user_id"`
		EventID  uuid.UUID `db:"event_id"`
		Role     string    `db:"role"`
		JoinedAt time.Time `db:"joined_at"`
	}
	row := memberRow{
		ID:       member.ID,
		TeamID:   member.TeamID,
		UserID:   member.UserID,
		EventID:  member.EventID,
		Role:     string(member.Role),
		JoinedAt: member.JoinedAt,
	}
	if row.ID == uuid.Nil {
		row.ID = uuid.New() // DB has default gen_random_uuid(), but this is fine
	}
	_, err := r.db.NamedExecContext(ctx, q, row)
	return err
}

func (r *PgTeamRepository) GetMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamMember, error) {
	const q = `
		SELECT id, team_id, user_id, event_id, role, joined_at
		FROM team_members
		WHERE team_id = $1
		ORDER BY joined_at ASC
	`
	var rows []struct {
		ID       uuid.UUID `db:"id"`
		TeamID   uuid.UUID `db:"team_id"`
		UserID   uuid.UUID `db:"user_id"`
		EventID  uuid.UUID `db:"event_id"`
		Role     string    `db:"role"`
		JoinedAt time.Time `db:"joined_at"`
	}
	if err := r.db.SelectContext(ctx, &rows, q, teamID); err != nil {
		return nil, err
	}
	
	members := make([]domain.TeamMember, len(rows))
	for i, row := range rows {
		members[i] = domain.TeamMember{
			ID:       row.ID,
			TeamID:   row.TeamID,
			UserID:   row.UserID,
			EventID:  row.EventID,
			Role:     domain.TeamMemberRole(row.Role),
			JoinedAt: row.JoinedAt,
		}
	}
	return members, nil
}
