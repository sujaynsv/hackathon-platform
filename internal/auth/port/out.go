package port

import (
	"context"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, user *domain.User) error
}

// PasswordHasher is the outbound port for password hashing.
// Backed by bcrypt adapter — use case never imports golang.org/x/crypto.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(hash, plain string) bool
}

// PasswordValidator checks if a password meets external security criteria (e.g., not breached).
type PasswordValidator interface {
	IsCompromised(ctx context.Context, password string) (bool, error)
}

// TokenIssuer is the outbound port for JWT generation.
type TokenIssuer interface {
	IssueAccessToken(userID, email string, isAdmin bool) (string, time.Time, error)
	IssueRefreshToken(userID string) (string, time.Time, error)
}

// RefreshTokenRepository is the outbound port for refresh token persistence.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, email, token string) error
}

type EmailVerificationRepository interface {
	Save(ctx context.Context, token *domain.EmailVerificationToken) error
	FindByHash(ctx context.Context, hash string) (*domain.EmailVerificationToken, error)
	MarkUsed(ctx context.Context, tokenID uuid.UUID) error
}

type WebAuthnRepository interface {
	Save(ctx context.Context, cred *domain.WebAuthnCredential) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.WebAuthnCredential, error)
	Update(ctx context.Context, cred *domain.WebAuthnCredential) error
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

type CaptchaValidator interface {
	Verify(ctx context.Context, token string, remoteIP string) (bool, error)
}
