package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
)

type RefreshService struct {
	users   port.UserRepository
	refresh port.RefreshTokenRepository
	tokens  port.TokenIssuer
}

func NewRefreshService(
	users port.UserRepository,
	refresh port.RefreshTokenRepository,
	tokens port.TokenIssuer,
) *RefreshService {
	return &RefreshService{
		users:   users,
		refresh: refresh,
		tokens:  tokens,
	}
}

func (s *RefreshService) Refresh(ctx context.Context, cmd port.RefreshCommand) (*port.TokenPair, error) {
	tokenHash := domain.HashToken(cmd.RawRefreshToken)

	// 1. Find the refresh token by its hash
	rt, err := s.refresh.FindByHash(ctx, tokenHash)
	if err != nil {
		return nil, response.ErrInvalidRefreshToken
	}

	// 2. Check if expired or revoked
	if rt.RevokedAt != nil {
		return nil, response.ErrTokenRevoked
	}
	if rt.IsExpired() {
		return nil, response.ErrInvalidRefreshToken
	}

	// 3. Load user
	user, err := s.users.FindByID(ctx, rt.UserID)
	if err != nil || !user.IsActive {
		return nil, fmt.Errorf("%w: user not found or disabled", response.ErrUnauthorized)
	}

	// 4. Revoke the old refresh token (rotation)
	if err := s.refresh.Revoke(ctx, rt.ID); err != nil {
		if errors.Is(err, response.ErrTokenRevoked) {
			return nil, response.ErrTokenRevoked
		}
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	// 5. Issue new tokens
	newAccess, _, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	newRaw, exp, err := s.tokens.IssueRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// 6. Store new refresh token
	newRT := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Hash:      domain.HashToken(newRaw),
		ExpiresAt: exp,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.refresh.Save(ctx, newRT); err != nil {
		return nil, fmt.Errorf("save new refresh token: %w", err)
	}

	return &port.TokenPair{
		AccessToken:  newAccess,
		RefreshToken: newRaw,
		User: port.UserDTO{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			IsAdmin:     user.IsAdmin,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}
