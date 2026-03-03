package repositories

import (
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/utils"
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

const (
	luaSuccess          = 1
	luaAlreadyProcessed = 2
	luaInvalidInput     = -1
	luaLimitExceeded    = -2
	luaOutOfStock       = -3
	luaEventNotFound    = -4
)

type RedisRepo struct {
	rdb        *redis.Client
	scriptSHA  string
	scriptBody string
}

func NewRedisRepo(ctx context.Context, rdb *redis.Client) (*RedisRepo, error) {
	sha, err := rdb.ScriptLoad(ctx, scripts.BuyTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load buy_ticket lua script: %w", err)
	}

	log.Printf("[REPO][INFO] Lua script loaded successfully | sha=%s", sha)

	return &RedisRepo{
		rdb:        rdb,
		scriptSHA:  sha,
		scriptBody: scripts.BuyTicketScript,
	}, nil
}

func (r *RedisRepo) InitializeEvent(ctx context.Context, eventID string, stock, limit int) error {
	stockKey := fmt.Sprintf("ticket:stock:%s", eventID)
	limitKey := fmt.Sprintf("ticket:limit:%s", eventID)

	created, err := r.rdb.SetNX(ctx, stockKey, stock, 0).Result()
	if err != nil {
		log.Printf("[REPO][ERROR] Failed to execute SetNX | event_id=%s | stock=%d | limit=%d | err=%v", eventID, stock, limit, err)
		return fmt.Errorf("failed to set stock in redis for event %s: %w", eventID, err)
	}
	if !created {
		log.Printf("[REPO][WARN] Event already initialized | event_id=%s | stock=%d | limit=%d", eventID, stock, limit)
		return fmt.Errorf("event %s already exists", eventID)
	}

	err = r.rdb.Set(ctx, limitKey, limit, 0).Err()
	if err != nil {
		log.Printf("[REPO][ERROR] Failed to set limit | event_id=%s | stock=%d | limit=%d | err=%v", eventID, stock, limit, err)
		return fmt.Errorf("failed to set limit for event %s: %w", eventID, err)
	}

	log.Printf("[REPO][INFO] Event initialized successfully | event_id=%s | stock=%d | limit=%d", eventID, stock, limit)

	return nil
}

func (r *RedisRepo) PurchaseTicket(ctx context.Context, eventID, userID, reqID string, qty int) error {
	keys := []string{
		fmt.Sprintf("ticket:stock:%s", eventID),
		fmt.Sprintf("ticket:limit:%s", eventID),
		fmt.Sprintf("ticket:history:%s:%s", eventID, userID),
		fmt.Sprintf("ticket:req_processed:%s:%s", eventID, reqID),
	}

	args := []interface{}{qty, 86400}

	res, err := utils.EvalShaWithFallback(ctx, r.rdb, r.scriptSHA, r.scriptBody, keys, args...).Int()
	if err != nil {
		log.Printf("[REPO][ERROR] Lua script execution failed | event_id=%s | user_id=%s | req_id=%s | qty=%d | err=%v", eventID, userID, reqID, qty, err)
		return fmt.Errorf("failed to execute buy ticket script for event %s: %w", eventID, err)
	}

	log.Printf("[REPO][INFO] Lua script result | event_id=%s | user_id=%s | req_id=%s | qty=%d | res=%d", eventID, userID, reqID, qty, res)

	switch res {
	case luaSuccess:
		return nil
	case luaAlreadyProcessed:
		return models.ErrAlreadyProcessed
	case luaInvalidInput:
		return models.ErrInvalidInput
	case luaLimitExceeded:
		return models.ErrLimitExceeded
	case luaOutOfStock:
		return models.ErrOutOfStock
	case luaEventNotFound:
		return models.ErrEventNotFound
	default:
		return fmt.Errorf("unexpected lua response code %d: %w", res, models.ErrInternal)
	}
}
