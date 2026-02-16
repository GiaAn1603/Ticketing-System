package repositories

import (
	"Ticketing-System/scripts"
	"context"
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
		return nil, err
	}

	log.Printf("[REPO][INFO] Lua script loaded successfully | sha=%s", sha)

	return &RedisRepo{
		rdb:       rdb,
		scriptSHA: sha,
	}, nil
}
