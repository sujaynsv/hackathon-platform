package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MustConnect establishes a connection to Redis and panics if it fails.
func MustConnect(url string) *redis.Client {
	opts, err := redis.ParseURL(url)
	if err != nil {
		panic(fmt.Sprintf("invalid redis url: %v", err))
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("failed to connect to redis: %v", err))
	}

	return client
}
