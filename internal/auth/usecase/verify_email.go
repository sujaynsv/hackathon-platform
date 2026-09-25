package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
)

type VerifyEmailService struct {
	tokens port.EmailVerificationRepository
	users  port.UserRepository
}

func NewVerifyEmailService(tokens port.EmailVerificationRepository, users port.UserRepository) *VerifyEmailService {
	return &VerifyEmailService{tokens: tokens, users: users}
}

func (s *VerifyEmailService) Verify(ctx context.Context, cmd port.VerifyEmailCommand) error {
	// 1. Hash the provided token
	h := sha256.New()
	h.Write([]byte(cmd.Token))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	// 2. Find the token
	token, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		return err // Could be ErrNotFound, mapped in handler
	}

	// 3. Check if used or expired
	if token.IsUsed {
		return fmt.Errorf("%w: token already used", response.ErrInvalidTransition)
	}
	if token.IsExpired() {
		return fmt.Errorf("%w: token expired", response.ErrDeadlinePassed)
	}

	// 4. Find the user
	user, err := s.users.FindByID(ctx, token.UserID)
	if err != nil {
		return err
	}

	// 5. Update domain
	user.Verify()

	// 6. Save changes
	// Note: in a real system we'd wrap this in a transaction.
	// For this modular monolith, since we don't have transaction wrappers exposed cleanly here yet,
	// we do it sequentially.
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}
	if err := s.tokens.MarkUsed(ctx, token.ID); err != nil {
		return err
	}

	return nil
}
