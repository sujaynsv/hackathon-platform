package repository

import (
    "context"
    "time"
    "fmt"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type PgAssignmentRepository struct {
    db *sqlx.DB
}

func NewPgAssignmentRepository(db *sqlx.DB) *PgAssignmentRepository {
    return &PgAssignmentRepository{db: db}
}

func (r *PgAssignmentRepository) Save(ctx context.Context, a *domain.JudgeAssignment) error {
    const q = `
        INSERT INTO judge_assignments (id, event_id, judge_id, submission_id, rubric_id, status, assigned_at, completed_at)
        VALUES (:id, :event_id, :judge_id, :submission_id, :rubric_id, :status, :assigned_at, :completed_at)
        ON CONFLICT (id) DO UPDATE SET
            status = EXCLUDED.status,
            completed_at = EXCLUDED.completed_at
    `
    row := toAssignmentRow(a)
    _, err := r.db.NamedExecContext(ctx, q, row)
    return err
}

func (r *PgAssignmentRepository) BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error {
    const q = `
        INSERT INTO judge_assignments (id, event_id, judge_id, submission_id, rubric_id, status, assigned_at)
        SELECT
            gen_random_uuid(),
            $1,                                -- event_id
            er.user_id,                        -- judge
            s.id,                              -- submission
            r.id,                              -- rubric
            'pending',
            NOW()
        FROM submissions s
        CROSS JOIN event_roles er
        JOIN rubrics r ON r.event_id = $1
        WHERE s.event_id = $1
          AND s.status = 'submitted'
          AND er.event_id = $1
          AND er.role = 'judge'
        ON CONFLICT (judge_id, submission_id) DO NOTHING;
    `
    _, err := r.db.ExecContext(ctx, q, eventID)
    return err
}

func (r *PgAssignmentRepository) FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error) {
    const q = `
        SELECT id, event_id, judge_id, submission_id, rubric_id, status, assigned_at, completed_at
        FROM judge_assignments
        WHERE id = $1 AND judge_id = $2
    `
    var row assignmentRow
    if err := r.db.GetContext(ctx, &row, q, id, judgeID); err != nil {
        return nil, err
    }
    return row.toDomain(), nil
}

func (r *PgAssignmentRepository) ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) {
    baseQuery := `
        FROM judge_assignments ja
        JOIN submissions s ON s.id = ja.submission_id
        JOIN events e ON e.id = ja.event_id
        LEFT JOIN teams t ON t.id = s.team_id
        WHERE ja.judge_id = $1
    `
    
    args := []interface{}{judgeID}
    argID := 2

    if q.EventID != nil {
        baseQuery += fmt.Sprintf(" AND ja.event_id = $%d", argID)
        args = append(args, *q.EventID)
        argID++
    }

    if q.Status != "" && q.Status != "all" {
        baseQuery += fmt.Sprintf(" AND ja.status = $%d", argID)
        args = append(args, q.Status)
        argID++
    }

    countQuery := "SELECT COUNT(*) " + baseQuery
    var total int
    if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
        return nil, 0, err
    }

    if total == 0 {
        return []*port.QueueRow{}, 0, nil
    }

    offset := (q.Page - 1) * q.PageSize
    limit := q.PageSize

    selectQuery := `
        SELECT 
            ja.id as assignment_id,
            s.id as submission_id,
            s.title as submission_title,
            COALESCE(t.name, 'No Team') as team_name,
            s.cover_url,
            e.id as event_id,
            e.title as event_title,
            ja.rubric_id,
            ja.status,
            ja.assigned_at
    ` + baseQuery + fmt.Sprintf(" ORDER BY ja.assigned_at DESC LIMIT $%d OFFSET $%d", argID, argID+1)
    
    args = append(args, limit, offset)

    var rows []*port.QueueRow
    if err := r.db.SelectContext(ctx, &rows, selectQuery, args...); err != nil {
        return nil, 0, err
    }

    return rows, total, nil
}

func (r *PgAssignmentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error {
    const q = `UPDATE judge_assignments SET status = $1 WHERE id = $2`
    _, err := r.db.ExecContext(ctx, q, status, id)
    return err
}

type assignmentRow struct {
    ID           uuid.UUID  `db:"id"`
    EventID      uuid.UUID  `db:"event_id"`
    JudgeID      uuid.UUID  `db:"judge_id"`
    SubmissionID uuid.UUID  `db:"submission_id"`
    RubricID     uuid.UUID  `db:"rubric_id"`
    Status       string     `db:"status"`
    AssignedAt   time.Time  `db:"assigned_at"`
    CompletedAt  *time.Time `db:"completed_at"`
}

func toAssignmentRow(a *domain.JudgeAssignment) assignmentRow {
    return assignmentRow{
        ID:           a.ID,
        EventID:      a.EventID,
        JudgeID:      a.JudgeID,
        SubmissionID: a.SubmissionID,
        RubricID:     a.RubricID,
        Status:       string(a.Status),
        AssignedAt:   a.AssignedAt,
        CompletedAt:  a.CompletedAt,
    }
}

func (r *assignmentRow) toDomain() *domain.JudgeAssignment {
    return &domain.JudgeAssignment{
        ID:           r.ID,
        EventID:      r.EventID,
        JudgeID:      r.JudgeID,
        SubmissionID: r.SubmissionID,
        RubricID:     r.RubricID,
        Status:       domain.AssignmentStatus(r.Status),
        AssignedAt:   r.AssignedAt,
        CompletedAt:  r.CompletedAt,
    }
}
