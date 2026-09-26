package usecase

import (
	"context"
	"fmt"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
)

type CreateEventService struct {
	events port.EventRepository
	roles  port.EventRoleRepository
}

func NewCreateEventService(events port.EventRepository, roles port.EventRoleRepository) *CreateEventService {
	return &CreateEventService{
		events: events,
		roles:  roles,
	}
}

func (s *CreateEventService) Create(ctx context.Context, cmd port.CreateEventCommand) (*port.EventDTO, error) {
	// 1. Check slug uniqueness
	exists, err := s.events.ExistsBySlug(ctx, cmd.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: slug '%s' is already taken", response.ErrInvariantViolated, cmd.Slug)
	}

	// 2. Create domain event
	event, err := domain.NewEvent(cmd.Slug, cmd.Title, cmd.OrganizerID, cmd.MaxTeamSize)
	if err != nil {
		return nil, err
	}
	// Set optional date fields
	event.Description = cmd.Description
	event.RegistrationOpensAt = cmd.RegistrationOpensAt
	event.RegistrationClosesAt = cmd.RegistrationClosesAt
	if cmd.SubmissionDeadlineAt != nil {
		event.SubmissionDeadlineAt = cmd.SubmissionDeadlineAt
	}
	if cmd.JudgingDeadlineAt != nil {
		event.JudgingDeadlineAt = cmd.JudgingDeadlineAt
	}
	event.VotingOpensAt = cmd.VotingOpensAt
	event.VotingClosesAt = cmd.VotingClosesAt

	// 3. Persist
	if err := s.events.Save(ctx, event); err != nil {
		return nil, fmt.Errorf("save event: %w", err)
	}

	// 4. Grant organizer role to creator
	if err := s.roles.GrantRole(ctx, cmd.OrganizerID, event.ID, "organizer"); err != nil {
		return nil, fmt.Errorf("grant organizer role: %w", err)
	}

	dto := toEventDTO(event)
	return &dto, nil
}
