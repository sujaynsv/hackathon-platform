package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
)

type RegisterService struct {
	events       port.EventReader
	participants port.ParticipantRepository
}

func NewRegisterService(events port.EventReader, participants port.ParticipantRepository) *RegisterService {
	return &RegisterService{events: events, participants: participants}
}

func (s *RegisterService) Register(ctx context.Context, cmd port.RegisterCommand) (*port.RegistrationDTO, error) {
	event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
	if err != nil {
		return nil, err
	}

	if err := domain.CheckCanRegister(event.Status, event.RegistrationClosesAt); err != nil {
		if errors.Is(err, domain.ErrDeadlinePassed) {
			return nil, fmt.Errorf("%w: %s", response.ErrDeadlinePassed, err.Error())
		}
		return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
	}

	for _, role := range []string{"judge", "organizer"} {
		hasRole, err := s.participants.HasRole(ctx, cmd.UserID, event.ID, role)
		if err != nil {
			return nil, err
		}
		if hasRole {
			return nil, fmt.Errorf("%w: user already has role '%s' in this event", response.ErrInvariantViolated, role)
		}
	}

	registered, err := s.participants.HasRole(ctx, cmd.UserID, event.ID, "participant")
	if err != nil {
		return nil, err
	}
	if registered {
		return nil, fmt.Errorf("%w: already registered", response.ErrDuplicate)
	}

	registeredAt, err := s.participants.GrantParticipantRole(ctx, cmd.UserID, event.ID)
	if err != nil {
		return nil, err
	}

	return &port.RegistrationDTO{
		Registered:   true,
		EventID:      event.ID.String(),
		EventSlug:    event.Slug,
		RegisteredAt: registeredAt.UTC().Format(time.RFC3339),
	}, nil
}

type UnregisterService struct {
	events       port.EventReader
	participants port.ParticipantRepository
	teamMembers  port.TeamMemberRepository
}

func NewUnregisterService(events port.EventReader, participants port.ParticipantRepository, teamMembers port.TeamMemberRepository) *UnregisterService {
	return &UnregisterService{events: events, participants: participants, teamMembers: teamMembers}
}

func (s *UnregisterService) Unregister(ctx context.Context, cmd port.UnregisterCommand) error {
	event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
	if err != nil {
		return err
	}

	registered, err := s.participants.HasRole(ctx, cmd.UserID, event.ID, "participant")
	if err != nil {
		return err
	}
	if !registered {
		return fmt.Errorf("%w: user is not registered for this event", response.ErrNotFound)
	}

	hasTeam, err := s.teamMembers.HasTeamInEvent(ctx, cmd.UserID, event.ID)
	if err != nil {
		return err
	}
	if hasTeam {
		return fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrAlreadyHasTeam.Error())
	}

	return s.participants.RevokeParticipantRole(ctx, cmd.UserID, event.ID)
}
