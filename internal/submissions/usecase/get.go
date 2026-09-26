package usecase

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/google/uuid"
)

type GetSubmissionService struct {
	subs port.SubmissionRepository
}

func NewGetSubmissionService(subs port.SubmissionRepository) *GetSubmissionService {
	return &GetSubmissionService{subs: subs}
}

func (s *GetSubmissionService) Get(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	return s.subs.FindByID(ctx, id)
}
