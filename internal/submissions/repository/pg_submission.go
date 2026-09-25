package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgSubmissionRepository struct {
	db *sqlx.DB
}

func NewPgSubmissionRepository(db *sqlx.DB) *PgSubmissionRepository {
	return &PgSubmissionRepository{db: db}
}

type submissionRow struct {
	ID          uuid.UUID `db:"id"`
	TeamID      uuid.UUID `db:"team_id"`
	EventID     uuid.UUID `db:"event_id"`
	TrackID     *uuid.UUID `db:"track_id"`
	Title       string    `db:"title"`
	Description *string   `db:"description"`
	DemoURL     *string   `db:"demo_url"`
	RepoURL     *string   `db:"repo_url"`
	VideoURL    *string   `db:"video_url"`
	CoverURL    *string   `db:"cover_image_url"`
	Status      string    `db:"status"`
	FinalScore  *float64  `db:"final_score"`
	Rank        *int      `db:"rank"`
	SubmittedAt *sql.NullTime `db:"submitted_at"`
	CreatedAt   sql.NullTime  `db:"created_at"`
	UpdatedAt   sql.NullTime  `db:"updated_at"`
}

func (r *PgSubmissionRepository) Save(ctx context.Context, sub *domain.Submission) error {
	const q = `
		INSERT INTO submissions (
			id, team_id, event_id, track_id, title, description, 
			demo_url, repo_url, video_url, cover_image_url, status, created_at, updated_at
		) VALUES (
			:id, :team_id, :event_id, :track_id, :title, :description,
			:demo_url, :repo_url, :video_url, :cover_image_url, :status, :created_at, :updated_at
		)
	`
	row := submissionRow{
		ID:          sub.ID,
		TeamID:      sub.TeamID,
		EventID:     sub.EventID,
		TrackID:     sub.TrackID,
		Title:       sub.Title,
		Description: sub.Description,
		DemoURL:     sub.DemoURL,
		RepoURL:     sub.RepoURL,
		VideoURL:    sub.VideoURL,
		CoverURL:    sub.CoverURL,
		Status:      string(sub.Status),
		CreatedAt:   sql.NullTime{Time: sub.CreatedAt, Valid: true},
		UpdatedAt:   sql.NullTime{Time: sub.UpdatedAt, Valid: true},
	}
	_, err := r.db.NamedExecContext(ctx, q, row)
	return err
}

func (r *PgSubmissionRepository) FindByTeamAndEvent(ctx context.Context, teamID, eventID uuid.UUID) (*domain.Submission, error) {
	const q = `SELECT * FROM submissions WHERE team_id = $1 AND event_id = $2`
	var row submissionRow
	err := r.db.GetContext(ctx, &row, q, teamID, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain(&row), nil
}

func (r *PgSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	const q = `SELECT * FROM submissions WHERE id = $1`
	var row submissionRow
	err := r.db.GetContext(ctx, &row, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.toDomain(&row), nil
}

func (r *PgSubmissionRepository) Update(ctx context.Context, sub *domain.Submission) error {
	const q = `
		UPDATE submissions SET 
			track_id = :track_id,
			title = :title,
			description = :description,
			demo_url = :demo_url,
			repo_url = :repo_url,
			video_url = :video_url,
			cover_image_url = :cover_image_url,
			status = :status,
			final_score = :final_score,
			rank = :rank,
			updated_at = :updated_at,
			submitted_at = :submitted_at
		WHERE id = :id
	`
	var submittedAt sql.NullTime
	if sub.SubmittedAt != nil {
		submittedAt = sql.NullTime{Time: *sub.SubmittedAt, Valid: true}
	}

	row := submissionRow{
		ID:          sub.ID,
		TrackID:     sub.TrackID,
		Title:       sub.Title,
		Description: sub.Description,
		DemoURL:     sub.DemoURL,
		RepoURL:     sub.RepoURL,
		VideoURL:    sub.VideoURL,
		CoverURL:    sub.CoverURL,
		Status:      string(sub.Status),
		FinalScore:  sub.FinalScore,
		Rank:        sub.TrackRank,
		SubmittedAt: &submittedAt,
		UpdatedAt:   sql.NullTime{Time: sub.UpdatedAt, Valid: true},
	}
	_, err := r.db.NamedExecContext(ctx, q, row)
	return err
}

func (r *PgSubmissionRepository) toDomain(row *submissionRow) *domain.Submission {
	sub := &domain.Submission{
		ID:          row.ID,
		TeamID:      row.TeamID,
		EventID:     row.EventID,
		TrackID:     row.TrackID,
		Title:       row.Title,
		Description: row.Description,
		DemoURL:     row.DemoURL,
		RepoURL:     row.RepoURL,
		VideoURL:    row.VideoURL,
		CoverURL:    row.CoverURL,
		Status:      domain.SubmissionStatus(row.Status),
		FinalScore:  row.FinalScore,
		TrackRank:   row.Rank,
	}
	if row.SubmittedAt != nil && row.SubmittedAt.Valid {
		sub.SubmittedAt = &row.SubmittedAt.Time
	}
	if row.CreatedAt.Valid {
		sub.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		sub.UpdatedAt = row.UpdatedAt.Time
	}
	return sub
}
