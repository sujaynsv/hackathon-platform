package domain

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	IssuedAt  time.Time
	IsUsed    bool
}

// NewEmailVerificationToken creates a new token domain object.
// Returns the raw token string (to be sent via email) and the domain object (to be saved).
func NewEmailVerificationToken(userID uuid.UUID) (string, *EmailVerificationToken) {
	rawToken := uuid.New().String()

	h := sha256.New()
	h.Write([]byte(rawToken))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	now := time.Now().UTC()
	return rawToken, &EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: now.Add(24 * time.Hour), // 24 hour TTL
		IssuedAt:  now,
		IsUsed:    false,
	}
}

func (t *EmailVerificationToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}
