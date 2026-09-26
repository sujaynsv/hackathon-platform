package usecase_test

import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/judging/usecase"
    "github.com/dogfood-platform/dogfood/internal/shared/response"
)

type mockScoreWriter struct {
    upsertAll func(ctx context.Context, assignmentID, judgeID uuid.UUID, scores []port.ScoreInput) error
}

func (m *mockScoreWriter) UpsertAll(ctx context.Context, assignmentID, judgeID uuid.UUID, scores []port.ScoreInput) error {
    if m.upsertAll != nil {
        return m.upsertAll(ctx, assignmentID, judgeID, scores)
    }
    return nil
}

type mockEventReader struct {
    findSummaryByID func(ctx context.Context, id uuid.UUID) (*port.EventSummary, error)
}

func (m *mockEventReader) FindSummaryByID(ctx context.Context, id uuid.UUID) (*port.EventSummary, error) {
    if m.findSummaryByID != nil {
        return m.findSummaryByID(ctx, id)
    }
    return &port.EventSummary{}, nil
}

type mockRubricReaderForScores struct {
    findCriteriaByRubricID func(ctx context.Context, rubricID uuid.UUID) ([]port.CriterionDTO, error)
}
func (m *mockRubricReaderForScores) FindByID(ctx context.Context, id uuid.UUID) (*port.RubricDTO, error) {
    return nil, nil
}
func (m *mockRubricReaderForScores) FindCriteriaByRubricID(ctx context.Context, rubricID uuid.UUID) ([]port.CriterionDTO, error) {
    if m.findCriteriaByRubricID != nil {
        return m.findCriteriaByRubricID(ctx, rubricID)
    }
    return []port.CriterionDTO{}, nil
}

type mockAssignmentRepoScores struct {
    findByIDAndJudge func(ctx context.Context, id, jid uuid.UUID) (*domain.JudgeAssignment, error)
    updateStatus     func(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error
}
func (m *mockAssignmentRepoScores) Save(ctx context.Context, a *domain.JudgeAssignment) error { return nil }
func (m *mockAssignmentRepoScores) BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error { return nil }
func (m *mockAssignmentRepoScores) ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) { return nil, 0, nil }
func (m *mockAssignmentRepoScores) FindByIDAndJudge(ctx context.Context, id, jid uuid.UUID) (*domain.JudgeAssignment, error) {
    if m.findByIDAndJudge != nil {
        return m.findByIDAndJudge(ctx, id, jid)
    }
    return nil, nil
}
func (m *mockAssignmentRepoScores) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error {
    if m.updateStatus != nil {
        return m.updateStatus(ctx, id, status)
    }
    return nil
}

func TestSubmitScoresService_ValidScores_Returns200_StatusCompleted(t *testing.T) {
    judgeID := uuid.New()
    assignmentID := uuid.New()
    eventID := uuid.New()
    rubricID := uuid.New()
    crit1 := uuid.New()
    crit2 := uuid.New()

    mockRepo := &mockAssignmentRepoScores{
        findByIDAndJudge: func(ctx context.Context, id, jid uuid.UUID) (*domain.JudgeAssignment, error) {
            return &domain.JudgeAssignment{
                ID:       id,
                EventID:  eventID,
                JudgeID:  jid,
                RubricID: rubricID,
            }, nil
        },
        updateStatus: func(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error {
            assert.Equal(t, domain.AssignmentCompleted, status)
            return nil
        },
    }

    mockEvents := &mockEventReader{
        findSummaryByID: func(ctx context.Context, id uuid.UUID) (*port.EventSummary, error) {
            deadline := time.Now().UTC().Add(time.Hour)
            return &port.EventSummary{ID: eventID, JudgingDeadlineAt: &deadline}, nil
        },
    }

    mockRubrics := &mockRubricReaderForScores{
        findCriteriaByRubricID: func(ctx context.Context, rid uuid.UUID) ([]port.CriterionDTO, error) {
            return []port.CriterionDTO{
                {ID: crit1, Name: "A", MaxScore: 10, Weight: 0.5},
                {ID: crit2, Name: "B", MaxScore: 10, Weight: 0.5},
            }, nil
        },
    }

    var upsertCalled bool
    mockScores := &mockScoreWriter{
        upsertAll: func(ctx context.Context, aid, jid uuid.UUID, scores []port.ScoreInput) error {
            upsertCalled = true
            assert.Equal(t, aid, assignmentID)
            assert.Equal(t, jid, judgeID)
            return nil
        },
    }

    svc := usecase.NewSubmitScoresService(mockRepo, mockEvents, mockRubrics, mockScores)

    cmd := port.SubmitScoresCommand{
        JudgeID:      judgeID,
        AssignmentID: assignmentID,
        Scores: []port.ScoreInput{
            {CriterionID: crit1, RawScore: 8},
            {CriterionID: crit2, RawScore: 9},
        },
    }

    res, err := svc.SubmitScores(context.Background(), cmd)
    require.NoError(t, err)
    assert.NotNil(t, res)
    assert.True(t, upsertCalled)
    assert.Equal(t, "completed", res.Status)
    assert.InDelta(t, 8.5, res.WeightedTotal, 0.001)
}

func TestSubmitScoresService_WrongJudge_Returns404(t *testing.T) {
    mockRepo := &mockAssignmentRepoScores{
        findByIDAndJudge: func(ctx context.Context, id, jid uuid.UUID) (*domain.JudgeAssignment, error) {
            return nil, nil // Not found
        },
    }

    svc := usecase.NewSubmitScoresService(mockRepo, nil, nil, nil)
    _, err := svc.SubmitScores(context.Background(), port.SubmitScoresCommand{})
    assert.ErrorIs(t, err, response.ErrNotFound)
}

func TestSubmitScoresService_PastDeadline_Returns422(t *testing.T) {
    mockRepo := &mockAssignmentRepoScores{
        findByIDAndJudge: func(ctx context.Context, id, jid uuid.UUID) (*domain.JudgeAssignment, error) {
            return &domain.JudgeAssignment{}, nil
        },
    }

    mockEvents := &mockEventReader{
        findSummaryByID: func(ctx context.Context, id uuid.UUID) (*port.EventSummary, error) {
            deadline := time.Now().UTC().Add(-time.Hour) // Past
            return &port.EventSummary{JudgingDeadlineAt: &deadline}, nil
        },
    }

    svc := usecase.NewSubmitScoresService(mockRepo, mockEvents, nil, nil)
    _, err := svc.SubmitScores(context.Background(), port.SubmitScoresCommand{})
    assert.ErrorIs(t, err, response.ErrDeadlinePassed)
}
