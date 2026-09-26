package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/go-chi/chi/v5"
)

type TeamHandler struct {
	create port.CreateTeamUseCase
	getMy  port.GetMyTeamUseCase
}

func NewTeamHandler(create port.CreateTeamUseCase, getMy port.GetMyTeamUseCase) *TeamHandler {
	return &TeamHandler{create: create, getMy: getMy}
}

func (h *TeamHandler) RegisterRoutes(r chi.Router) {
	r.Post("/events/{slug}/teams", h.CreateTeam)
	r.Get("/events/{slug}/teams/mine", h.GetMyTeam)
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "INVALID_REQUEST", "invalid body")
		return
	}
	if req.Name == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "team name is required")
		return
	}

	cmd := port.CreateTeamCommand{
		UserID:    userID,
		EventSlug: chi.URLParam(r, "slug"),
		TeamName:  req.Name,
	}

	result, err := h.create.Create(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.Created(w, r, result)
}

func (h *TeamHandler) GetMyTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	query := port.GetMyTeamQuery{
		UserID:    userID,
		EventSlug: chi.URLParam(r, "slug"),
	}

	result, err := h.getMy.GetMyTeam(r.Context(), query)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, result)
}
