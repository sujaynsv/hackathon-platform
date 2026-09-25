package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/handler"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRegisterUseCase struct {
	shouldErr error
}

func (m *mockRegisterUseCase) Register(ctx context.Context, cmd port.RegisterCommand) (*port.AuthResponse, error) {
	if m.shouldErr != nil {
		return nil, m.shouldErr
	}
	return &port.AuthResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		User: port.UserDTO{
			ID:          "123",
			Email:       cmd.Email,
			DisplayName: cmd.DisplayName,
		},
	}, nil
}

func TestAuthHandler_Register_Success(t *testing.T) {
	uc := &mockRegisterUseCase{}
	h := handler.NewAuthHandler(uc)
	
	r := chi.NewRouter()
	r.Mount("/", h.Routes())

	reqBody := `{"email":"alice@example.com","password":"Password123!","displayName":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	
	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)

	data := res["data"].(map[string]interface{})
	assert.Equal(t, "alice@example.com", data["user"].(map[string]interface{})["email"])
	assert.Equal(t, "access", data["accessToken"])
}

func TestAuthHandler_Register_InvalidEmail(t *testing.T) {
	uc := &mockRegisterUseCase{shouldErr: domain.ErrInvalidEmail}
	h := handler.NewAuthHandler(uc)
	
	r := chi.NewRouter()
	r.Mount("/", h.Routes())

	reqBody := `{"email":"bademail","password":"Password123!","displayName":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	
	errData := res["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errData["code"])
}
