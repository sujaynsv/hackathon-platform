package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TeamsHandler struct {
	register   port.RegisterForEventUseCase
	unregister port.UnregisterFromEventUseCase
	createTeam port.CreateTeamUseCase
	getMyTeam  port.GetMyTeamUseCase
}

func NewTeamsHandler(
	register port.RegisterForEventUseCase,
	unregister port.UnregisterFromEventUseCase,
	createTeam port.CreateTeamUseCase,
	getMyTeam port.GetMyTeamUseCase,
) *TeamsHandler {
	return &TeamsHandler{
		register:   register,
		unregister: unregister,
		createTeam: createTeam,
		getMyTeam:  getMyTeam,
	}
}

func (h *TeamsHandler) RegisterRoutes(r chi.Router) {
	r.Post("/events/{slug}/register", h.RegisterForEvent)
	r.Delete("/events/{slug}/register", h.UnregisterFromEvent)
	r.Post("/events/{slug}/teams", h.CreateTeam)
	r.Get("/events/{slug}/teams/mine", h.GetMyTeam)
}

func (h *TeamsHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "team name is required")
		return
	}

	result, err := h.createTeam.Create(r.Context(), port.CreateTeamCommand{
		UserID:    userID,
		EventSlug: chi.URLParam(r, "slug"),
		TeamName:  body.Name,
	})
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.Created(w, r, result)
}

func (h *TeamsHandler) GetMyTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	result, err := h.getMyTeam.GetMyTeam(r.Context(), userID, chi.URLParam(r, "slug"))
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *TeamsHandler) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	result, err := h.register.Register(r.Context(), port.RegisterCommand{
		UserID:    userID,
		EventSlug: chi.URLParam(r, "slug"),
	})
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.Created(w, r, result)
}

func (h *TeamsHandler) UnregisterFromEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	if err := h.unregister.Unregister(r.Context(), port.UnregisterCommand{
		UserID:    userID,
		EventSlug: chi.URLParam(r, "slug"),
	}); err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, struct {
		Unregistered bool `json:"unregistered"`
	}{Unregistered: true})
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return uuid.Nil, false
	}
	return userID, true
}
