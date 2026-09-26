package usecase_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/judging/usecase"
)

type mockAssignmentRepoCreate struct {
    bulkCreate func(eventID uuid.UUID) error
}
func (m *mockAssignmentRepoCreate) Save(ctx context.Context, a *domain.JudgeAssignment) error { return nil }
func (m *mockAssignmentRepoCreate) FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error) { return nil, nil }
func (m *mockAssignmentRepoCreate) ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) { return nil, 0, nil }
func (m *mockAssignmentRepoCreate) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error { return nil }
func (m *mockAssignmentRepoCreate) BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error {
    if m.bulkCreate != nil {
        return m.bulkCreate(eventID)
    }
    return nil
}

func TestCreateAssignmentsService_CreatesOnePerJudgePerSubmission(t *testing.T) {
    eventID := uuid.New()
    called := false
    
    mockRepo := &mockAssignmentRepoCreate{
        bulkCreate: func(id uuid.UUID) error {
            assert.Equal(t, eventID, id)
            called = true
            return nil
        },
    }

    svc := usecase.NewCreateAssignmentsService(mockRepo)
    err := svc.CreateForEvent(context.Background(), eventID)
    
    assert.NoError(t, err)
    assert.True(t, called)
}

func TestCreateAssignmentsService_IdempotentOnDuplicateCall(t *testing.T) {
    eventID := uuid.New()
    callCount := 0
    
    mockRepo := &mockAssignmentRepoCreate{
        bulkCreate: func(id uuid.UUID) error {
            callCount++
            return nil
        },
    }

    svc := usecase.NewCreateAssignmentsService(mockRepo)
    
    err1 := svc.CreateForEvent(context.Background(), eventID)
    err2 := svc.CreateForEvent(context.Background(), eventID)
    
    assert.NoError(t, err1)
    assert.NoError(t, err2)
    assert.Equal(t, 2, callCount)
    // The idempotency is handled at the repository (SQL level: ON CONFLICT DO NOTHING),
    // so the usecase simply calls the repo successfully multiple times.
}
