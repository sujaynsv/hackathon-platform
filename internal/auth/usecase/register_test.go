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

type mockUserRepo struct {
	exists bool
	saved  *domain.User
}

func (m *mockUserRepo) Save(ctx context.Context, user *domain.User) error {
	m.saved = user
	return nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	m.saved = user
	return nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.exists {
		if m.saved != nil {
			return m.saved, nil
		}
		return &domain.User{}, nil
	}
	return nil, response.ErrNotFound
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return m.exists, nil
}

type mockHasher struct{}

func (m *mockHasher) Hash(plain string) (string, error) { return "hashed_" + plain, nil }
func (m *mockHasher) Verify(hash, plain string) bool    { return true }

type mockTokenIssuer struct{}

func (m *mockTokenIssuer) IssueAccessToken(userID, email string, isAdmin bool) (string, error) {
	return "access_token", nil
}
func (m *mockTokenIssuer) IssueRefreshToken(userID string) (string, error) {
	return "refresh_token", nil
}

type mockRefreshRepo struct{}

func (m *mockRefreshRepo) Save(ctx context.Context, token *domain.RefreshToken) error { return nil }
func (m *mockRefreshRepo) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	return nil, nil
}
func (m *mockRefreshRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error { return nil }

type mockPasswordValidator struct {
	compromised bool
	err         error
}

func (m *mockPasswordValidator) IsCompromised(ctx context.Context, password string) (bool, error) {
	return m.compromised, m.err
}

func TestRegisterService_ValidInput_ReturnsAuthResponse(t *testing.T) {
	repo := &mockUserRepo{}
	svc := usecase.NewRegisterService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{}, &mockPasswordValidator{}, nil, nil)

	resp, err := svc.Register(context.Background(), port.RegisterCommand{
		Email:       "alice@example.com",
		Password:    "Password123!",
		DisplayName: "Alice",
	})

	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", resp.User.Email)
	assert.Equal(t, "Alice", resp.User.DisplayName)
	assert.Equal(t, "access_token", resp.AccessToken)
	assert.Equal(t, "refresh_token", resp.RefreshToken)
	assert.NotNil(t, repo.saved)
	assert.Equal(t, "hashed_Password123!", repo.saved.PasswordHash)
}

func TestRegisterService_DuplicateEmail_ReturnsErrDuplicate(t *testing.T) {
	repo := &mockUserRepo{exists: true, saved: &domain.User{IsVerified: true}}
	svc := usecase.NewRegisterService(repo, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{}, &mockPasswordValidator{}, nil, nil)

	_, err := svc.Register(context.Background(), port.RegisterCommand{
		Email:       "alice@example.com",
		Password:    "Password123!",
		DisplayName: "Alice",
	})

	assert.ErrorIs(t, err, response.ErrDuplicate)
}

func TestRegisterService_WeakPassword_ReturnsErrWeakPassword(t *testing.T) {
	svc := usecase.NewRegisterService(&mockUserRepo{}, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{}, &mockPasswordValidator{}, nil, nil)

	_, err := svc.Register(context.Background(), port.RegisterCommand{
		Email:       "alice@example.com",
		Password:    "abc", // < 8 chars
		DisplayName: "Alice",
	})

	assert.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestRegisterService_CompromisedPassword_ReturnsErrWeakPassword(t *testing.T) {
	validator := &mockPasswordValidator{compromised: true}
	svc := usecase.NewRegisterService(&mockUserRepo{}, &mockHasher{}, &mockTokenIssuer{}, &mockRefreshRepo{}, validator, nil, nil)

	_, err := svc.Register(context.Background(), port.RegisterCommand{
		Email:       "alice@example.com",
		Password:    "Password123!",
		DisplayName: "Alice",
	})

	assert.ErrorIs(t, err, domain.ErrWeakPassword)
	assert.ErrorContains(t, err, "password has appeared in a data breach")
}
