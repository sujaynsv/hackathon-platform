package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/dogfood-platform/dogfood/internal/teams/usecase"
	"github.com/google/uuid"
)

type teamEventReaderStub struct {
	event *port.EventSummary
	err   error
}

func (r teamEventReaderStub) FindSummaryBySlug(context.Context, string) (*port.EventSummary, error) {
	return r.event, r.err
}

type teamParticipantsStub struct {
	isParticipant bool
	err           error
}

func (r teamParticipantsStub) HasRole(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return r.isParticipant, r.err
}

func (r teamParticipantsStub) GrantParticipantRole(context.Context, uuid.UUID, uuid.UUID) (time.Time, error) {
	return time.Time{}, nil
}

func (r teamParticipantsStub) RevokeParticipantRole(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type teamRepositoryStub struct {
	existingTeam *domain.Team
	findErr      error
	nameTaken    bool
	nameErr      error
	createErr    error
	createdTeam  *domain.Team
}

func (r *teamRepositoryStub) CreateWithLeader(_ context.Context, team *domain.Team, leader *domain.TeamMember) (*domain.Team, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	leader.DisplayName = "Test Participant"
	team.Members = []domain.TeamMember{*leader}
	r.createdTeam = team
	return team, nil
}

func (r *teamRepositoryStub) FindByEventAndUser(context.Context, uuid.UUID, uuid.UUID) (*domain.Team, error) {
	return r.existingTeam, r.findErr
}

func (r *teamRepositoryStub) ExistsByName(context.Context, uuid.UUID, string) (bool, error) {
	return r.nameTaken, r.nameErr
}

func teamUseCaseFixture(status string, isParticipant bool) (*port.EventSummary, *teamParticipantsStub, *teamRepositoryStub) {
	return &port.EventSummary{
		ID:     uuid.New(),
		Slug:   "t002-test-event",
		Status: status,
	}, &teamParticipantsStub{isParticipant: isParticipant}, &teamRepositoryStub{}
}

func TestCreateTeamService_ValidInputReturnsTeamWithOwner(t *testing.T) {
	event, participants, teams := teamUseCaseFixture("registration_open", true)
	service := usecase.NewCreateTeamService(teamEventReaderStub{event: event}, participants, teams)

	result, err := service.Create(context.Background(), port.CreateTeamCommand{
		UserID:    uuid.New(),
		EventSlug: event.Slug,
		TeamName:  "Team Rocket",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if result.Name != "Team Rocket" || len(result.InviteCode) != 8 {
		t.Fatalf("unexpected created team: %+v", result)
	}
	if len(result.Members) != 1 || result.Members[0].Role != domain.TeamOwnerRole || result.Members[0].DisplayName != "Test Participant" {
		t.Fatalf("unexpected owner member: %+v", result.Members)
	}
}

func TestCreateTeamService_NotParticipantReturnsForbidden(t *testing.T) {
	event, participants, teams := teamUseCaseFixture("registration_open", false)
	service := usecase.NewCreateTeamService(teamEventReaderStub{event: event}, participants, teams)

	_, err := service.Create(context.Background(), port.CreateTeamCommand{UserID: uuid.New(), EventSlug: event.Slug, TeamName: "Team Rocket"})
	if !errors.Is(err, response.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestCreateTeamService_UserAlreadyHasTeamReturnsDuplicate(t *testing.T) {
	event, participants, teams := teamUseCaseFixture("registration_open", true)
	teams.existingTeam = &domain.Team{ID: uuid.New()}
	service := usecase.NewCreateTeamService(teamEventReaderStub{event: event}, participants, teams)

	_, err := service.Create(context.Background(), port.CreateTeamCommand{UserID: uuid.New(), EventSlug: event.Slug, TeamName: "Team Rocket"})
	if !errors.Is(err, response.ErrDuplicate) {
		t.Fatalf("error = %v, want ErrDuplicate", err)
	}
}

func TestCreateTeamService_EventNotRegistrationOpenReturnsInvariant(t *testing.T) {
	event, participants, teams := teamUseCaseFixture("draft", true)
	service := usecase.NewCreateTeamService(teamEventReaderStub{event: event}, participants, teams)

	_, err := service.Create(context.Background(), port.CreateTeamCommand{UserID: uuid.New(), EventSlug: event.Slug, TeamName: "Team Rocket"})
	if !errors.Is(err, response.ErrInvariantViolated) {
		t.Fatalf("error = %v, want ErrInvariantViolated", err)
	}
}

func TestCreateTeamService_DuplicateNameReturnsDuplicate(t *testing.T) {
	event, participants, teams := teamUseCaseFixture("registration_open", true)
	teams.nameTaken = true
	service := usecase.NewCreateTeamService(teamEventReaderStub{event: event}, participants, teams)

	_, err := service.Create(context.Background(), port.CreateTeamCommand{UserID: uuid.New(), EventSlug: event.Slug, TeamName: "Team Rocket"})
	if !errors.Is(err, response.ErrDuplicate) {
		t.Fatalf("error = %v, want ErrDuplicate", err)
	}
}

func TestGetMyTeamService_NoTeamReturnsNotFound(t *testing.T) {
	event, _, teams := teamUseCaseFixture("registration_open", true)
	service := usecase.NewGetMyTeamService(teamEventReaderStub{event: event}, teams)

	_, err := service.GetMyTeam(context.Background(), uuid.New(), event.Slug)
	if !errors.Is(err, response.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetMyTeamService_ReturnsMyTeam(t *testing.T) {
	event, _, teams := teamUseCaseFixture("registration_open", true)
	teams.existingTeam = &domain.Team{
		ID:         uuid.New(),
		EventID:    event.ID,
		Name:       "Team Rocket",
		InviteCode: "HACK2026",
		Members: []domain.TeamMember{{
			UserID:      uuid.New(),
			DisplayName: "Test Participant",
			Role:        domain.TeamOwnerRole,
		}},
	}
	service := usecase.NewGetMyTeamService(teamEventReaderStub{event: event}, teams)

	result, err := service.GetMyTeam(context.Background(), uuid.New(), event.Slug)
	if err != nil {
		t.Fatalf("GetMyTeam returned error: %v", err)
	}
	if result.Name != teams.existingTeam.Name || result.InviteCode != teams.existingTeam.InviteCode {
		t.Fatalf("unexpected team DTO: %+v", result)
	}
}
