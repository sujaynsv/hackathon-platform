package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/dogfood-platform/dogfood/internal/teams/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetMyTeamService_NotFound(t *testing.T) {
	events := &mockEventReader{summary: &port.EventSummary{ID: uuid.New()}}
	teams := &mockTeamRepository{}
	svc := usecase.NewGetMyTeamService(events, teams)

	_, err := svc.GetMyTeam(context.Background(), port.GetMyTeamQuery{
		UserID:    uuid.New(),
		EventSlug: "test",
	})
	assert.ErrorIs(t, err, response.ErrNotFound)
}

func TestGetMyTeamService_Success(t *testing.T) {
	eventID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	events := &mockEventReader{summary: &port.EventSummary{ID: eventID}}
	teams := &mockTeamRepository{
		teams: map[uuid.UUID]*domain.Team{
			teamID: {
				ID:        teamID,
				EventID:   eventID,
				Name:      "My Team",
				CreatedBy: userID,
			},
		},
		members: map[uuid.UUID][]domain.TeamMember{
			teamID: {
				{
					ID:       uuid.New(),
					TeamID:   teamID,
					UserID:   userID,
					EventID:  eventID,
					Role:     domain.RoleOwner,
					JoinedAt: time.Now(),
				},
			},
		},
	}
	svc := usecase.NewGetMyTeamService(events, teams)

	dto, err := svc.GetMyTeam(context.Background(), port.GetMyTeamQuery{
		UserID:    userID,
		EventSlug: "test",
	})
	assert.NoError(t, err)
	assert.Equal(t, "My Team", dto.Name)
	assert.Len(t, dto.Members, 1)
}
