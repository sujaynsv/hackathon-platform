package usecase_test

import (
	"context"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/auth/usecase"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEmailTokenRepo struct {
	token *domain.EmailVerificationToken
	used  bool
}

func (m *mockEmailTokenRepo) Save(ctx context.Context, token *domain.EmailVerificationToken) error {
	m.token = token
	return nil
}

func (m *mockEmailTokenRepo) FindByHash(ctx context.Context, hash string) (*domain.EmailVerificationToken, error) {
	if m.token != nil && m.token.TokenHash == hash {
		return m.token, nil
	}
	return nil, response.ErrNotFound
}

func (m *mockEmailTokenRepo) MarkUsed(ctx context.Context, tokenID uuid.UUID) error {
	if m.token != nil && m.token.ID == tokenID {
		m.token.IsUsed = true
		m.used = true
	}
	return nil
}

type mockUserRepoForVerify struct {
	user *domain.User
}

func (m *mockUserRepoForVerify) Save(ctx context.Context, user *domain.User) error   { return nil }
func (m *mockUserRepoForVerify) Update(ctx context.Context, user *domain.User) error { return nil }
func (m *mockUserRepoForVerify) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepoForVerify) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, response.ErrNotFound
}
func (m *mockUserRepoForVerify) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func TestVerifyEmailService_Verify_Success(t *testing.T) {
	userID := uuid.New()
	user, _ := domain.NewUser("test@example.com", "Test")
	user.ID = userID

	rawToken, tokenDomain := domain.NewEmailVerificationToken(userID)

	tokens := &mockEmailTokenRepo{token: tokenDomain}
	users := &mockUserRepoForVerify{user: user}

	svc := usecase.NewVerifyEmailService(tokens, users)

	err := svc.Verify(context.Background(), port.VerifyEmailCommand{Token: rawToken})
	require.NoError(t, err)

	assert.True(t, user.IsVerified)
	assert.NotNil(t, user.VerifiedAt)
	assert.True(t, tokens.used)
}

func TestVerifyEmailService_Verify_InvalidToken(t *testing.T) {
	tokens := &mockEmailTokenRepo{}
	users := &mockUserRepoForVerify{}

	svc := usecase.NewVerifyEmailService(tokens, users)

	err := svc.Verify(context.Background(), port.VerifyEmailCommand{Token: "invalid_token"})
	require.ErrorIs(t, err, response.ErrNotFound)
}
