package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SubmissionHandler struct {
	create port.CreateSubmissionUseCase
}

func NewSubmissionHandler(create port.CreateSubmissionUseCase) *SubmissionHandler {
	return &SubmissionHandler{
		create: create,
	}
}

func (h *SubmissionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/events/{slug}/submissions", h.CreateDraft)
}

type createDraftRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	RepoURL     *string    `json:"repoUrl"`
	DemoURL     *string    `json:"demoUrl"`
	VideoURL    *string    `json:"videoUrl"`
	TrackID     *uuid.UUID `json:"trackId"`
}

func (h *SubmissionHandler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return
	}

	var body createDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if body.Title == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "title is required")
		return
	}

	cmd := port.CreateSubmissionCommand{
		CallerID:    userID,
		EventSlug:   slug,
		Title:       body.Title,
		Description: body.Description,
		TrackID:     body.TrackID,
		RepoURL:     body.RepoURL,
		DemoURL:     body.DemoURL,
		VideoURL:    body.VideoURL,
	}

	result, err := h.create.Create(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.Created(w, r, result)
}
