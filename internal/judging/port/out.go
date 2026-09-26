package port

import (
    "context"
    "time"
    "github.com/google/uuid"
    "github.com/dogfood-platform/dogfood/internal/judging/domain"
)

type AssignmentRepository interface {
    Save(ctx context.Context, a *domain.JudgeAssignment) error
    BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error // single INSERT...SELECT
    FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error)
    ListByJudge(ctx context.Context, judgeID uuid.UUID, q QueueQuery) ([]*QueueRow, int, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error
}

type QueueRow struct {
    AssignmentID    uuid.UUID               `db:"assignment_id"`
    SubmissionID    uuid.UUID               `db:"submission_id"`
    SubmissionTitle string                  `db:"submission_title"`
    TeamName        string                  `db:"team_name"`
    CoverURL        *string                 `db:"cover_url"`
    EventID         uuid.UUID               `db:"event_id"`
    EventTitle      string                  `db:"event_title"`
    RubricID        uuid.UUID               `db:"rubric_id"`
    Status          domain.AssignmentStatus `db:"status"`
    AssignedAt      time.Time               `db:"assigned_at"`
}

type SubmissionReader interface {
    FindByID(ctx context.Context, id uuid.UUID) (*SubmissionDTO, error)
}

type RubricReader interface {
    FindByID(ctx context.Context, id uuid.UUID) (*RubricDTO, error)
    FindCriteriaByRubricID(ctx context.Context, rubricID uuid.UUID) ([]CriterionDTO, error)
}

type ScoreReader interface {
    ListByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]ScoreDTO, error)
}

type ScoreWriter interface {
    UpsertAll(ctx context.Context, assignmentID, judgeID uuid.UUID, scores []ScoreInput) error
}

type EventSummary struct {
    ID                uuid.UUID
    JudgingDeadlineAt *time.Time
}

type EventReader interface {
    FindSummaryByID(ctx context.Context, id uuid.UUID) (*EventSummary, error)
}

