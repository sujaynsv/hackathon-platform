package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgEventRepository struct {
	db *sqlx.DB
}

func NewPgEventRepository(db *sqlx.DB) *PgEventRepository {
	return &PgEventRepository{db: db}
}

func toEventRow(e *domain.Event) map[string]interface{} {
	return map[string]interface{}{
		"id":                     e.ID,
		"slug":                   e.Slug,
		"title":                  e.Title,
		"description":            e.Description,
		"banner_url":             e.BannerURL,
		"organizer_id":           e.OrganizerID,
		"status":                 e.Status,
		"normalization_status":   e.NormalizationStatus,
		"registration_opens_at":  e.RegistrationOpensAt,
		"registration_closes_at": e.RegistrationClosesAt,
		"submission_deadline_at": e.SubmissionDeadlineAt,
		"judging_deadline_at":    e.JudgingDeadlineAt,
		"voting_opens_at":        e.VotingOpensAt,
		"voting_closes_at":       e.VotingClosesAt,
		"max_team_size":          e.MaxTeamSize,
		"created_at":             e.CreatedAt,
		"updated_at":             e.UpdatedAt,
	}
}

func fromEventRow(row map[string]interface{}) *domain.Event {
	// A helper could be written here, but let's just use StructScan usually
	return nil
}

type eventRow struct {
	ID                   uuid.UUID  `db:"id"`
	Slug                 string     `db:"slug"`
	Title                string     `db:"title"`
	Description          *string    `db:"description"`
	BannerURL            *string    `db:"banner_url"`
	OrganizerID          uuid.UUID  `db:"organizer_id"`
	Status               string     `db:"status"`
	NormalizationStatus  string     `db:"normalization_status"`
	RegistrationOpensAt  *time.Time `db:"registration_opens_at"`
	RegistrationClosesAt *time.Time `db:"registration_closes_at"`
	SubmissionDeadlineAt *time.Time `db:"submission_deadline_at"`
	JudgingDeadlineAt    *time.Time `db:"judging_deadline_at"`
	VotingOpensAt        *time.Time `db:"voting_opens_at"`
	VotingClosesAt       *time.Time `db:"voting_closes_at"`
	MaxTeamSize          int        `db:"max_team_size"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
}

func (r *eventRow) toDomain() *domain.Event {
	return &domain.Event{
		ID:                   r.ID,
		Slug:                 r.Slug,
		Title:                r.Title,
		Description:          r.Description,
		BannerURL:            r.BannerURL,
		OrganizerID:          r.OrganizerID,
		Status:               domain.EventStatus(r.Status),
		NormalizationStatus:  r.NormalizationStatus,
		RegistrationOpensAt:  r.RegistrationOpensAt,
		RegistrationClosesAt: r.RegistrationClosesAt,
		SubmissionDeadlineAt: r.SubmissionDeadlineAt,
		JudgingDeadlineAt:    r.JudgingDeadlineAt,
		VotingOpensAt:        r.VotingOpensAt,
		VotingClosesAt:       r.VotingClosesAt,
		MaxTeamSize:          r.MaxTeamSize,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func (r *PgEventRepository) Save(ctx context.Context, event *domain.Event) error {
	const q = `
		INSERT INTO events (
			id, slug, title, description, banner_url, organizer_id, status, normalization_status,
			registration_opens_at, registration_closes_at, submission_deadline_at,
			judging_deadline_at, voting_opens_at, voting_closes_at, max_team_size, created_at, updated_at
		) VALUES (
			:id, :slug, :title, :description, :banner_url, :organizer_id, :status, :normalization_status,
			:registration_opens_at, :registration_closes_at, :submission_deadline_at,
			:judging_deadline_at, :voting_opens_at, :voting_closes_at, :max_team_size, :created_at, :updated_at
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			banner_url = EXCLUDED.banner_url,
			status = EXCLUDED.status,
			normalization_status = EXCLUDED.normalization_status,
			registration_opens_at = EXCLUDED.registration_opens_at,
			registration_closes_at = EXCLUDED.registration_closes_at,
			submission_deadline_at = EXCLUDED.submission_deadline_at,
			judging_deadline_at = EXCLUDED.judging_deadline_at,
			voting_opens_at = EXCLUDED.voting_opens_at,
			voting_closes_at = EXCLUDED.voting_closes_at,
			max_team_size = EXCLUDED.max_team_size,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.NamedExecContext(ctx, q, toEventRow(event))
	return err
}

func (r *PgEventRepository) FindBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	const q = `SELECT * FROM events WHERE slug = $1`
	var row eventRow
	err := r.db.GetContext(ctx, &row, q, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *PgEventRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM events WHERE slug = $1)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, q, slug)
	return exists, err
}

func (r *PgEventRepository) ListPublished(ctx context.Context, page, pageSize int) ([]*domain.Event, int, error) {
	const q = `
		SELECT *, COUNT(*) OVER() AS total_count
		FROM events
		WHERE status != 'draft'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	type listRow struct {
		eventRow
		TotalCount int `db:"total_count"`
	}

	offset := (page - 1) * pageSize
	var rows []listRow
	err := r.db.SelectContext(ctx, &rows, q, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []*domain.Event{}, 0, nil
	}

	events := make([]*domain.Event, len(rows))
	for i, r := range rows {
		events[i] = r.toDomain()
	}
	return events, rows[0].TotalCount, nil
}

func (r *PgEventRepository) GetEventDetail(ctx context.Context, slug string, callerID *uuid.UUID) (*domain.Event, []domain.Track, *string, error) {
	const q = `
		SELECT
			e.*,
			t.id AS track_id, t.name AS track_name, t.description AS track_desc, t.created_at AS track_created_at,
			er.role AS my_role
		FROM events e
		LEFT JOIN tracks t ON t.event_id = e.id
		LEFT JOIN event_roles er ON er.event_id = e.id AND er.user_id = $2
		WHERE e.slug = $1
	`

	type detailRow struct {
		eventRow
		TrackID        *uuid.UUID `db:"track_id"`
		TrackName      *string    `db:"track_name"`
		TrackDesc      *string    `db:"track_desc"`
		TrackCreatedAt *time.Time `db:"track_created_at"`
		MyRole         *string    `db:"my_role"`
	}

	var rows []detailRow
	var idParam interface{} = uuid.Nil
	if callerID != nil {
		idParam = *callerID
	}

	err := r.db.SelectContext(ctx, &rows, q, slug, idParam)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, nil, response.ErrNotFound
	}

	event := rows[0].toDomain()
	var myRole *string = rows[0].MyRole

	var tracks []domain.Track
	for _, row := range rows {
		if row.TrackID != nil {
			tracks = append(tracks, domain.Track{
				ID:          *row.TrackID,
				EventID:     event.ID,
				Name:        *row.TrackName,
				Description: row.TrackDesc,
				CreatedAt:   *row.TrackCreatedAt,
			})
		}
	}

	return event, tracks, myRole, nil
}
