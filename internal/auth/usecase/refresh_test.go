package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/auth/usecase"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRefreshRepoAdvanced struct {
	mockRefreshRepo
	rt *domain.RefreshToken
}

func (m *mockRefreshRepoAdvanced) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	if m.rt != nil && m.rt.Hash == hash {
		return m.rt, nil
	}
	return nil, response.ErrNotFound
}
func (m *mockRefreshRepoAdvanced) Revoke(ctx context.Context, id uuid.UUID) error {
	if m.rt != nil && m.rt.ID == id {
		if m.rt.RevokedAt != nil {
			return response.ErrTokenRevoked
		}
		now := time.Now()
		m.rt.RevokedAt = &now
	}
	return nil
}

func TestRefreshService_ValidToken_ReturnsNewTokenPair(t *testing.T) {
	userID := uuid.New()
	rawToken := "valid_refresh"
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Hash:      domain.HashToken(rawToken),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	userRepo := &mockUserRepo{
		exists: true,
		saved: &domain.User{
			ID:       userID,
			Email:    "user@example.com",
			IsActive: true,
		},
	}
	refreshRepo := &mockRefreshRepoAdvanced{rt: rt}
	svc := usecase.NewRefreshService(userRepo, refreshRepo, &mockTokenIssuer{})

	pair, err := svc.Refresh(context.Background(), port.RefreshCommand{RawRefreshToken: rawToken})
	require.NoError(t, err)
	assert.Equal(t, "access_token", pair.AccessToken)
	assert.Equal(t, "refresh_token", pair.RefreshToken)
	
	// Should be revoked now
	assert.NotNil(t, rt.RevokedAt)
}

func TestRefreshService_ExpiredToken_Returns401(t *testing.T) {
	rawToken := "expired_refresh"
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		Hash:      domain.HashToken(rawToken),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	refreshRepo := &mockRefreshRepoAdvanced{rt: rt}
	svc := usecase.NewRefreshService(&mockUserRepo{}, refreshRepo, &mockTokenIssuer{})

	_, err := svc.Refresh(context.Background(), port.RefreshCommand{RawRefreshToken: rawToken})
	assert.ErrorIs(t, err, response.ErrInvalidRefreshToken)
}

func TestRefreshService_RevokedToken_Returns401(t *testing.T) {
	rawToken := "revoked_refresh"
	now := time.Now()
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		Hash:      domain.HashToken(rawToken),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		RevokedAt: &now, // Revoked
	}
	refreshRepo := &mockRefreshRepoAdvanced{rt: rt}
	svc := usecase.NewRefreshService(&mockUserRepo{}, refreshRepo, &mockTokenIssuer{})

	_, err := svc.Refresh(context.Background(), port.RefreshCommand{RawRefreshToken: rawToken})
	assert.ErrorIs(t, err, response.ErrTokenRevoked)
}
