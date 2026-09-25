package port

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
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
	IssueAccessToken(userID, email string, isAdmin bool) (string, error)
	IssueRefreshToken(userID string) (string, error)
}

// RefreshTokenRepository is the outbound port for refresh token persistence.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
