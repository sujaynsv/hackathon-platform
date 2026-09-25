package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRateLimiter(t *testing.T) {
	ctx := context.Background()

	// Start Redis container
	redisC, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer redisC.Terminate(ctx) //nolint:errcheck

	uri, err := redisC.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := goredis.ParseURL(uri)
	require.NoError(t, err)
	rdb := goredis.NewClient(opts)
	defer rdb.Close() //nolint:errcheck

	limiter := middleware.NewRateLimiter(rdb)

	// Create a dummy handler that returns 200 OK
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with rate limiter middleware: 5 requests per 15 minutes
	limit := 5
	window := 15 * time.Minute
	handler := limiter.RateLimit(limit, window)(dummyHandler)

	// Make 5 successful requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
		req.RemoteAddr = "192.168.1.1:1234" // same IP
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}

	// 6th request should be rate limited (429)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	req.RemoteAddr = "192.168.1.1:1234" // same IP
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "6th request should be rate limited")
	assert.Contains(t, rec.Body.String(), "RATE_LIMITED")

	// Different IP should still be allowed
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code, "different IP should be allowed")
}
