package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
)

type LogoutService struct {
	cache port.Cache
}

func NewLogoutService(cache port.Cache) *LogoutService {
	return &LogoutService{cache: cache}
}

func (s *LogoutService) Logout(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // Token already expired — blacklisting it is a no-op
	}
	// Write to Redis: SET revoked:{jti} 1 EX <ttl_seconds>
	key := fmt.Sprintf("revoked:%s", jti)
	return s.cache.Set(ctx, key, []byte("1"), ttl)
}
