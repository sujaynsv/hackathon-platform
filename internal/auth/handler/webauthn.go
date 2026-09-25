package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/go-chi/chi/v5"
)

type WebAuthnHandler struct {
	webAuthn port.WebAuthnUseCase
}

func NewWebAuthnHandler(webAuthn port.WebAuthnUseCase) *WebAuthnHandler {
	return &WebAuthnHandler{webAuthn: webAuthn}
}

func (h *WebAuthnHandler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/webauthn/login/begin", h.LoginBegin)
	r.Post("/auth/webauthn/login/finish", h.LoginFinish)
}

func (h *WebAuthnHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/auth/webauthn/register/begin", h.RegisterBegin)
	r.Post("/auth/webauthn/register/finish", h.RegisterFinish)
}

func (h *WebAuthnHandler) RegisterBegin(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "UNAUTHORIZED", "must be logged in to register a passkey")
		return
	}

	cmd := port.WebAuthnRegisterBeginCommand{UserID: userID}
	creationData, sessionID, err := h.webAuthn.RegisterBegin(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	w.Header().Set("X-WebAuthn-Session", sessionID)
	// Wrap creationData and sessionID together for the frontend
	response.OK(w, r, map[string]any{
		"options":   creationData,
		"sessionId": sessionID,
	})
}

func (h *WebAuthnHandler) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "UNAUTHORIZED", "must be logged in to register a passkey")
		return
	}

	sessionID := r.Header.Get("X-WebAuthn-Session")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("sessionId")
		if sessionID == "" {
			response.BadRequest(w, r, "VALIDATION_ERROR", "missing sessionId header or query param")
			return
		}
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "unable to read body")
		return
	}

	cmd := port.WebAuthnRegisterFinishCommand{
		UserID:    userID,
		SessionID: sessionID,
		Body:      bodyBytes,
	}

	authResp, err := h.webAuthn.RegisterFinish(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.Created(w, r, authResp)
}

func (h *WebAuthnHandler) LoginBegin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}

	cmd := port.WebAuthnLoginBeginCommand{Email: body.Email}
	assertionData, sessionID, err := h.webAuthn.LoginBegin(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	w.Header().Set("X-WebAuthn-Session", sessionID)
	response.OK(w, r, map[string]any{
		"options":   assertionData,
		"sessionId": sessionID,
	})
}

func (h *WebAuthnHandler) LoginFinish(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-WebAuthn-Session")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("sessionId")
		if sessionID == "" {
			response.BadRequest(w, r, "VALIDATION_ERROR", "missing sessionId header or query param")
			return
		}
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "missing email query param")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "unable to read body")
		return
	}

	cmd := port.WebAuthnLoginFinishCommand{
		Email:     email,
		SessionID: sessionID,
		Body:      bodyBytes,
	}

	authResp, err := h.webAuthn.LoginFinish(r.Context(), cmd)
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}

	response.OK(w, r, authResp)
}
