package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	register port.RegisterUseCase
	verify   port.VerifyEmailUseCase
	login    port.LoginUseCase
	refresh  port.RefreshUseCase
	logout   port.LogoutUseCase
	cache    port.Cache
	jwtSecret string
}

func NewAuthHandler(
	register port.RegisterUseCase,
	verify port.VerifyEmailUseCase,
	login port.LoginUseCase,
	refresh port.RefreshUseCase,
	logout port.LogoutUseCase,
	cache port.Cache,
	jwtSecret string,
) *AuthHandler {
	return &AuthHandler{
		register:  register,
		verify:    verify,
		login:     login,
		refresh:   refresh,
		logout:    logout,
		cache:     cache,
		jwtSecret: jwtSecret,
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/verify-email", h.VerifyEmail)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware(h.jwtSecret, h.cache))
		r.Post("/logout", h.Logout)
	})
	return r
}

type registerRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	DisplayName  string `json:"displayName"`
	CaptchaToken string `json:"captchaToken"`
	CFTurnstile  string `json:"cf-turnstile-response"`
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

	token := req.CaptchaToken
	if token == "" {
		token = req.CFTurnstile
	}

	result, err := h.register.Register(r.Context(), port.RegisterCommand{
		Email:        req.Email,
		Password:     req.Password,
		DisplayName:  req.DisplayName,
		CaptchaToken: token,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidEmail) || errors.Is(err, domain.ErrWeakPassword) || errors.Is(err, domain.ErrInvalidDisplayName) {
			response.BadRequest(w, r, "VALIDATION_ERROR", err.Error())
			return
		}
		response.HandleDomainError(w, r, err)
		return
	}
	response.Created(w, r, result)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req port.VerifyEmailCommand
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Token == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "token is required")
		return
	}

	if err := h.verify.Verify(r.Context(), req); err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, map[string]string{"message": "email successfully verified"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "email and password are required")
		return
	}
	result, err := h.login.Login(r.Context(), port.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, result)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.BadRequest(w, r, "VALIDATION_ERROR", "refreshToken is required")
		return
	}
	pair, err := h.refresh.Refresh(r.Context(), port.RefreshCommand{RawRefreshToken: req.RefreshToken})
	if err != nil {
		response.HandleDomainError(w, r, err)
		return
	}
	response.OK(w, r, pair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	jti := middleware.GetJTI(r.Context())
	exp := middleware.GetTokenExpiry(r.Context())
	_ = h.logout.Logout(r.Context(), jti, exp)
	response.OK(w, r, map[string]bool{"loggedOut": true})
}
