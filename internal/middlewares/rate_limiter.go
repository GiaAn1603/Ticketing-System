package middlewares

import (
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	luaSuccess       = 1
	luaInvalidInput  = -1
	luaLimitExceeded = -2
)

type RateLimiter struct {
	rdb       *redis.Client
	capacity  int
	rate      int
	timeout   time.Duration
	scriptSHA string
}

func NewRateLimiter(ctx context.Context, rdb *redis.Client, capacity, rate int, timeout time.Duration) (*RateLimiter, error) {
	sha, err := rdb.ScriptLoad(ctx, scripts.RateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate_limit lua script: %w", err)
	}

	log.Printf("[MIDDLEWARE][INFO] Lua script loaded successfully | sha=%s", sha)

	return &RateLimiter{
		rdb:       rdb,
		capacity:  capacity,
		rate:      rate,
		timeout:   timeout,
		scriptSHA: sha,
	}, nil
}
