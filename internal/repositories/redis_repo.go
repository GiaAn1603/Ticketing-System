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

func (r *RedisRepo) InitializeEvent(ctx context.Context, eventID string, stock int) error {
	key := fmt.Sprintf("ticket:stock:%s", eventID)

	created, err := r.rdb.SetNX(ctx, key, stock, 0).Result()

	if err != nil {
		log.Printf("[REPO][ERROR] Failed to execute SetNX | event_id=%s | err=%v", eventID, err)
		return fmt.Errorf("failed to set stock in redis for event %s: %w", eventID, err)
	}

	if !created {
		log.Printf("[REPO][WARN] Event already initialized | event_id=%s", eventID)
		return fmt.Errorf("event %s already exists", eventID)
	}

	log.Printf("[REPO][INFO] Event initialized successfully | event_id=%s | stock=%d", eventID, stock)

	return nil
}
