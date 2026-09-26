package handler

import (
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/shared/middleware"
    "github.com/dogfood-platform/dogfood/internal/shared/response"
)

type JudgingHandler struct {
    getQueue  port.GetQueueUseCase
    getDetail port.GetAssignmentDetailUseCase
}

func NewJudgingHandler(q port.GetQueueUseCase, d port.GetAssignmentDetailUseCase) *JudgingHandler {
    return &JudgingHandler{
        getQueue:  q,
        getDetail: d,
    }
}

func (h *JudgingHandler) RegisterRoutes(r chi.Router) {
    r.Route("/judging", func(r chi.Router) {
        // I16 - only judges can see queue? Well, it returns my assignments
        r.Get("/queue", h.GetQueue)
        r.Get("/assignments/{id}", h.GetAssignmentDetail)
    })
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
    userID, err := uuid.Parse(middleware.GetUserID(r.Context()))
    if err != nil {
        response.Unauthorized(w, r, "UNAUTHORIZED", "invalid user token")
        return uuid.Nil, false
    }
    return userID, true
}

func (h *JudgingHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
    judgeID, ok := authenticatedUserID(w, r)
    if !ok {
        return
    }

    q := port.QueueQuery{
        Status: r.URL.Query().Get("status"),
    }
    if q.Status == "" {
        q.Status = "pending"
    }
    if eventIDStr := r.URL.Query().Get("eventId"); eventIDStr != "" {
        if eid, err := uuid.Parse(eventIDStr); err == nil {
            q.EventID = &eid
        }
    }
    if pageStr := r.URL.Query().Get("page"); pageStr != "" {
        if page, err := strconv.Atoi(pageStr); err == nil {
            q.Page = page
        }
    }
    if sizeStr := r.URL.Query().Get("pageSize"); sizeStr != "" {
        if size, err := strconv.Atoi(sizeStr); err == nil {
            q.PageSize = size
        }
    }

    dto, err := h.getQueue.GetQueue(r.Context(), judgeID, q)
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }

    response.OKList(w, r, dto.Data, dto.Meta.Page, dto.Meta.TotalPages, dto.Meta.TotalCount)
}

func (h *JudgingHandler) GetAssignmentDetail(w http.ResponseWriter, r *http.Request) {
    judgeID, ok := authenticatedUserID(w, r)
    if !ok {
        return
    }

    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        response.BadRequest(w, r, "INVALID_ID", "invalid assignment ID format")
        return
    }

    dto, err := h.getDetail.GetDetail(r.Context(), judgeID, id)
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }

    response.OK(w, r, dto)
}
