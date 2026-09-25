package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
)

type LoginService struct {
	users   port.UserRepository
	hasher  port.PasswordHasher
	tokens  port.TokenIssuer
	refresh port.RefreshTokenRepository
}

func NewLoginService(
	users port.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	refresh port.RefreshTokenRepository,
) *LoginService {
	return &LoginService{users: users, hasher: hasher, tokens: tokens, refresh: refresh}
}

func (s *LoginService) Login(ctx context.Context, cmd port.LoginCommand) (*port.AuthResponse, error) {
	// 1. Find user by email
	user, err := s.users.FindByEmail(ctx, cmd.Email)
	if err != nil {
		// Use a generic error: never reveal if email exists
		return nil, response.ErrInvalidCredentials
	}

	// 2. Check account is active and verified
	if !user.IsActive {
		return nil, fmt.Errorf("%w: account is disabled", response.ErrForbidden)
	}
	if !user.IsVerified {
		return nil, fmt.Errorf("%w: account is not verified", response.ErrForbidden)
	}

	// 3. Verify password (timing-safe bcrypt compare via hasher port)
	if !s.hasher.Verify(user.PasswordHash, cmd.Password) {
		return nil, response.ErrInvalidCredentials
	}

	// 4. Issue tokens
	accessToken, _, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	rawRefreshToken, exp, err := s.tokens.IssueRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// 5. Store hashed refresh token
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Hash:      domain.HashToken(rawRefreshToken),
		ExpiresAt: exp,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.refresh.Save(ctx, rt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &port.AuthResponse{
		User: port.UserDTO{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			IsAdmin:     user.IsAdmin,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		},
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
	}, nil
}
