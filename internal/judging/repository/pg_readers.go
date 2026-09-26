package repository

import (
    "context"

    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type PgReaders struct {
    db *sqlx.DB
}

func NewPgReaders(db *sqlx.DB) *PgReaders {
    return &PgReaders{db: db}
}

func (r *PgReaders) FindByID(ctx context.Context, id uuid.UUID) (*port.SubmissionDTO, error) {
    const q = `
        SELECT
            s.id, s.title, s.description, s.repo_url, s.demo_url, s.cover_url,
            t.id AS team_id, t.name AS team_name
        FROM submissions s
        JOIN teams t ON s.team_id = t.id
        WHERE s.id = $1
    `
    var row struct {
        ID          uuid.UUID `db:"id"`
        Title       string    `db:"title"`
        Description string    `db:"description"`
        RepoURL     string    `db:"repo_url"`
        DemoURL     string    `db:"demo_url"`
        CoverURL    *string   `db:"cover_url"`
        TeamID      uuid.UUID `db:"team_id"`
        TeamName    string    `db:"team_name"`
    }
    err := r.db.GetContext(ctx, &row, q, id)
    if err != nil {
        return nil, err
    }
    return &port.SubmissionDTO{
        ID:          row.ID,
        Title:       row.Title,
        Description: row.Description,
        RepoURL:     row.RepoURL,
        DemoURL:     row.DemoURL,
        CoverURL:    row.CoverURL,
        Team: port.TeamDTO{
            ID:   row.TeamID,
            Name: row.TeamName,
        },
    }, nil
}

type pgRubricReader struct {
    db *sqlx.DB
}

func NewPgRubricReader(db *sqlx.DB) *pgRubricReader {
    return &pgRubricReader{db: db}
}

func (r *pgRubricReader) FindByID(ctx context.Context, id uuid.UUID) (*port.RubricDTO, error) {
    const q = `SELECT id, name FROM rubrics WHERE id = $1`
    var dto port.RubricDTO
    err := r.db.GetContext(ctx, &dto, q, id)
    if err != nil {
        return nil, err
    }

    const qCrit = `SELECT id, name, weight, max_score, description FROM rubric_criteria WHERE rubric_id = $1`
    err = r.db.SelectContext(ctx, &dto.Criteria, qCrit, id)
    if err != nil {
        return nil, err
    }

    return &dto, nil
}

type pgScoreReader struct {
    db *sqlx.DB
}

func NewPgScoreReader(db *sqlx.DB) *pgScoreReader {
    return &pgScoreReader{db: db}
}

func (r *pgScoreReader) ListByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]port.ScoreDTO, error) {
    const q = `SELECT criterion_id, raw_score, notes FROM scores WHERE assignment_id = $1`
    var scores []port.ScoreDTO
    err := r.db.SelectContext(ctx, &scores, q, assignmentID)
    if err != nil {
        return nil, err
    }
    return scores, nil
}
