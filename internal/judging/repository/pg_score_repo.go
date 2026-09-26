package repository

import (
    "context"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type PgScoreRepository struct {
    db *sqlx.DB
}

func NewPgScoreRepository(db *sqlx.DB) *PgScoreRepository {
    return &PgScoreRepository{db: db}
}

func (r *PgScoreRepository) UpsertAll(ctx context.Context, assignmentID, judgeID uuid.UUID, scores []port.ScoreInput) error {
    const q = `
        INSERT INTO scores (id, assignment_id, criterion_id, judge_id, raw_score, notes, created_at, updated_at)
        VALUES (gen_random_uuid(), :assignment_id, :criterion_id, :judge_id, :raw_score, :notes, NOW(), NOW())
        ON CONFLICT (assignment_id, criterion_id)
        DO UPDATE SET raw_score = EXCLUDED.raw_score, notes = EXCLUDED.notes, updated_at = NOW();
    `
    
    // We should use a transaction for multiple rows or just NamedExec
    // Since NamedExec supports slices natively, we can pass a slice of structs
    type scoreRow struct {
        AssignmentID uuid.UUID `db:"assignment_id"`
        CriterionID  uuid.UUID `db:"criterion_id"`
        JudgeID      uuid.UUID `db:"judge_id"`
        RawScore     int       `db:"raw_score"`
        Notes        *string   `db:"notes"`
    }

    rows := make([]scoreRow, len(scores))
    for i, s := range scores {
        rows[i] = scoreRow{
            AssignmentID: assignmentID,
            CriterionID:  s.CriterionID,
            JudgeID:      judgeID,
            RawScore:     s.RawScore,
            Notes:        s.Notes,
        }
    }

    _, err := r.db.NamedExecContext(ctx, q, rows)
    return err
}
