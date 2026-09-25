package usecase

import (
	"context"
	"math"

	"github.com/dogfood-platform/dogfood/internal/events/port"
)

type ListEventsService struct {
	events port.EventRepository
}

func NewListEventsService(events port.EventRepository) *ListEventsService {
	return &ListEventsService{
		events: events,
	}
}

func (s *ListEventsService) List(ctx context.Context, q port.ListEventsQuery) (*port.EventListDTO, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	if q.PageSize > 50 {
		q.PageSize = 50
	}

	events, totalCount, err := s.events.ListPublished(ctx, q.Page, q.PageSize)
	if err != nil {
		return nil, err
	}

	dtos := make([]port.EventDTO, len(events))
	for i, e := range events {
		dtos[i] = toEventDTO(e)
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(q.PageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &port.EventListDTO{
		Events:     dtos,
		Page:       q.Page,
		PageSize:   q.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}
