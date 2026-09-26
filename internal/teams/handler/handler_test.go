package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/handler"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/dogfood-platform/dogfood/internal/teams/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type flowEventReader struct {
	event *port.EventSummary
}

func (r flowEventReader) FindSummaryBySlug(_ context.Context, slug string) (*port.EventSummary, error) {
	if slug != r.event.Slug {
		return nil, response.ErrNotFound
	}
	return r.event, nil
}

type flowParticipants struct {
	roles map[uuid.UUID]map[string]bool
}

func (r *flowParticipants) HasRole(_ context.Context, userID, _ uuid.UUID, role string) (bool, error) {
	return r.roles[userID][role], nil
}

func (r *flowParticipants) GrantParticipantRole(_ context.Context, userID, _ uuid.UUID) (time.Time, error) {
	if r.roles[userID] == nil {
		r.roles[userID] = make(map[string]bool)
	}
	r.roles[userID]["participant"] = true
	return time.Date(2026, time.September, 26, 10, 0, 0, 0, time.UTC), nil
}

func (r *flowParticipants) RevokeParticipantRole(_ context.Context, userID, _ uuid.UUID) error {
	r.roles[userID]["participant"] = false
	return nil
}

type flowTeamMembers struct {
	users map[uuid.UUID]bool
}

func (r flowTeamMembers) HasTeamInEvent(_ context.Context, userID, _ uuid.UUID) (bool, error) {
	return r.users[userID], nil
}

func TestRegistrationHTTPFlow(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	event := &port.EventSummary{ID: uuid.New(), Slug: "hackathon-2026", Status: "registration_open"}
	participants := &flowParticipants{roles: make(map[uuid.UUID]map[string]bool)}
	teamMembers := flowTeamMembers{users: make(map[uuid.UUID]bool)}

	register := usecase.NewRegisterService(flowEventReader{event: event}, participants)
	unregister := usecase.NewUnregisterService(flowEventReader{event: event}, participants, teamMembers)
	teamsHandler := handler.NewTeamsHandler(register, unregister, nil, nil)

	currentUserID := userID
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "user_id", currentUserID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	teamsHandler.RegisterRoutes(router)

	request := func(method string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/events/"+event.Slug+"/register", nil)
		router.ServeHTTP(recorder, req)
		return recorder
	}

	firstRegistration := request(http.MethodPost)
	if firstRegistration.Code != http.StatusCreated {
		t.Fatalf("first registration status = %d, want %d: %s", firstRegistration.Code, http.StatusCreated, firstRegistration.Body.String())
	}
	var created struct {
		Data port.RegistrationDTO `json:"data"`
	}
	if err := json.Unmarshal(firstRegistration.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	if !created.Data.Registered || created.Data.EventID != event.ID.String() || created.Data.EventSlug != event.Slug {
		t.Fatalf("unexpected registration response: %+v", created.Data)
	}

	duplicate := request(http.MethodPost)
	assertErrorCode(t, duplicate, http.StatusConflict, "DUPLICATE_RESOURCE")

	currentUserID = otherUserID
	event.Status = "draft"
	closedEvent := request(http.MethodPost)
	assertErrorCode(t, closedEvent, http.StatusUnprocessableEntity, "INVARIANT_VIOLATION")

	event.Status = "registration_open"
	currentUserID = userID
	unregistered := request(http.MethodDelete)
	if unregistered.Code != http.StatusOK {
		t.Fatalf("unregister status = %d, want %d: %s", unregistered.Code, http.StatusOK, unregistered.Body.String())
	}
	var removed struct {
		Data struct {
			Unregistered bool `json:"unregistered"`
		} `json:"data"`
	}
	if err := json.Unmarshal(unregistered.Body.Bytes(), &removed); err != nil {
		t.Fatalf("decode unregister response: %v", err)
	}
	if !removed.Data.Unregistered {
		t.Fatalf("expected unregistered=true, got %+v", removed.Data)
	}

	currentUserID = otherUserID
	if res := request(http.MethodPost); res.Code != http.StatusCreated {
		t.Fatalf("second user registration status = %d, want %d: %s", res.Code, http.StatusCreated, res.Body.String())
	}
	teamMembers.users[otherUserID] = true
	lockedUnregister := request(http.MethodDelete)
	assertErrorCode(t, lockedUnregister, http.StatusUnprocessableEntity, "INVARIANT_VIOLATION")
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d: %s", response.Code, wantStatus, response.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", body.Error.Code, wantCode)
	}
}
