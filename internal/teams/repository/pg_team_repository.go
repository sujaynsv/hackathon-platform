package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type PgTeamRepository struct {
	db *sqlx.DB
}

func NewPgTeamRepository(db *sqlx.DB) *PgTeamRepository {
	return &PgTeamRepository{db: db}
}

func (r *PgTeamRepository) CreateWithLeader(ctx context.Context, team *domain.Team, leader *domain.TeamMember) (*domain.Team, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	const insertTeam = `
		INSERT INTO teams (id, event_id, name, invite_code, created_by, is_locked, created_at)
		VALUES ($1, $2, $3, $4, $5, false, $6)
	`
	if _, err := tx.ExecContext(ctx, insertTeam,
		team.ID, team.EventID, team.Name, team.InviteCode, team.CreatedBy, team.CreatedAt,
	); err != nil {
		return nil, mapTeamConstraintError(err)
	}

	const insertMember = `
		INSERT INTO team_members (team_id, user_id, event_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.ExecContext(ctx, insertMember,
		leader.TeamID, leader.UserID, leader.EventID, leader.Role, leader.JoinedAt,
	); err != nil {
		return nil, mapTeamConstraintError(err)
	}

	const profileQuery = `SELECT display_name, avatar_url FROM users WHERE id = $1`
	if err := tx.QueryRowxContext(ctx, profileQuery, leader.UserID).Scan(&leader.DisplayName, &leader.AvatarURL); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	team.Members = []domain.TeamMember{*leader}
	return team, nil
}

func (r *PgTeamRepository) FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*domain.Team, error) {
	const query = `
		SELECT t.id, t.event_id, t.name, t.invite_code, t.created_by, t.is_locked, t.created_at
		FROM teams t
		JOIN team_members caller_member ON caller_member.team_id = t.id
		WHERE t.event_id = $1 AND caller_member.user_id = $2
	`
	var row teamRow
	if err := r.db.GetContext(ctx, &row, query, eventID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	team := row.toDomain()
	members, err := r.findMembers(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	team.Members = members
	return team, nil
}

func (r *PgTeamRepository) ExistsByName(ctx context.Context, eventID uuid.UUID, name string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM teams WHERE event_id = $1 AND name = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, eventID, name)
	return exists, err
}

func (r *PgTeamRepository) findMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamMember, error) {
	const query = `
		SELECT tm.team_id, tm.user_id, tm.event_id, u.display_name, u.avatar_url, tm.role, tm.joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.team_id = $1
		ORDER BY tm.joined_at, tm.user_id
	`
	var rows []teamMemberRow
	if err := r.db.SelectContext(ctx, &rows, query, teamID); err != nil {
		return nil, err
	}
	members := make([]domain.TeamMember, len(rows))
	for i, row := range rows {
		members[i] = row.toDomain()
	}
	return members, nil
}

func mapTeamConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}

	var domainErr error
	switch pgErr.ConstraintName {
	case "idx_teams_invite_code":
		domainErr = domain.ErrInviteCodeTaken
	case "idx_teams_event_name":
		domainErr = domain.ErrTeamNameTaken
	case "idx_team_members_user_event":
		domainErr = domain.ErrAlreadyOnTeam
	default:
		return err
	}
	return fmt.Errorf("%w: %w", domainErr, err)
}

type teamRow struct {
	ID         uuid.UUID `db:"id"`
	EventID    uuid.UUID `db:"event_id"`
	Name       string    `db:"name"`
	InviteCode string    `db:"invite_code"`
	CreatedBy  uuid.UUID `db:"created_by"`
	IsLocked   bool      `db:"is_locked"`
	CreatedAt  time.Time `db:"created_at"`
}

func (row teamRow) toDomain() *domain.Team {
	return &domain.Team{
		ID:         row.ID,
		EventID:    row.EventID,
		Name:       row.Name,
		InviteCode: row.InviteCode,
		CreatedBy:  row.CreatedBy,
		IsLocked:   row.IsLocked,
		CreatedAt:  row.CreatedAt,
	}
}

type teamMemberRow struct {
	TeamID      uuid.UUID `db:"team_id"`
	UserID      uuid.UUID `db:"user_id"`
	EventID     uuid.UUID `db:"event_id"`
	DisplayName string    `db:"display_name"`
	AvatarURL   *string   `db:"avatar_url"`
	Role        string    `db:"role"`
	JoinedAt    time.Time `db:"joined_at"`
}

func (row teamMemberRow) toDomain() domain.TeamMember {
	return domain.TeamMember{
		TeamID:      row.TeamID,
		UserID:      row.UserID,
		EventID:     row.EventID,
		DisplayName: row.DisplayName,
		AvatarURL:   row.AvatarURL,
		Role:        row.Role,
		JoinedAt:    row.JoinedAt,
	}
}
