package handler

import (
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TeamsHandler struct {
	register   port.RegisterForEventUseCase
	unregister port.UnregisterFromEventUseCase
}

func NewTeamsHandler(register port.RegisterForEventUseCase, unregister port.UnregisterFromEventUseCase) *TeamsHandler {
	return &TeamsHandler{register: register, unregister: unregister}
}

func (h *TeamsHandler) RegisterRoutes(r chi.Router) {
	r.Post("/events/{slug}/register", h.RegisterForEvent)
	r.Delete("/events/{slug}/register", h.UnregisterFromEvent)
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
