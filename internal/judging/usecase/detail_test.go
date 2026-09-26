package usecase_test

import (
    "context"
    "errors"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/judging/usecase"
    "github.com/dogfood-platform/dogfood/internal/shared/response"
)

type mockAssignmentRepoDetail struct {
    findBy func(id, judgeID uuid.UUID) (*domain.JudgeAssignment, error)
}
func (m *mockAssignmentRepoDetail) Save(ctx context.Context, a *domain.JudgeAssignment) error { return nil }
func (m *mockAssignmentRepoDetail) BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error { return nil }
func (m *mockAssignmentRepoDetail) ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) { return nil, 0, nil }
func (m *mockAssignmentRepoDetail) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error { return nil }
func (m *mockAssignmentRepoDetail) FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error) {
    if m.findBy != nil {
        return m.findBy(id, judgeID)
    }
    return nil, nil
}

type mockSubReader struct {
    sub *port.SubmissionDTO
}
func (m *mockSubReader) FindByID(ctx context.Context, id uuid.UUID) (*port.SubmissionDTO, error) {
    return m.sub, nil
}

type mockRubricReader struct {
    rubric *port.RubricDTO
}
func (m *mockRubricReader) FindByID(ctx context.Context, id uuid.UUID) (*port.RubricDTO, error) {
    return m.rubric, nil
}

type mockScoreReader struct {
    scores []port.ScoreDTO
}
func (m *mockScoreReader) ListByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]port.ScoreDTO, error) {
    return m.scores, nil
}

func TestGetAssignmentDetailService_WrongJudge_Returns404(t *testing.T) {
    // I16 - Returns 404 when judge is wrong
    mockRepo := &mockAssignmentRepoDetail{
        findBy: func(id, judgeID uuid.UUID) (*domain.JudgeAssignment, error) {
            return nil, response.ErrNotFound
        },
    }

    svc := usecase.NewGetDetailService(mockRepo, nil, nil, nil)
    _, err := svc.GetDetail(context.Background(), uuid.New(), uuid.New())
    
    assert.Error(t, err)
    assert.True(t, errors.Is(err, response.ErrNotFound))
}

func TestGetAssignmentDetailService_CorrectJudge_ReturnsFullDetail(t *testing.T) {
    assignmentID := uuid.New()
    judgeID := uuid.New()
    
    mockRepo := &mockAssignmentRepoDetail{
        findBy: func(id, jid uuid.UUID) (*domain.JudgeAssignment, error) {
            assert.Equal(t, assignmentID, id)
            assert.Equal(t, judgeID, jid)
            return &domain.JudgeAssignment{
                ID: assignmentID,
                JudgeID: judgeID,
                Status: domain.AssignmentInProgress,
            }, nil
        },
    }

    sub := &port.SubmissionDTO{ID: uuid.New(), Title: "AI App"}
    rubric := &port.RubricDTO{ID: uuid.New(), Name: "Standard"}
    scores := []port.ScoreDTO{{CriterionID: uuid.New(), RawScore: 9}}

    svc := usecase.NewGetDetailService(
        mockRepo, 
        &mockSubReader{sub: sub},
        &mockRubricReader{rubric: rubric},
        &mockScoreReader{scores: scores},
    )

    dto, err := svc.GetDetail(context.Background(), judgeID, assignmentID)
    
    assert.NoError(t, err)
    assert.Equal(t, assignmentID, dto.ID)
    assert.Equal(t, "in_progress", dto.Status)
    assert.Equal(t, sub.Title, dto.Submission.Title)
    assert.Equal(t, rubric.Name, dto.Rubric.Name)
    assert.Len(t, dto.ExistingScores, 1)
    assert.Equal(t, 9, dto.ExistingScores[0].RawScore)
}
