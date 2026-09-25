package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
)

type RegisterService struct {
	users   port.UserRepository
	hasher  port.PasswordHasher
	tokens  port.TokenIssuer
	refresh port.RefreshTokenRepository
}

func NewRegisterService(
	users port.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	refresh port.RefreshTokenRepository,
) *RegisterService {
	return &RegisterService{users: users, hasher: hasher, tokens: tokens, refresh: refresh}
}

func (s *RegisterService) Register(ctx context.Context, cmd port.RegisterCommand) (*port.AuthResponse, error) {
	// 1. Validate password length (domain rule)
	if len(cmd.Password) < 8 {
		return nil, fmt.Errorf("%w", domain.ErrWeakPassword)
	}

	// 2. Check email uniqueness
	exists, err := s.users.ExistsByEmail(ctx, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w: email already registered", response.ErrInvariantViolated) // maps to 409
	}

	// 3. Build domain user (validates email format)
	user, err := domain.NewUser(cmd.Email, cmd.DisplayName)
	if err != nil {
		return nil, err
	}

	// 4. Hash password (via port — bcrypt adapter)
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = hash

	// 5. Persist user
	if err := s.users.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	// 6. Issue tokens
	accessToken, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, err := s.tokens.IssueRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
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
		RefreshToken: refreshToken,
	}, nil
}
