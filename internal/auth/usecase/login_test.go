package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/auth/usecase"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginService_ValidCredentials_ReturnsAuthResponse(t *testing.T) {
	repo := &mockUserRepo{
		exists: true,
		saved: &domain.User{
			Email:        "bob@example.com",
			PasswordHash: "hashed_bobpass",
			IsActive:     true,
			DisplayName:  "Bob",
		},
	}
	svc := usecase.NewLoginService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{})

	resp, err := svc.Login(context.Background(), port.LoginCommand{
		Email:    "bob@example.com",
		Password: "bobpass",
	})
	require.NoError(t, err)
	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
	assert.Equal(t, "bob@example.com", resp.User.Email)
}

func TestLoginService_WrongPassword_ReturnsErrUnauthorized(t *testing.T) {
	repo := &mockUserRepo{
		exists: true,
		saved: &domain.User{
			Email:        "bob@example.com",
			PasswordHash: "hashed_realpass",
			IsActive:     true,
		},
	}
	// mockHasher returns "hashed_" + plain, which won't match "hashed_realpass"
	svc := usecase.NewLoginService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{})

	_, err := svc.Login(context.Background(), port.LoginCommand{
		Email:    "bob@example.com",
		Password: "wrongpass", // hashed_wrongpass != hashed_realpass
	})
	assert.ErrorIs(t, err, response.ErrUnauthorized)
}

func TestLoginService_UserNotFound_ReturnsErrUnauthorized(t *testing.T) {
	repo := &mockUserRepo{exists: false}
	svc := usecase.NewLoginService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{})

	_, err := svc.Login(context.Background(), port.LoginCommand{
		Email:    "nobody@example.com",
		Password: "pass",
	})
	assert.ErrorIs(t, err, response.ErrUnauthorized)
}

func TestLoginService_InactiveUser_ReturnsErrForbidden(t *testing.T) {
	repo := &mockUserRepo{
		exists: true,
		saved: &domain.User{
			Email:        "bob@example.com",
			PasswordHash: "hashed_pass",
			IsActive:     false,
		},
	}
	svc := usecase.NewLoginService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{})

	_, err := svc.Login(context.Background(), port.LoginCommand{
		Email:    "bob@example.com",
		Password: "pass",
	})
	assert.ErrorIs(t, err, response.ErrForbidden)
}
