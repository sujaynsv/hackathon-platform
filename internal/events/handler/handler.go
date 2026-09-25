package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/dogfood-platform/dogfood/internal/events/port"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EventHandler struct {
	create port.CreateEventUseCase
	list   port.ListEventsUseCase
	get    port.GetEventUseCase
}

func NewEventHandler(create port.CreateEventUseCase, list port.ListEventsUseCase, get port.GetEventUseCase) *EventHandler {
	return &EventHandler{
		create: create,
		list:   list,
		get:    get,
	}
}

type createEventRequest struct {
	Slug                 string     `json:"slug"`
	Title                string     `json:"title"`
	Description          *string    `json:"description"`
	MaxTeamSize          int        `json:"maxTeamSize"`
	RegistrationOpensAt  *time.Time `json:"registrationOpensAt"`
	RegistrationClosesAt *time.Time `json:"registrationClosesAt"`
	SubmissionDeadlineAt *time.Time `json:"submissionDeadlineAt"`
	JudgingDeadlineAt    *time.Time `json:"judgingDeadlineAt"`
	VotingOpensAt        *time.Time `json:"votingOpensAt"`
	VotingClosesAt       *time.Time `json:"votingClosesAt"`
}

func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	callerIDStr := middleware.GetUserID(r.Context())
	var callerID uuid.UUID
	if callerIDStr != "" {
		callerID, _ = uuid.Parse(callerIDStr)
	}

	if callerID == uuid.Nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "authentication required")
		return
	}

	var body createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if body.Slug == "" || body.Title == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "slug and title are required")
		return
	}

	cmd := port.CreateEventCommand{
		OrganizerID:          callerID,
		Slug:                 body.Slug,
		Title:                body.Title,
		Description:          body.Description,
		MaxTeamSize:          body.MaxTeamSize,
		RegistrationOpensAt:  body.RegistrationOpensAt,
		RegistrationClosesAt: body.RegistrationClosesAt,
		SubmissionDeadlineAt: body.SubmissionDeadlineAt,
		JudgingDeadlineAt:    body.JudgingDeadlineAt,
		VotingOpensAt:        body.VotingOpensAt,
		VotingClosesAt:       body.VotingClosesAt,
	}

	dto, err := h.create.Create(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.Created(w, r, dto)
}

func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	callerIDStr := middleware.GetUserID(r.Context())
	var callerID uuid.UUID
	if callerIDStr != "" {
		callerID, _ = uuid.Parse(callerIDStr)
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	var ptrCallerID *uuid.UUID
	if callerID != uuid.Nil {
		ptrCallerID = &callerID
	}

	q := port.ListEventsQuery{
		CallerID: ptrCallerID,
		Page:     page,
		PageSize: pageSize,
	}

	dto, err := h.list.List(r.Context(), q)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OKList(w, r, dto.Events, dto.Page, dto.TotalPages, dto.TotalCount)
}

func (h *EventHandler) GetEventBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	callerIDStr := middleware.GetUserID(r.Context())
	var callerID uuid.UUID
	if callerIDStr != "" {
		callerID, _ = uuid.Parse(callerIDStr)
	}

	var ptrCallerID *uuid.UUID
	if callerID != uuid.Nil {
		ptrCallerID = &callerID
	}

	dto, err := h.get.GetBySlug(r.Context(), slug, ptrCallerID)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, dto)
}

func (h *EventHandler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/api/v1/events", h.ListEvents)
	r.Get("/api/v1/events/{slug}", h.GetEventBySlug)
}

func (h *EventHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/api/v1/events", h.CreateEvent)
}
