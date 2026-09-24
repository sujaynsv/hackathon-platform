package response

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Meta is the metadata envelope attached to every API response.
type Meta struct {
	RequestID  string `json:"requestId"`
	Timestamp  string `json:"timestamp"`
	Page       *int   `json:"page,omitempty"`
	TotalPages *int   `json:"totalPages,omitempty"`
	TotalCount *int   `json:"totalCount,omitempty"`
}

// ApiResponse is the standard success envelope for single-object responses.
type ApiResponse[T any] struct {
	Data *T   `json:"data,omitempty"`
	Meta Meta `json:"meta"`
}

// ErrorBody holds the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

func newMeta(r *http.Request) Meta {
	return Meta{
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// OK writes a 200 response with the given data wrapped in the standard envelope.
func OK[T any](w http.ResponseWriter, r *http.Request, data T) {
	writeJSON(w, http.StatusOK, ApiResponse[T]{Data: &data, Meta: newMeta(r)})
}

// Created writes a 201 response with the given data wrapped in the standard envelope.
func Created[T any](w http.ResponseWriter, r *http.Request, data T) {
	writeJSON(w, http.StatusCreated, ApiResponse[T]{Data: &data, Meta: newMeta(r)})
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// OKList writes a 200 response for paginated list results.
func OKList[T any](w http.ResponseWriter, r *http.Request, data T, page, totalPages, totalCount int) {
	writeJSON(w, http.StatusOK, ApiResponse[T]{
		Data: &data,
		Meta: Meta{
			RequestID:  uuid.New().String(),
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Page:       &page,
			TotalPages: &totalPages,
			TotalCount: &totalCount,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
