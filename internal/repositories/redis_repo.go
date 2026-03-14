package repositories

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/utils"
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log/slog"

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
	rdb                *redis.Client
	buyScriptSHA       string
	buyScriptBody      string
	rollbackScriptSHA  string
	rollbackScriptBody string
	historyTTL         int
	log                *slog.Logger
}

func NewRedisRepo(ctx context.Context, rdb *redis.Client, ttl int) (*RedisRepo, error) {
	logger := infrastructure.GetLogger("REDIS_REPO")

	buySHA, err := rdb.ScriptLoad(ctx, scripts.BuyTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load buy_ticket lua script: %w", err)
	}

	rollbackSHA, err := rdb.ScriptLoad(ctx, scripts.RollbackTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rollback_ticket lua script: %w", err)
	}

	logger.Info(
		"Lua script loaded successfully",
		"buy_sha", buySHA,
		"rollback_sha", rollbackSHA,
	)

	return &RedisRepo{
		rdb:                rdb,
		buyScriptSHA:       buySHA,
		buyScriptBody:      scripts.BuyTicketScript,
		rollbackScriptSHA:  rollbackSHA,
		rollbackScriptBody: scripts.RollbackTicketScript,
		historyTTL:         ttl,
		log:                logger,
	}, nil
}

func (r *RedisRepo) InitializeEvent(ctx context.Context, eventID string, stock, maxLimit int) error {
	stockKey := fmt.Sprintf("ticket:stock:%s", eventID)
	limitKey := fmt.Sprintf("ticket:limit:%s", eventID)

	created, err := r.rdb.SetNX(ctx, stockKey, stock, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to set stock for event %s in redis: %w", eventID, err)
	}
	if !created {
		return fmt.Errorf("event %s already exists in redis", eventID)
	}

	if err = r.rdb.Set(ctx, limitKey, maxLimit, 0).Err(); err != nil {
		return fmt.Errorf("failed to set limit for event %s in redis: %w", eventID, err)
	}

	r.log.Debug(
		"Event initialized successfully",
		"event_id", eventID,
		"stock", stock,
		"max_limit", maxLimit,
	)

	return nil
}

func (r *RedisRepo) PurchaseTicket(ctx context.Context, eventID, userID, reqID string, quantity int) error {
	keys := []string{
		fmt.Sprintf("ticket:stock:%s", eventID),
		fmt.Sprintf("ticket:limit:%s", eventID),
		fmt.Sprintf("ticket:history:%s:%s", eventID, userID),
		fmt.Sprintf("ticket:req_processed:%s:%s:%s", eventID, userID, reqID),
	}

	args := []interface{}{quantity, r.historyTTL}

	result, err := utils.EvalShaWithFallback(ctx, r.rdb, r.buyScriptSHA, r.buyScriptBody, keys, args...).Int()
	if err != nil {
		return fmt.Errorf("failed to execute buy ticket script for event %s in redis: %w", eventID, err)
	}

	r.log.Debug(
		"Purchase Lua script executed",
		"event_id", eventID,
		"user_id", userID,
		"request_id", reqID,
		"quantity", quantity,
		"result", result,
	)

	switch result {
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
		return fmt.Errorf("unexpected lua response code %d: %w", result, models.ErrInternal)
	}
}

func (r *RedisRepo) RollbackPurchase(ctx context.Context, eventID, userID, reqID string, quantity int) error {
	keys := []string{
		fmt.Sprintf("ticket:stock:%s", eventID),
		fmt.Sprintf("ticket:history:%s:%s", eventID, userID),
		fmt.Sprintf("ticket:req_processed:%s:%s:%s", eventID, userID, reqID),
	}

	args := []interface{}{quantity}

	res, err := utils.EvalShaWithFallback(ctx, r.rdb, r.rollbackScriptSHA, r.rollbackScriptBody, keys, args...).Int()
	if err != nil {
		return fmt.Errorf("failed to execute rollback script for event %s: %w", eventID, err)
	}
	if res == 0 {
		r.log.Debug(
			"Rollback skipped",
			"event_id", eventID,
			"request_id", reqID,
			"quantity", quantity,
		)
		return nil
	}

	r.log.Debug(
		"Redis rollback completed successfully",
		"event_id", eventID,
		"user_id", userID,
		"request_id", reqID,
		"quantity", quantity,
	)

	return nil
}
