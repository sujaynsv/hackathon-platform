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

type MockTrackRepo struct {
	mock.Mock
}

func (m *MockTrackRepo) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.Track, error) {
	args := m.Called(ctx, eventID)
	var t []*domain.Track
	if args.Get(0) != nil {
		t = args.Get(0).([]*domain.Track)
	}
	return t, args.Error(1)
}
func (m *MockTrackRepo) Save(ctx context.Context, track *domain.Track) error { return nil }

func TestUpdateEventService_ValidTitleChange_ReturnsUpdatedDTO(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	tracks := new(MockTrackRepo)
	svc := usecase.NewUpdateEventService(events, roles, tracks)

	callerID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		Slug:        "test",
		Title:       "Old Title",
		Status:      domain.StatusDraft,
		OrganizerID: callerID,
	}

	events.On("FindBySlug", mock.Anything, "test").Return(event, nil)
	roles.On("GetRole", mock.Anything, callerID, eventID).Return("organizer", nil)
	events.On("Update", mock.Anything, mock.AnythingOfType("*domain.Event")).Return(nil)
	tracks.On("FindByEventID", mock.Anything, eventID).Return([]*domain.Track{}, nil)

	newTitle := "New Title"
	res, err := svc.Update(context.Background(), port.UpdateEventCommand{
		CallerID: callerID,
		Slug:     "test",
		Title:    &newTitle,
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Title", res.Title)
	events.AssertExpectations(t)
}

func TestUpdateEventService_NonOrganizerCaller_ReturnsErrForbidden(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	svc := usecase.NewUpdateEventService(events, roles, nil)

	callerID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		Slug:        "test",
		Status:      domain.StatusDraft,
	}

	events.On("FindBySlug", mock.Anything, "test").Return(event, nil)
	roles.On("GetRole", mock.Anything, callerID, eventID).Return("participant", nil)

	_, err := svc.Update(context.Background(), port.UpdateEventCommand{
		CallerID: callerID,
		Slug:     "test",
	})

	assert.ErrorIs(t, err, response.ErrForbidden)
}

func TestUpdateEventService_InvalidTransition_ReturnsErrInvalidTransition(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	svc := usecase.NewUpdateEventService(events, roles, nil)

	callerID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		Slug:        "test",
		Status:      domain.StatusDraft,
	}

	events.On("FindBySlug", mock.Anything, "test").Return(event, nil)
	roles.On("GetRole", mock.Anything, callerID, eventID).Return("organizer", nil)

	status := string(domain.StatusJudging)
	_, err := svc.Update(context.Background(), port.UpdateEventCommand{
		CallerID:  callerID,
		Slug:      "test",
		NewStatus: &status,
	})

	assert.ErrorIs(t, err, response.ErrInvalidTransition)
}

func TestUpdateEventService_ChangeMaxTeamSizeAfterRegistrationClose_ReturnsErrInvariantViolated(t *testing.T) {
	events := new(MockEventRepo)
	roles := new(MockEventRoleRepo)
	svc := usecase.NewUpdateEventService(events, roles, nil)

	callerID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		Slug:        "test",
		Status:      domain.StatusSubmissionsOpen, // past registration
	}

	events.On("FindBySlug", mock.Anything, "test").Return(event, nil)
	roles.On("GetRole", mock.Anything, callerID, eventID).Return("organizer", nil)

	size := 10
	_, err := svc.Update(context.Background(), port.UpdateEventCommand{
		CallerID:    callerID,
		Slug:        "test",
		MaxTeamSize: &size,
	})

	assert.ErrorIs(t, err, response.ErrInvariantViolated)
}
