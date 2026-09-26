package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SubmissionHandler struct {
	create     port.CreateSubmissionUseCase
	update     port.UpdateSubmissionUseCase
	submit     port.FinalSubmitUseCase
	upload     port.UploadUseCase
	listFiles  port.ListFilesUseCase
	listGallery port.ListSubmissionsUseCase
	disqualify port.DisqualifyUseCase
}

func NewSubmissionHandler(
	create port.CreateSubmissionUseCase,
	update port.UpdateSubmissionUseCase,
	submit port.FinalSubmitUseCase,
	upload port.UploadUseCase,
	listFiles port.ListFilesUseCase,
	listGallery port.ListSubmissionsUseCase,
	disqualify port.DisqualifyUseCase,
) *SubmissionHandler {
	return &SubmissionHandler{
		create:      create,
		update:      update,
		submit:      submit,
		upload:      upload,
		listFiles:   listFiles,
		listGallery: listGallery,
		disqualify:  disqualify,
	}
}

func (h *SubmissionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/events/{slug}/submissions", h.CreateDraft)
	r.Patch("/submissions/{id}", h.UpdateDraft)
	r.Post("/submissions/{id}/submit", h.SubmitDraft)
	r.Post("/submissions/{id}/upload", h.UploadFile)
	r.Get("/submissions/{id}/files", h.ListFiles)
	r.Get("/events/{slug}/submissions", h.ListGallery)
	r.Post("/submissions/{id}/disqualify", h.Disqualify)
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

type updateDraftRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	RepoURL     *string `json:"repoUrl"`
	DemoURL     *string `json:"demoUrl"`
}

func (h *SubmissionHandler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid submission ID")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return
	}

	var body updateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}

	cmd := port.UpdateSubmissionCommand{
		CallerID:     userID,
		SubmissionID: subID,
		Title:        body.Title,
		Description:  body.Description,
		RepoURL:      body.RepoURL,
		DemoURL:      body.DemoURL,
	}

	result, err := h.update.Update(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, result)
}

func (h *SubmissionHandler) SubmitDraft(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	subID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid submission ID")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return
	}

	cmd := port.SubmitCommand{
		CallerID:     userID,
		SubmissionID: subID,
	}

	result, err := h.submit.Submit(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, result)
}

func (h *SubmissionHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	subID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid submission ID")
		return
	}
	
	userIDStr := middleware.GetUserID(r.Context())
	callerID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return
	}

	// Max 50MB + 10MB overhead for form fields
	r.Body = http.MaxBytesReader(w, r.Body, (50+10)*1024*1024)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		if strings.Contains(err.Error(), "too large") {
			response.PayloadTooLarge(w, r, "FILE_TOO_LARGE", "file exceeds 50MB limit")
			return
		}
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "missing file in form data")
		return
	}
	defer file.Close()

	if header.Size > 50*1024*1024 {
		response.PayloadTooLarge(w, r, "FILE_TOO_LARGE", "file exceeds 50MB limit")
		return
	}

	uploadType := r.FormValue("type")
	if uploadType != "cover" && uploadType != "attachment" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "type must be cover or attachment")
		return
	}

	cmd := port.UploadCommand{
		CallerID:     callerID,
		SubmissionID: subID,
		FileName:     header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		SizeBytes:    header.Size,
		UploadType:   uploadType,
		File:         file,
	}

	res, err := h.upload.Upload(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.Created(w, r, res)
}

func (h *SubmissionHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	subID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid submission ID")
		return
	}

	res, err := h.listFiles.ListFiles(r.Context(), subID)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, res)
}

func (h *SubmissionHandler) ListGallery(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 50
	}

	var trackID *uuid.UUID
	trackIDStr := r.URL.Query().Get("trackId")
	if trackIDStr != "" {
		if id, err := uuid.Parse(trackIDStr); err == nil {
			trackID = &id
		}
	}

	query := port.ListSubmissionsQuery{
		EventSlug: slug,
		TrackID:   trackID,
		Page:      page,
		PageSize:  pageSize,
	}

	dtos, total, err := h.listGallery.List(r.Context(), query)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	response.OKList(w, r, dtos, page, totalPages, total)
}

type disqualifyRequest struct {
	Reason string `json:"reason"`
}

func (h *SubmissionHandler) Disqualify(w http.ResponseWriter, r *http.Request) {
	subID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid submission ID")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	adminID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
		return
	}

	var req disqualifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Reason == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "reason is required")
		return
	}

	cmd := port.DisqualifyCommand{
		AdminID:      adminID,
		SubmissionID: subID,
		Reason:       req.Reason,
	}

	err = h.disqualify.Disqualify(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, map[string]string{"status": "disqualified"})
}
