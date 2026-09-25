package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
)

type LogoutService struct {
	cache   port.Cache
	refresh port.RefreshTokenRepository
}

func NewLogoutService(cache port.Cache, refresh port.RefreshTokenRepository) *LogoutService {
	return &LogoutService{cache: cache, refresh: refresh}
}

func (s *LogoutService) Logout(ctx context.Context, cmd port.LogoutCommand) error {
	ttl := time.Until(cmd.ExpiresAt)
	if ttl > 0 {
		// Write to Redis: SET revoked:{jti} 1 EX <ttl_seconds>
		key := fmt.Sprintf("revoked:%s", cmd.JTI)
		if err := s.cache.Set(ctx, key, []byte("1"), ttl); err != nil {
			return fmt.Errorf("blacklist access token: %w", err)
		}
	}

	if cmd.RawRefreshToken != "" {
		tokenHash := domain.HashToken(cmd.RawRefreshToken)
		rt, err := s.refresh.FindByHash(ctx, tokenHash)
		if err == nil && rt != nil {
			_ = s.refresh.Revoke(ctx, rt.ID)
		}
	}

	return nil
}
