package usecase

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/google/uuid"
)

type GetEventService struct {
	events port.EventRepository
}

func NewGetEventService(events port.EventRepository) *GetEventService {
	return &GetEventService{
		events: events,
	}
}

func (s *GetEventService) GetBySlug(ctx context.Context, slug string, callerID *uuid.UUID) (*port.EventDetailDTO, error) {
	event, tracks, myRole, err := s.events.GetEventDetail(ctx, slug, callerID)
	if err != nil {
		return nil, err
	}

	dto := port.EventDetailDTO{
		EventDTO: toEventDTO(event),
		Tracks:   make([]port.TrackDTO, len(tracks)),
		MyRole:   myRole,
	}

	for i, t := range tracks {
		dto.Tracks[i] = toTrackDTO(&t)
	}

	return &dto, nil
}
