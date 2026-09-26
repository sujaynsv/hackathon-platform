package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/teams/handler"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type createTeamUseCaseStub struct {
	command port.CreateTeamCommand
	result  *port.TeamDTO
	err     error
}

func (s *createTeamUseCaseStub) Create(_ context.Context, command port.CreateTeamCommand) (*port.TeamDTO, error) {
	s.command = command
	return s.result, s.err
}

type getMyTeamUseCaseStub struct {
	userID    uuid.UUID
	eventSlug string
	result    *port.TeamDTO
	err       error
}

func (s *getMyTeamUseCaseStub) GetMyTeam(_ context.Context, userID uuid.UUID, eventSlug string) (*port.TeamDTO, error) {
	s.userID = userID
	s.eventSlug = eventSlug
	return s.result, s.err
}

func TestCreateTeamRoute_Returns201AndLeaderDTO(t *testing.T) {
	userID := uuid.New()
	createTeam := &createTeamUseCaseStub{result: &port.TeamDTO{
		ID:         uuid.New().String(),
		EventID:    uuid.New().String(),
		Name:       "Team Rocket",
		InviteCode: "HACK2026",
		Members: []port.TeamMemberDTO{{
			UserID:      userID.String(),
			DisplayName: "Alice",
			Role:        "owner",
		}},
	}}
	teamsHandler := handler.NewTeamsHandler(nil, nil, createTeam, nil)
	recorder := serveWithUser(t, teamsHandler, userID, http.MethodPost, "/events/event-2026/teams", `{"name":"Team Rocket"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if createTeam.command.UserID != userID || createTeam.command.EventSlug != "event-2026" || createTeam.command.TeamName != "Team Rocket" {
		t.Fatalf("unexpected create command: %+v", createTeam.command)
	}
	var body struct {
		Data port.TeamDTO `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if body.Data.InviteCode != "HACK2026" || len(body.Data.Members) != 1 || body.Data.Members[0].Role != "owner" {
		t.Fatalf("unexpected create response: %+v", body.Data)
	}
}

func TestCreateTeamRoute_EmptyNameReturns400(t *testing.T) {
	createTeam := &createTeamUseCaseStub{}
	teamsHandler := handler.NewTeamsHandler(nil, nil, createTeam, nil)
	recorder := serveWithUser(t, teamsHandler, uuid.New(), http.MethodPost, "/events/event-2026/teams", `{"name":"  "}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if createTeam.command.EventSlug != "" {
		t.Fatal("create use case was called with a blank team name")
	}
}

func TestGetMyTeamRoute_ReturnsTeamDTO(t *testing.T) {
	userID := uuid.New()
	getMyTeam := &getMyTeamUseCaseStub{result: &port.TeamDTO{
		ID:         uuid.New().String(),
		EventID:    uuid.New().String(),
		Name:       "Team Rocket",
		InviteCode: "HACK2026",
	}}
	teamsHandler := handler.NewTeamsHandler(nil, nil, nil, getMyTeam)
	recorder := serveWithUser(t, teamsHandler, userID, http.MethodGet, "/events/event-2026/teams/mine", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if getMyTeam.userID != userID || getMyTeam.eventSlug != "event-2026" {
		t.Fatalf("unexpected get-my-team arguments: user=%s slug=%q", getMyTeam.userID, getMyTeam.eventSlug)
	}
	var body struct {
		Data port.TeamDTO `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode get-my-team response: %v", err)
	}
	if body.Data.InviteCode != "HACK2026" {
		t.Fatalf("unexpected get-my-team response: %+v", body.Data)
	}
}

func serveWithUser(t *testing.T, teamsHandler *handler.TeamsHandler, userID uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "user_id", userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	teamsHandler.RegisterRoutes(router)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	router.ServeHTTP(recorder, req)
	return recorder
}
