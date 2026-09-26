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

type mockTeamRepository struct {
	teams      map[uuid.UUID]*domain.Team
	members    map[uuid.UUID][]domain.TeamMember
	existsName bool
	findErr    error
	saveErr    error
}

func (m *mockTeamRepository) FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*domain.Team, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, team := range m.teams {
		if team.EventID == eventID && team.CreatedBy == userID {
			return team, nil
		}
	}
	return nil, nil
}

func (m *mockTeamRepository) ExistsByName(ctx context.Context, eventID uuid.UUID, name string) (bool, error) {
	return m.existsName, nil
}

func (m *mockTeamRepository) FindByInviteCode(ctx context.Context, code string) (*domain.Team, error) {
	return nil, nil
}

func (m *mockTeamRepository) Save(ctx context.Context, team *domain.Team) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if m.teams == nil {
		m.teams = make(map[uuid.UUID]*domain.Team)
	}
	m.teams[team.ID] = team
	return nil
}

func (m *mockTeamRepository) AddMember(ctx context.Context, member *domain.TeamMember) error {
	if m.members == nil {
		m.members = make(map[uuid.UUID][]domain.TeamMember)
	}
	m.members[member.TeamID] = append(m.members[member.TeamID], *member)
	return nil
}

func (m *mockTeamRepository) GetMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamMember, error) {
	return m.members[teamID], nil
}

type mockEventReader struct {
	summary *port.EventSummary
	err     error
}

func (m *mockEventReader) FindSummaryBySlug(ctx context.Context, slug string) (*port.EventSummary, error) {
	return m.summary, m.err
}

type mockParticipants struct {
	role string
	err  error
}

func (m *mockParticipants) HasRole(ctx context.Context, userID, eventID uuid.UUID, role string) (bool, error) {
	return m.role == role, nil
}

func (m *mockParticipants) GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error) {
	return m.role, m.err
}

func (m *mockParticipants) GrantParticipantRole(ctx context.Context, userID, eventID uuid.UUID) (time.Time, error) {
	return time.Now(), nil
}

func (m *mockParticipants) RevokeParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error {
	return nil
}

func TestCreateTeamService_CannotCreateWhenNotOpen(t *testing.T) {
	events := &mockEventReader{summary: &port.EventSummary{Status: "draft"}}
	participants := &mockParticipants{role: "participant"}
	teams := &mockTeamRepository{}
	svc := usecase.NewCreateTeamService(events, participants, teams)

	_, err := svc.Create(context.Background(), port.CreateTeamCommand{
		UserID:    uuid.New(),
		EventSlug: "test",
		TeamName:  "Test Team",
	})
	assert.ErrorIs(t, err, response.ErrInvariantViolated)
}

func TestCreateTeamService_CannotCreateWhenNotParticipant(t *testing.T) {
	events := &mockEventReader{summary: &port.EventSummary{Status: "registration_open"}}
	participants := &mockParticipants{role: "unregistered"}
	teams := &mockTeamRepository{}
	svc := usecase.NewCreateTeamService(events, participants, teams)

	_, err := svc.Create(context.Background(), port.CreateTeamCommand{
		UserID:    uuid.New(),
		EventSlug: "test",
		TeamName:  "Test Team",
	})
	assert.ErrorIs(t, err, response.ErrForbidden)
}

func TestCreateTeamService_Success(t *testing.T) {
	events := &mockEventReader{summary: &port.EventSummary{ID: uuid.New(), Status: "registration_open"}}
	participants := &mockParticipants{role: "participant"}
	teams := &mockTeamRepository{}
	svc := usecase.NewCreateTeamService(events, participants, teams)

	dto, err := svc.Create(context.Background(), port.CreateTeamCommand{
		UserID:    uuid.New(),
		EventSlug: "test",
		TeamName:  "Test Team",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Test Team", dto.Name)
	assert.Len(t, dto.InviteCode, 8)
	assert.Len(t, dto.Members, 1)
	assert.Equal(t, string(domain.RoleOwner), dto.Members[0].Role)
}
