package response

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors used across all use cases and handlers.
// Use errors.Is() to check — never compare error strings directly.
var (
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvariantViolated = errors.New("invariant violation")
	ErrDeadlinePassed    = errors.New("deadline passed")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrRateLimited       = errors.New("rate limited")
	ErrDuplicate         = errors.New("duplicate resource")
)

type apiError struct {
	status  int
	code    string
	message string
}

// HandleDomainError maps a domain/service error to the correct HTTP status + error envelope.
// This is the single place where error-to-HTTP mapping lives — never duplicate this logic in handlers.
func HandleDomainError(w http.ResponseWriter, r *http.Request, err error) {
	ae := mapError(err)
	writeJSON(w, ae.status, ErrorResponse{
		Error: ErrorBody{Code: ae.code, Message: ae.message},
		Meta:  newMeta(r),
	})
}

func mapError(err error) apiError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apiError{409, "DUPLICATE_RESOURCE", "resource already exists"}
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return apiError{404, "NOT_FOUND", "resource not found"}
	case errors.Is(err, ErrUnauthorized):
		return apiError{401, "UNAUTHORIZED", "authentication required"}
	case errors.Is(err, ErrForbidden):
		return apiError{403, "FORBIDDEN", "insufficient permissions"}
	case errors.Is(err, ErrRateLimited):
		return apiError{429, "RATE_LIMITED", "too many requests"}
	case errors.Is(err, ErrInvalidTransition):
		return apiError{422, "INVALID_STATE_TRANSITION", err.Error()}
	case errors.Is(err, ErrDeadlinePassed):
		return apiError{422, "DEADLINE_PASSED", "deadline has passed"}
	case errors.Is(err, ErrInvariantViolated):
		return apiError{422, "INVARIANT_VIOLATION", err.Error()}
	case errors.Is(err, ErrDuplicate):
		return apiError{409, "DUPLICATE_RESOURCE", "resource already exists"}
	default:
		return apiError{500, "INTERNAL_SERVER_ERROR", "an unexpected error occurred"}
	}
}

// BadRequest writes a 400 response with the given machine-readable code and message.
func BadRequest(w http.ResponseWriter, r *http.Request, code, message string) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
		Meta:  newMeta(r),
	})
}
