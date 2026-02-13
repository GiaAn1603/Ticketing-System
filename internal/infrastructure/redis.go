package infrastructure

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Printf("[INFRA][INFO] Connected to Redis successfully | addr=%s", addr)

	return client, nil
}
