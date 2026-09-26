package usecase

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type GetQueueService struct {
    assignments port.AssignmentRepository
}

func NewGetQueueService(assignments port.AssignmentRepository) *GetQueueService {
    return &GetQueueService{assignments: assignments}
}

func (s *GetQueueService) GetQueue(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) (*port.AssignmentListDTO, error) {
    if q.Page < 1 {
        q.Page = 1
    }
    if q.PageSize < 1 {
        q.PageSize = 20
    }

    rows, total, err := s.assignments.ListByJudge(ctx, judgeID, q)
    if err != nil {
        return nil, fmt.Errorf("failed to list queue: %w", err)
    }

    // Map to DTO
    var data []port.QueueRow
    for _, r := range rows {
        data = append(data, *r)
    }
    if data == nil {
        data = make([]port.QueueRow, 0)
    }

    totalPages := total / q.PageSize
    if total%q.PageSize != 0 {
        totalPages++
    }

    return &port.AssignmentListDTO{
        Data: data,
        Meta: port.ListMeta{
            Page:       q.Page,
            PageSize:   q.PageSize,
            TotalCount: total,
            TotalPages: totalPages,
        },
    }, nil
}
