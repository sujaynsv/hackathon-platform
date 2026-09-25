package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/auth/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCache struct {
	keys map[string][]byte
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if val, ok := m.keys[key]; ok {
		return val, nil
	}
	return nil, nil // Actually should return ErrCacheMiss, but nil works for this mock
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	m.keys[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.keys, key)
	return nil
}

func TestLogoutService_ValidJTI_BlacklistsInRedis(t *testing.T) {
	cache := &mockCache{keys: make(map[string][]byte)}
	svc := usecase.NewLogoutService(cache, &mockRefreshRepo{})

	jti := "test-jti-123"
	// Expires in 1 hour
	exp := time.Now().Add(1 * time.Hour)

	err := svc.Logout(context.Background(), port.LogoutCommand{
		JTI:             jti,
		ExpiresAt:       exp,
		RawRefreshToken: "",
	})
	require.NoError(t, err)

	assert.Contains(t, cache.keys, "revoked:test-jti-123")
	assert.Equal(t, []byte("1"), cache.keys["revoked:test-jti-123"])
}

func TestLogoutService_AlreadyExpiredToken_NoOp_Returns200(t *testing.T) {
	cache := &mockCache{keys: make(map[string][]byte)}
	svc := usecase.NewLogoutService(cache, &mockRefreshRepo{})

	jti := "test-jti-456"
	// Expired 1 hour ago
	exp := time.Now().Add(-1 * time.Hour)

	err := svc.Logout(context.Background(), port.LogoutCommand{
		JTI:             jti,
		ExpiresAt:       exp,
		RawRefreshToken: "",
	})
	require.NoError(t, err)

	// Should not have touched cache
	assert.NotContains(t, cache.keys, "revoked:test-jti-456")
}
