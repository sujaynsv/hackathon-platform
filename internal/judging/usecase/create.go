package usecase

import (
    "context"

    "github.com/google/uuid"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type CreateAssignmentsService struct {
    assignments port.AssignmentRepository
}

func NewCreateAssignmentsService(assignments port.AssignmentRepository) *CreateAssignmentsService {
    return &CreateAssignmentsService{assignments: assignments}
}

func (s *CreateAssignmentsService) CreateForEvent(ctx context.Context, eventID uuid.UUID) error {
    return s.assignments.BulkCreateForEvent(ctx, eventID)
}
