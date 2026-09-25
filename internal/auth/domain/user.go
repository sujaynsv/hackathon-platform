package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Domain errors — use fmt.Errorf("%w: detail", domain.ErrXxx) to wrap with context.
var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password too short: minimum 8 characters")
	ErrInvalidDisplayName = errors.New("display name is required")
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  string
	AvatarURL    *string
	IsActive     bool
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Hash      string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// NewUser validates and constructs a new User (without saving).
// All validation lives here — not in the handler or use case.
func NewUser(email, displayName string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}
	now := time.Now().UTC()
	return &User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: displayName,
		IsActive:    true,
		IsAdmin:     false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func isValidEmail(email string) bool {
	// Simple check: contains exactly one @ with non-empty parts before and after
	parts := strings.Split(email, "@")
	return len(parts) == 2 && len(parts[0]) > 0 && strings.Contains(parts[1], ".")
}
