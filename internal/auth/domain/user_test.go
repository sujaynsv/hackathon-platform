package domain_test

import (
	"testing"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser_ValidInput_ReturnsUser(t *testing.T) {
	email := "  ALICE@Example.com  "
	displayName := "Alice Chen"

	user, err := domain.NewUser(email, displayName)

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice Chen", user.DisplayName)
	assert.True(t, user.IsActive)
	assert.False(t, user.IsAdmin)
	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
}

func TestNewUser_EmptyDisplayName_ReturnsError(t *testing.T) {
	_, err := domain.NewUser("alice@example.com", "   ")
	assert.ErrorContains(t, err, "display name is required")
}

func TestNewUser_InvalidEmail_ReturnsErrInvalidEmail(t *testing.T) {
	tests := []string{
		"notanemail",
		"missingat.com",
		"@missinglocal.com",
		"missingdomain@",
	}

	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			_, err := domain.NewUser(tc, "Alice Chen")
			assert.ErrorIs(t, err, domain.ErrInvalidEmail)
		})
	}
}
