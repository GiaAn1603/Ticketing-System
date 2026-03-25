package infrastructure

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, addr string, poolSize, minIdle int) (*redis.Client, error) {
	logger := GetLogger("INFRA_REDIS")

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     poolSize,
		MinIdleConns: minIdle,
	})

	if err := redisotel.InstrumentTracing(client); err != nil {
		return nil, fmt.Errorf("instrument redis tracing: %w", err)
	}

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.Info(
		"Redis connected successfully",
		"addr", addr,
	)

	return client, nil
}
