package response

import (
	"errors"
	"log"
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
	ErrInvalidFileType   = errors.New("invalid file type")

	// Auth errors
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrTokenRevoked        = errors.New("token revoked")
)

type apiError struct {
	status  int
	code    string
	message string
}

// HandleDomainError maps a domain/service error to the correct HTTP status + error envelope.
// This is the single place where error-to-HTTP mapping lives — never duplicate this logic in handlers.
func HandleDomainError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Domain error: %v", err)
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
	case errors.Is(err, ErrInvalidFileType):
		return apiError{400, "INVALID_FILE_TYPE", err.Error()}
	case errors.Is(err, ErrDuplicate):
		return apiError{409, "DUPLICATE_RESOURCE", "resource already exists"}
	case errors.Is(err, ErrInvalidCredentials):
		return apiError{401, "INVALID_CREDENTIALS", "invalid email or password"}
	case errors.Is(err, ErrInvalidRefreshToken):
		return apiError{401, "INVALID_REFRESH_TOKEN", "invalid refresh token"}
	case errors.Is(err, ErrTokenRevoked):
		return apiError{401, "TOKEN_REVOKED", "token revoked"}
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

// Unauthorized writes a 401 response with the given machine-readable code and message.
func Unauthorized(w http.ResponseWriter, r *http.Request, code, message string) {
	writeJSON(w, http.StatusUnauthorized, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
		Meta:  newMeta(r),
	})
}

// PayloadTooLarge writes a 413 response.
func PayloadTooLarge(w http.ResponseWriter, r *http.Request, code, message string) {
	writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
		Meta:  newMeta(r),
	})
}
