package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/events/handler"
	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCreate struct {
	mock.Mock
}

func (m *MockCreate) Create(ctx context.Context, cmd port.CreateEventCommand) (*port.EventDTO, error) {
	args := m.Called(ctx, cmd)
	var e *port.EventDTO
	if args.Get(0) != nil {
		e = args.Get(0).(*port.EventDTO)
	}
	return e, args.Error(1)
}

type MockList struct {
	mock.Mock
}

func (m *MockList) List(ctx context.Context, q port.ListEventsQuery) (*port.EventListDTO, error) {
	args := m.Called(ctx, q)
	var e *port.EventListDTO
	if args.Get(0) != nil {
		e = args.Get(0).(*port.EventListDTO)
	}
	return e, args.Error(1)
}

type MockGet struct {
	mock.Mock
}

func (m *MockGet) GetBySlug(ctx context.Context, slug string, callerID *uuid.UUID) (*port.EventDetailDTO, error) {
	args := m.Called(ctx, slug, callerID)
	var e *port.EventDetailDTO
	if args.Get(0) != nil {
		e = args.Get(0).(*port.EventDetailDTO)
	}
	return e, args.Error(1)
}

func TestCreateEventHandler_ValidBody_Returns201(t *testing.T) {
	m := new(MockCreate)
	h := handler.NewEventHandler(m, nil, nil, nil)

	body := map[string]interface{}{
		"slug":  "test-event",
		"title": "Test Event",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(jsonBody))
	// mock auth context
	ctx := context.WithValue(req.Context(), "user_id", uuid.New().String())
	req = req.WithContext(ctx)

	m.On("Create", mock.Anything, mock.AnythingOfType("port.CreateEventCommand")).Return(&port.EventDTO{
		Slug:   "test-event",
		Status: "draft",
	}, nil)

	rec := httptest.NewRecorder()
	h.CreateEvent(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	m.AssertExpectations(t)
}

func TestCreateEventHandler_MissingTitle_Returns400(t *testing.T) {
	h := handler.NewEventHandler(nil, nil, nil, nil)

	body := map[string]interface{}{
		"slug": "test-event",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(jsonBody))
	ctx := context.WithValue(req.Context(), "user_id", uuid.New().String())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateEvent(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateEventHandler_UnauthenticatedUser_Returns401(t *testing.T) {
	h := handler.NewEventHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	h.CreateEvent(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetEventHandler_UnknownSlug_Returns404(t *testing.T) {
	m := new(MockGet)
	h := handler.NewEventHandler(nil, nil, m, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/unknown", nil)
	// chi route param stub
	// For testing chi context, we can just test the inner mock behavior, since handler uses chi.URLParam
	// Alternatively, just inject it into context:
	// But it's easier to mock the get method returning ErrNotFound

	m.On("GetBySlug", mock.Anything, "", mock.Anything).Return(nil, response.ErrNotFound)

	rec := httptest.NewRecorder()
	h.GetEventBySlug(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

type MockUpdate struct {
	mock.Mock
}

func (m *MockUpdate) Update(ctx context.Context, cmd port.UpdateEventCommand) (*port.EventDetailDTO, error) {
	args := m.Called(ctx, cmd)
	var e *port.EventDetailDTO
	if args.Get(0) != nil {
		e = args.Get(0).(*port.EventDetailDTO)
	}
	return e, args.Error(1)
}

func TestUpdateEventHandler_ValidBody_Returns200(t *testing.T) {
	m := new(MockUpdate)
	h := handler.NewEventHandler(nil, nil, nil, m)

	body := map[string]interface{}{
		"title": "New Title",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/events/test-event", bytes.NewReader(jsonBody))
	ctx := context.WithValue(req.Context(), "user_id", uuid.New().String())
	req = req.WithContext(ctx)

	m.On("Update", mock.Anything, mock.AnythingOfType("port.UpdateEventCommand")).Return(&port.EventDetailDTO{}, nil)

	rec := httptest.NewRecorder()
	h.UpdateEvent(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateEventHandler_Unauthenticated_Returns401(t *testing.T) {
	h := handler.NewEventHandler(nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPatch, "/events/test-event", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	h.UpdateEvent(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateEventHandler_NonOrganizer_Returns403(t *testing.T) {
	m := new(MockUpdate)
	h := handler.NewEventHandler(nil, nil, nil, m)

	req := httptest.NewRequest(http.MethodPatch, "/events/test-event", bytes.NewReader([]byte("{}")))
	ctx := context.WithValue(req.Context(), "user_id", uuid.New().String())
	req = req.WithContext(ctx)

	m.On("Update", mock.Anything, mock.Anything).Return(nil, response.ErrForbidden)

	rec := httptest.NewRecorder()
	h.UpdateEvent(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpdateEventHandler_InvalidTransition_Returns422(t *testing.T) {
	m := new(MockUpdate)
	h := handler.NewEventHandler(nil, nil, nil, m)

	req := httptest.NewRequest(http.MethodPatch, "/events/test-event", bytes.NewReader([]byte("{}")))
	ctx := context.WithValue(req.Context(), "user_id", uuid.New().String())
	req = req.WithContext(ctx)

	m.On("Update", mock.Anything, mock.Anything).Return(nil, response.ErrInvalidTransition)

	rec := httptest.NewRecorder()
	h.UpdateEvent(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
