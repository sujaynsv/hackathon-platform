package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding window rate limiter using Redis.
type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// RateLimit is a middleware that rate limits requests based on IP.
func (rl *RateLimiter) RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP properly
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			// (Optional: if behind trusted proxy, read X-Forwarded-For only if configured to trust it)

			key := fmt.Sprintf("ratelimit:%s:%s", r.URL.Path, ip)

			ctx := r.Context()
			now := time.Now()
			windowStart := now.Add(-window).UnixMilli()

			// Lua script or multi-exec could be used, but standard pipeline is okay for this.
			// Using ZSET for sliding window
			pipe := rl.client.TxPipeline()

			// Remove older entries
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

			// Add current request
			pipe.ZAdd(ctx, key, redis.Z{
				Score:  float64(now.UnixMilli()),
				Member: now.UnixNano(),
			})

			// Count requests in window
			countCmd := pipe.ZCard(ctx, key)

			// Update expiration
			pipe.Expire(ctx, key, window)

			_, err = pipe.Exec(ctx)
			if err != nil {
				slog.Error("Redis rate limit pipeline failed", "error", err)
				response.HandleDomainError(w, r, response.ErrRateLimited)
				return
			}

			count := countCmd.Val()
			if count > int64(limit) {
				response.HandleDomainError(w, r, response.ErrRateLimited)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
