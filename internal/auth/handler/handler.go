package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	register port.RegisterUseCase
}

func NewAuthHandler(register port.RegisterUseCase) *AuthHandler {
	return &AuthHandler{register: register}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	return r
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (req registerRequest) validate() error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if req.DisplayName == "" {
		return fmt.Errorf("displayName is required")
	}
	return nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if err := req.validate(); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.register.Register(r.Context(), port.RegisterCommand{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidEmail) || errors.Is(err, domain.ErrWeakPassword) || err.Error() == "display name is required" {
			response.BadRequest(w, r, "VALIDATION_ERROR", err.Error())
			return
		}
		response.HandleDomainError(w, r, err)
		return
	}
	response.Created(w, r, result)
}
