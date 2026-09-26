package port

import (
    "context"
    "github.com/google/uuid"
)

type GetQueueUseCase interface {
    GetQueue(ctx context.Context, judgeID uuid.UUID, q QueueQuery) (*AssignmentListDTO, error)
}

type GetAssignmentDetailUseCase interface {
    GetDetail(ctx context.Context, judgeID, assignmentID uuid.UUID) (*AssignmentDetailDTO, error)
}

type CreateAssignmentsUseCase interface {
    CreateForEvent(ctx context.Context, eventID uuid.UUID) error
}

type QueueQuery struct {
    EventID *uuid.UUID
    Status  string
    Page    int
    PageSize int
}

type AssignmentListDTO struct {
    Data []QueueRow `json:"data"`
    Meta ListMeta   `json:"meta"`
}

type ListMeta struct {
    Page       int `json:"page"`
    PageSize   int `json:"pageSize"`
    TotalCount int `json:"totalCount"`
    TotalPages int `json:"totalPages"`
}

type AssignmentDetailDTO struct {
    ID             uuid.UUID         `json:"id"`
    Status         string            `json:"status"`
    Submission     SubmissionDTO     `json:"submission"`
    Rubric         RubricDTO         `json:"rubric"`
    ExistingScores []ScoreDTO        `json:"existingScores"`
}

type SubmissionDTO struct {
    ID          uuid.UUID `json:"id" db:"id"`
    Title       string    `json:"title" db:"title"`
    Description string    `json:"description" db:"description"`
    RepoURL     string    `json:"repoUrl" db:"repo_url"`
    DemoURL     string    `json:"demoUrl" db:"demo_url"`
    CoverURL    *string   `json:"coverUrl" db:"cover_url"`
    Team        TeamDTO   `json:"team" db:"team"`
}

type TeamDTO struct {
    ID   uuid.UUID `json:"id" db:"id"`
    Name string    `json:"name" db:"name"`
}

type RubricDTO struct {
    ID       uuid.UUID      `json:"id" db:"id"`
    Name     string         `json:"name" db:"name"`
    Criteria []CriterionDTO `json:"criteria"`
}

type CriterionDTO struct {
    ID          uuid.UUID `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Weight      float64   `json:"weight" db:"weight"`
    MaxScore    int       `json:"maxScore" db:"max_score"`
    Description string    `json:"description" db:"description"`
}

type ScoreDTO struct {
    CriterionID uuid.UUID `json:"criterionId" db:"criterion_id"`
    RawScore    int       `json:"rawScore" db:"raw_score"`
    Notes       string    `json:"notes" db:"notes"`
}
