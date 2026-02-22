package repositories

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"Ticketing-System/pkg/apperrors"
	"Ticketing-System/scripts"
)

const (
	luaSuccess               = 1
	luaAlreadyProcessed      = 2
	luaInvalidInput          = -1000
	luaEventNotFound         = -2001
	luaOutOfStock            = -2002
	luaPurchaseLimitExceeded = -2003
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
		log.Printf("[REPO][ERROR] Failed to execute SetNX | event_id=%s | stock=%d | err=%v", eventID, stock, err)
		return fmt.Errorf("failed to set stock in redis for event %s: %w", eventID, err)
	}

	if !created {
		log.Printf("[REPO][WARN] Event already initialized | event_id=%s | stock=%d", eventID, stock)
		return fmt.Errorf("event %s already exists", eventID)
	}

	log.Printf("[REPO][INFO] Event initialized successfully | event_id=%s | stock=%d", eventID, stock)

	return nil
}

func (r *RedisRepo) PurchaseTicket(ctx context.Context, eventID, userID, reqID string, qty, limit int) error {
	keys := []string{
		fmt.Sprintf("ticket:stock:%s", eventID),
		fmt.Sprintf("ticket:history:%s:%s", eventID, userID),
		fmt.Sprintf("ticket:req_processed:%s:%s", eventID, reqID),
	}

	args := []interface{}{qty, limit, 86400}

	res, err := r.rdb.EvalSha(ctx, r.scriptSHA, keys, args...).Int()
	if err != nil {
		log.Printf("[REPO][ERROR] Lua script execution failed | event_id=%s | user_id=%s | req_id=%s | qty=%d | limit=%d | err=%v", eventID, userID, reqID, qty, limit, err)
		return err
	}

	log.Printf("[REPO][INFO] Lua script result | event_id=%s | user_id=%s | req_id=%s | qty=%d | limit=%d | res=%d", eventID, userID, reqID, qty, limit, res)

	switch res {
	case luaSuccess:
		return nil
	case luaAlreadyProcessed:
		return apperrors.ErrAlreadyProcessed
	case luaInvalidInput:
		return apperrors.ErrInvalidInput
	case luaEventNotFound:
		return apperrors.ErrEventNotFound
	case luaOutOfStock:
		return apperrors.ErrOutOfStock
	case luaPurchaseLimitExceeded:
		return apperrors.ErrPurchaseLimitExceeded
	default:
		return apperrors.ErrInternal
	}
}
