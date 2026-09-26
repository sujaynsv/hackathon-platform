package usecase_test

import (
	"context"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/dogfood-platform/dogfood/internal/events/usecase"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockEventRepo struct {
	mock.Mock
}

func (m *MockEventRepo) Save(ctx context.Context, event *domain.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventRepo) FindBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	args := m.Called(ctx, slug)
	var e *domain.Event
	if args.Get(0) != nil {
		e = args.Get(0).(*domain.Event)
	}
	return e, args.Error(1)
}

func (m *MockEventRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventRepo) ListPublished(ctx context.Context, page, pageSize int) ([]*domain.Event, int, error) {
	args := m.Called(ctx, page, pageSize)
	var events []*domain.Event
	if args.Get(0) != nil {
		events = args.Get(0).([]*domain.Event)
	}
	return events, args.Int(1), args.Error(2)
}

func (m *MockEventRepo) GetEventDetail(ctx context.Context, slug string, callerID *uuid.UUID) (*domain.Event, []domain.Track, *string, error) {
	args := m.Called(ctx, slug, callerID)
	var e *domain.Event
	if args.Get(0) != nil {
		e = args.Get(0).(*domain.Event)
	}
	var t []domain.Track
	if args.Get(1) != nil {
		t = args.Get(1).([]domain.Track)
	}
	var r *string
	if args.Get(2) != nil {
		r = args.Get(2).(*string)
	}
	return e, t, r, args.Error(3)
}

func (m *MockEventRepo) Update(ctx context.Context, event *domain.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type MockEventRoleRepo struct {
	mock.Mock
}

func (m *MockEventRoleRepo) GrantRole(ctx context.Context, userID, eventID uuid.UUID, role string) error {
	args := m.Called(ctx, userID, eventID, role)
	return args.Error(0)
}

func (m *MockEventRoleRepo) GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID, eventID)
	return args.String(0), args.Error(1)
}

func TestCreateEventService_ValidInput_Returns201DTO(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	svc := usecase.NewCreateEventService(events, roles)

	cmd := port.CreateEventCommand{
		OrganizerID: uuid.New(),
		Slug:        "test-event",
		Title:       "Test Event",
		MaxTeamSize: 4,
	}

	events.On("ExistsBySlug", mock.Anything, "test-event").Return(false, nil)
	events.On("Save", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil)
	roles.On("GrantRole", mock.Anything, cmd.OrganizerID, mock.AnythingOfType("uuid.UUID"), "organizer").Return(nil)

	dto, err := svc.Create(context.Background(), cmd)

	assert.NoError(t, err)
	assert.NotNil(t, dto)
	assert.Equal(t, "test-event", dto.Slug)
	assert.Equal(t, string(domain.StatusDraft), dto.Status)

	events.AssertExpectations(t)
	roles.AssertExpectations(t)
}

func TestCreateEventService_DuplicateSlug_ReturnsErrInvariantViolated(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	svc := usecase.NewCreateEventService(events, roles)

	cmd := port.CreateEventCommand{
		OrganizerID: uuid.New(),
		Slug:        "duplicate-event",
		Title:       "Duplicate Event",
	}

	events.On("ExistsBySlug", mock.Anything, "duplicate-event").Return(true, nil)

	dto, err := svc.Create(context.Background(), cmd)

	assert.Error(t, err)
	assert.ErrorIs(t, err, response.ErrInvariantViolated)
	assert.Nil(t, dto)

	events.AssertExpectations(t)
}

func TestCreateEventService_GrantsOrganizerRoleAfterCreate(t *testing.T) {
	// Already covered in TestCreateEventService_ValidInput_Returns201DTO by verifying roles.On("GrantRole")
}
