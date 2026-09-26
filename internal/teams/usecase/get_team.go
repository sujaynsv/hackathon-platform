package usecase

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
)

type GetMyTeamService struct {
	events port.EventReader
	teams  port.TeamRepository
}

func NewGetMyTeamService(events port.EventReader, teams port.TeamRepository) *GetMyTeamService {
	return &GetMyTeamService{events: events, teams: teams}
}

func (s *GetMyTeamService) GetMyTeam(ctx context.Context, query port.GetMyTeamQuery) (*port.TeamDTO, error) {
	event, err := s.events.FindSummaryBySlug(ctx, query.EventSlug)
	if err != nil {
		return nil, err
	}

	team, err := s.teams.FindByEventAndUser(ctx, event.ID, query.UserID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, response.ErrNotFound
	}

	members, err := s.teams.GetMembers(ctx, team.ID)
	if err != nil {
		return nil, err
	}

	return toTeamDTO(team, members), nil
}
