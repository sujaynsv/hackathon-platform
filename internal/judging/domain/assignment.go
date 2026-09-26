package domain

import (
    "time"
    "github.com/google/uuid"
)

type AssignmentStatus string
const (
    AssignmentPending    AssignmentStatus = "pending"
    AssignmentInProgress AssignmentStatus = "in_progress"
    AssignmentCompleted  AssignmentStatus = "completed"
    AssignmentRecused    AssignmentStatus = "recused"
)

type JudgeAssignment struct {
    ID           uuid.UUID
    EventID      uuid.UUID
    JudgeID      uuid.UUID
    SubmissionID uuid.UUID
    RubricID     uuid.UUID
    Status       AssignmentStatus
    AssignedAt   time.Time
    CompletedAt  *time.Time
}
