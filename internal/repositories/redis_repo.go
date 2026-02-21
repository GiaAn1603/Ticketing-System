package repositories

import (
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	rdb       *redis.Client
	scriptSHA string
}

func NewRedisRepo(ctx context.Context, rdb *redis.Client) (*RedisRepo, error) {
	sha, err := rdb.ScriptLoad(ctx, scripts.BuyTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load buy_ticket lua script: %w", err)
	}

	log.Printf("[REPO][INFO] Lua script loaded successfully | sha=%s", sha)

	return &RedisRepo{
		rdb:       rdb,
		scriptSHA: sha,
	}, nil
}
