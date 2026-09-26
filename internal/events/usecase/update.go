package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
)

type UpdateEventService struct {
	events port.EventRepository
	roles  port.EventRoleRepository
	tracks port.TrackRepository
}

func NewUpdateEventService(events port.EventRepository, roles port.EventRoleRepository, tracks port.TrackRepository) *UpdateEventService {
	return &UpdateEventService{events: events, roles: roles, tracks: tracks}
}

func (s *UpdateEventService) Update(ctx context.Context, cmd port.UpdateEventCommand) (*port.EventDetailDTO, error) {
	// 1. Load event
	event, err := s.events.FindBySlug(ctx, cmd.Slug)
	if err != nil {
		return nil, err // ErrNotFound wrapped or propagated
	}

	// 2. Check organizer role
	role, err := s.roles.GetRole(ctx, cmd.CallerID, event.ID)
	if err != nil || role != "organizer" {
		return nil, fmt.Errorf("%w: only organizers can update events", response.ErrForbidden)
	}

	// 3. Apply status transition first
	if cmd.NewStatus != nil {
		if err := event.Transition(domain.EventStatus(*cmd.NewStatus)); err != nil {
			return nil, fmt.Errorf("%w", response.ErrInvalidTransition)
		}
	}

	// 4. Apply field updates
	if cmd.MaxTeamSize != nil {
		if event.Status != domain.StatusDraft && event.Status != domain.StatusRegistrationOpen {
			return nil, fmt.Errorf("%w: maxTeamSize cannot be changed after registration closes", response.ErrInvariantViolated)
		}
		event.MaxTeamSize = *cmd.MaxTeamSize
	}
	if cmd.Title != nil {
		event.Title = *cmd.Title
	}
	if cmd.Description != nil {
		event.Description = cmd.Description
	}
	if cmd.RegistrationOpensAt != nil {
		event.RegistrationOpensAt = cmd.RegistrationOpensAt
	}
	if cmd.RegistrationClosesAt != nil {
		event.RegistrationClosesAt = cmd.RegistrationClosesAt
	}
	if cmd.SubmissionDeadlineAt != nil {
		event.SubmissionDeadlineAt = cmd.SubmissionDeadlineAt
	}
	if cmd.JudgingDeadlineAt != nil {
		event.JudgingDeadlineAt = cmd.JudgingDeadlineAt
	}
	if cmd.VotingOpensAt != nil {
		event.VotingOpensAt = cmd.VotingOpensAt
	}
	if cmd.VotingClosesAt != nil {
		event.VotingClosesAt = cmd.VotingClosesAt
	}

	event.UpdatedAt = time.Now().UTC()

	// 5. Persist
	if err := s.events.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	// 6. Load tracks for response
	tracks, _ := s.tracks.FindByEventID(ctx, event.ID)
	
	dto := port.EventDetailDTO{
		EventDTO: toEventDTO(event),
		Tracks:   make([]port.TrackDTO, len(tracks)),
	}

	myRole := "organizer"
	dto.MyRole = &myRole

	for i, t := range tracks {
		dto.Tracks[i] = toTrackDTO(t)
	}

	return &dto, nil
}
