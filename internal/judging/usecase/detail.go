package usecase

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/shared/response"
)

type GetDetailService struct {
    assignments port.AssignmentRepository
    submissions port.SubmissionReader
    rubrics     port.RubricReader
    scores      port.ScoreReader
}

func NewGetDetailService(a port.AssignmentRepository, s port.SubmissionReader, r port.RubricReader, sc port.ScoreReader) *GetDetailService {
    return &GetDetailService{
        assignments: a,
        submissions: s,
        rubrics:     r,
        scores:      sc,
    }
}

func (s *GetDetailService) GetDetail(ctx context.Context, judgeID, assignmentID uuid.UUID) (*port.AssignmentDetailDTO, error) {
    // FindByIDAndJudge — returns nil if judge doesn't own this assignment (I16)
    a, err := s.assignments.FindByIDAndJudge(ctx, assignmentID, judgeID)
    if err != nil || a == nil {
        return nil, fmt.Errorf("%w: assignment not found", response.ErrNotFound)
    }

    sub, err := s.submissions.FindByID(ctx, a.SubmissionID)
    if err != nil {
        return nil, fmt.Errorf("failed to load submission: %w", err)
    }

    rubric, err := s.rubrics.FindByID(ctx, a.RubricID)
    if err != nil {
        return nil, fmt.Errorf("failed to load rubric: %w", err)
    }

    scores, err := s.scores.ListByAssignment(ctx, a.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to load scores: %w", err)
    }
    if scores == nil {
        scores = []port.ScoreDTO{}
    }

    return &port.AssignmentDetailDTO{
        ID:             a.ID,
        Status:         string(a.Status),
        Submission:     *sub,
        Rubric:         *rubric,
        ExistingScores: scores,
    }, nil
}
