package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
)

type RegisterService struct {
	users     port.UserRepository
	hasher    port.PasswordHasher
	tokens    port.TokenIssuer
	refresh   port.RefreshTokenRepository
	validator port.PasswordValidator
	emailTks  port.EmailVerificationRepository
	sender    port.EmailSender
}

func NewRegisterService(
	users port.UserRepository,
	hasher port.PasswordHasher,
	tokens port.TokenIssuer,
	refresh port.RefreshTokenRepository,
	validator port.PasswordValidator,
	emailTks port.EmailVerificationRepository,
	sender port.EmailSender,
) *RegisterService {
	return &RegisterService{
		users:     users,
		hasher:    hasher,
		tokens:    tokens,
		refresh:   refresh,
		validator: validator,
		emailTks:  emailTks,
		sender:    sender,
	}
}

func (s *RegisterService) Register(ctx context.Context, cmd port.RegisterCommand) (*port.AuthResponse, error) {
	// 1. Validate password length (domain rule)
	if len(cmd.Password) < 8 || len(cmd.Password) > 72 {
		return nil, fmt.Errorf("%w", domain.ErrWeakPassword)
	}

	// 1.5 Check if password is breached
	if s.validator != nil {
		compromised, err := s.validator.IsCompromised(ctx, cmd.Password)
		if err != nil {
			return nil, fmt.Errorf("check breached password: %w", err)
		}
		if compromised {
			return nil, fmt.Errorf("password has appeared in a data breach: %w", domain.ErrWeakPassword)
		}
	}

	// 2. Check email uniqueness and resend logic
	existingUser, err := s.users.FindByEmail(ctx, cmd.Email)
	if err == nil {
		if existingUser.IsVerified {
			return nil, fmt.Errorf("%w: email already registered", response.ErrDuplicate)
		}
		// If unverified, we update password and resend token instead of failing
		hash, err := s.hasher.Hash(cmd.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		existingUser.PasswordHash = hash
		existingUser.DisplayName = cmd.DisplayName
		if err := s.users.Update(ctx, existingUser); err != nil {
			return nil, fmt.Errorf("update user: %w", err)
		}
		
		if s.emailTks != nil && s.sender != nil {
			rawToken, tokenDomain := domain.NewEmailVerificationToken(existingUser.ID)
			if err := s.emailTks.Save(ctx, tokenDomain); err != nil {
				return nil, fmt.Errorf("save email token: %w", err)
			}
			_ = s.sender.SendVerificationEmail(ctx, existingUser.Email, rawToken)
		}
		
		return &port.AuthResponse{
			User: port.UserDTO{
				ID:          existingUser.ID.String(),
				Email:       existingUser.Email,
				DisplayName: existingUser.DisplayName,
				AvatarURL:   existingUser.AvatarURL,
				IsAdmin:     existingUser.IsAdmin,
				CreatedAt:   existingUser.CreatedAt.Format(time.RFC3339),
			},
			RequiresVerification: true,
		}, nil
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

	// 5.5 Generate and send verification email
	if s.emailTks != nil && s.sender != nil {
		rawToken, tokenDomain := domain.NewEmailVerificationToken(user.ID)
		if err := s.emailTks.Save(ctx, tokenDomain); err != nil {
			return nil, fmt.Errorf("save email token: %w", err)
		}
		// In a real system, we'd send this async. We'll do it synchronously for now.
		if err := s.sender.SendVerificationEmail(ctx, user.Email, rawToken); err != nil {
			// Don't fail the whole registration if email fails, but log it (or handle it)
			// Returning err here might be too harsh, but let's be strict for A-006 testability
			return nil, fmt.Errorf("send verification email: %w", err)
		}
	}

	requiresVerification := s.emailTks != nil && s.sender != nil
	if requiresVerification {
		return &port.AuthResponse{
			User: port.UserDTO{
				ID:          user.ID.String(),
				Email:       user.Email,
				DisplayName: user.DisplayName,
				AvatarURL:   user.AvatarURL,
				IsAdmin:     user.IsAdmin,
				CreatedAt:   user.CreatedAt.Format(time.RFC3339),
			},
			RequiresVerification: true,
		}, nil
	}

	// 6. Issue tokens (only if verification is not required)
	accessToken, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, err := s.tokens.IssueRefreshToken(user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(refreshToken))
	rtHash := fmt.Sprintf("%x", h.Sum(nil))

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Hash:      rtHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
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
		RefreshToken: refreshToken,
	}, nil
}
