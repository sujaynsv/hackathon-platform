package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/google/uuid"
)

type CreateTeamService struct {
	events       port.EventReader
	participants port.ParticipantRepository
	teams        port.TeamRepository
}

func NewCreateTeamService(events port.EventReader, participants port.ParticipantRepository, teams port.TeamRepository) *CreateTeamService {
	return &CreateTeamService{events: events, participants: participants, teams: teams}
}

func (s *CreateTeamService) Create(ctx context.Context, cmd port.CreateTeamCommand) (*port.TeamDTO, error) {
	// 1. Load event
	event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
	if err != nil {
		return nil, err
	}

	// 2. Check I6: event must be registration_open
	if event.Status != "registration_open" {
		return nil, fmt.Errorf("%w: event is not open for registration", response.ErrInvariantViolated)
	}

	// 3. Check caller is a registered participant
	role, _ := s.participants.GetRole(ctx, cmd.UserID, event.ID)
	if role != "participant" {
		return nil, fmt.Errorf("%w: caller is not a participant", response.ErrForbidden)
	}

	// 4. I1: check user does not already have a team
	existing, _ := s.teams.FindByEventAndUser(ctx, event.ID, cmd.UserID)
	if existing != nil {
		return nil, fmt.Errorf("%w: %s", response.ErrDuplicate, domain.ErrAlreadyOnTeam.Error())
	}

	// 5. Check team name uniqueness
	nameTaken, _ := s.teams.ExistsByName(ctx, event.ID, cmd.TeamName)
	if nameTaken {
		return nil, fmt.Errorf("%w: team name '%s' already taken", response.ErrDuplicate, cmd.TeamName)
	}

	// 6. Generate invite code (retry on collision)
	var code string
	for i := 0; i < 3; i++ {
		code, err = domain.GenerateInviteCode()
		if err != nil {
			return nil, err
		}
		if existing, _ := s.teams.FindByInviteCode(ctx, code); existing == nil {
			break
		}
	}

	// 7. Create team
	now := time.Now().UTC()
	team := &domain.Team{
		ID:         uuid.New(),
		EventID:    event.ID,
		Name:       cmd.TeamName,
		InviteCode: code,
		CreatedBy:  cmd.UserID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.teams.Save(ctx, team); err != nil {
		return nil, err
	}

	// 8. Add creator as leader
	leader := &domain.TeamMember{
		ID:       uuid.New(),
		UserID:   cmd.UserID,
		TeamID:   team.ID,
		EventID:  event.ID,
		Role:     domain.RoleOwner, // from Issue #19 definition (wait, domain uses owner, issue used leader. The issue mentions "leader". We use RoleOwner)
		JoinedAt: now,
	}
	if err := s.teams.AddMember(ctx, leader); err != nil {
		return nil, err
	}

	return toTeamDTO(team, []domain.TeamMember{*leader}), nil
}

func toTeamDTO(team *domain.Team, members []domain.TeamMember) *port.TeamDTO {
	dto := &port.TeamDTO{
		ID:         team.ID.String(),
		EventID:    team.EventID.String(),
		Name:       team.Name,
		InviteCode: team.InviteCode,
		IsLocked:   team.IsLocked,
		CreatedAt:  team.CreatedAt.Format(time.RFC3339),
		Members:    make([]port.TeamMemberDTO, 0, len(members)),
	}

	for _, m := range members {
		dto.Members = append(dto.Members, port.TeamMemberDTO{
			ID:       m.ID.String(),
			UserID:   m.UserID.String(),
			Role:     string(m.Role),
			JoinedAt: m.JoinedAt.Format(time.RFC3339),
		})
	}
	return dto
}
