package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/google/uuid"
)

const inviteCodeAttempts = 3

type CreateTeamService struct {
	events       port.EventReader
	participants port.ParticipantRepository
	teams        port.TeamRepository
}

func NewCreateTeamService(events port.EventReader, participants port.ParticipantRepository, teams port.TeamRepository) *CreateTeamService {
	return &CreateTeamService{events: events, participants: participants, teams: teams}
}

func (s *CreateTeamService) Create(ctx context.Context, cmd port.CreateTeamCommand) (*port.TeamDTO, error) {
	event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
	if err != nil {
		return nil, err
	}
	if event.Status != "registration_open" {
		return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrEventNotRegOpen.Error())
	}

	isParticipant, err := s.participants.HasRole(ctx, cmd.UserID, event.ID, "participant")
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, fmt.Errorf("%w: %s", response.ErrForbidden, domain.ErrNotParticipant.Error())
	}

	existingTeam, err := s.teams.FindByEventAndUser(ctx, event.ID, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if existingTeam != nil {
		return nil, fmt.Errorf("%w: %s", response.ErrDuplicate, domain.ErrAlreadyOnTeam.Error())
	}

	nameTaken, err := s.teams.ExistsByName(ctx, event.ID, cmd.TeamName)
	if err != nil {
		return nil, err
	}
	if nameTaken {
		return nil, fmt.Errorf("%w: %s", response.ErrDuplicate, domain.ErrTeamNameTaken.Error())
	}

	for attempt := 0; attempt < inviteCodeAttempts; attempt++ {
		inviteCode, err := domain.GenerateInviteCode()
		if err != nil {
			return nil, err
		}
		team, leader, err := domain.NewTeam(event.ID, cmd.UserID, cmd.TeamName, inviteCode, time.Now().UTC())
		if err != nil {
			return nil, err
		}

		created, err := s.teams.CreateWithLeader(ctx, team, leader)
		if errors.Is(err, domain.ErrInviteCodeTaken) {
			continue
		}
		if errors.Is(err, domain.ErrAlreadyOnTeam) || errors.Is(err, domain.ErrTeamNameTaken) {
			return nil, fmt.Errorf("%w: %s", response.ErrDuplicate, err.Error())
		}
		if err != nil {
			return nil, err
		}
		return toTeamDTO(created), nil
	}

	return nil, fmt.Errorf("%w: could not allocate a unique invite code", response.ErrDuplicate)
}

type GetMyTeamService struct {
	events port.EventReader
	teams  port.TeamRepository
}

func NewGetMyTeamService(events port.EventReader, teams port.TeamRepository) *GetMyTeamService {
	return &GetMyTeamService{events: events, teams: teams}
}

func (s *GetMyTeamService) GetMyTeam(ctx context.Context, userID uuid.UUID, eventSlug string) (*port.TeamDTO, error) {
	event, err := s.events.FindSummaryBySlug(ctx, eventSlug)
	if err != nil {
		return nil, err
	}

	team, err := s.teams.FindByEventAndUser(ctx, event.ID, userID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, response.ErrNotFound
	}
	return toTeamDTO(team), nil
}
