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

type mockAssignmentRepo struct {
    rows   []*port.QueueRow
    total  int
    listBy func(judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error)
}

func (m *mockAssignmentRepo) Save(ctx context.Context, a *domain.JudgeAssignment) error { return nil }
func (m *mockAssignmentRepo) BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error { return nil }
func (m *mockAssignmentRepo) FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error) { return nil, nil }
func (m *mockAssignmentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error { return nil }
func (m *mockAssignmentRepo) ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) {
    if m.listBy != nil {
        return m.listBy(judgeID, q)
    }
    return m.rows, m.total, nil
}

func TestGetQueueService_OnlyReturnsCallerAssignments(t *testing.T) {
    judgeID := uuid.New()
    mockRepo := &mockAssignmentRepo{
        listBy: func(jid uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) {
            assert.Equal(t, judgeID, jid, "should query with caller's judge ID")
            return []*port.QueueRow{{AssignmentID: uuid.New()}}, 1, nil
        },
    }

    svc := usecase.NewGetQueueService(mockRepo)
    dto, err := svc.GetQueue(context.Background(), judgeID, port.QueueQuery{})
    
    assert.NoError(t, err)
    assert.Len(t, dto.Data, 1)
}

func TestGetQueueService_FilterByStatus_PendingOnly(t *testing.T) {
    judgeID := uuid.New()
    mockRepo := &mockAssignmentRepo{
        listBy: func(jid uuid.UUID, q port.QueueQuery) ([]*port.QueueRow, int, error) {
            assert.Equal(t, "pending", q.Status, "should pass status filter to repo")
            return []*port.QueueRow{{AssignmentID: uuid.New(), Status: "pending"}}, 1, nil
        },
    }

    svc := usecase.NewGetQueueService(mockRepo)
    dto, err := svc.GetQueue(context.Background(), judgeID, port.QueueQuery{Status: "pending"})
    
    assert.NoError(t, err)
    assert.Len(t, dto.Data, 1)
    assert.Equal(t, domain.AssignmentPending, dto.Data[0].Status)
}
