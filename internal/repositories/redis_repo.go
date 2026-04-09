package repositories

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"

	"Ticketing-System/internal/config"
	"Ticketing-System/internal/infrastructure/observability"
	"Ticketing-System/internal/infrastructure/resiliency"
	"Ticketing-System/internal/utils"
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
	rdb                *redis.Client
	cb                 *gobreaker.CircuitBreaker
	buyScriptSHA       string
	buyScriptBody      string
	rollbackScriptSHA  string
	rollbackScriptBody string
	cfg                config.RedisRepoConfig
	log                *slog.Logger
}

func NewRedisRepo(ctx context.Context, rdb *redis.Client, cfg config.RedisRepoConfig) (*RedisRepo, error) {
	logger := observability.GetLogger("REDIS_REPO")

	buySHA, err := rdb.ScriptLoad(ctx, scripts.BuyTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load buy_ticket script: %w", err)
	}

	rollbackSHA, err := rdb.ScriptLoad(ctx, scripts.RollbackTicketScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load rollback script: %w", err)
	}

	logger.Info(
		"Lua script loaded successfully",
		"buy_sha", buySHA,
		"rollback_sha", rollbackSHA,
	)

	cb := resiliency.NewCircuitBreaker(logger, "Redis_Repo_CB", cfg.CBConfig)

	return &RedisRepo{
		rdb:                rdb,
		cb:                 cb,
		buyScriptSHA:       buySHA,
		buyScriptBody:      scripts.BuyTicketScript,
		rollbackScriptSHA:  rollbackSHA,
		rollbackScriptBody: scripts.RollbackTicketScript,
		cfg:                cfg,
		log:                logger,
	}, nil
}

func (r *RedisRepo) InitializeEvent(ctx context.Context, eventID string, stock, maxLimit int) error {
	stockKey := fmt.Sprintf("ticket:stock:%s", eventID)
	limitKey := fmt.Sprintf("ticket:limit:%s", eventID)

	created, err := r.rdb.SetNX(ctx, stockKey, stock, 0).Result()
	if err != nil {
		return fmt.Errorf("set stock: %w", err)
	}
	if !created {
		return fmt.Errorf("event already exists")
	}

	if err = r.rdb.Set(ctx, limitKey, maxLimit, 0).Err(); err != nil {
		return fmt.Errorf("set limit: %w", err)
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

	args := []interface{}{quantity, r.cfg.HistoryTTL}

	rawResult, err := r.cb.Execute(func() (interface{}, error) {
		return utils.EvalShaWithFallback(ctx, r.rdb, r.buyScriptSHA, r.buyScriptBody, keys, args...).Int()
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return fmt.Errorf("pass circuit breaker: %w", apperrors.ErrInternal)
		}

		return fmt.Errorf("execute buy script: %w", err)
	}

	result := rawResult.(int)

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
		return fmt.Errorf("unexpected response code %d: %w", result, apperrors.ErrInternal)
	}
}

func (r *RedisRepo) RollbackPurchase(ctx context.Context, eventID, userID, reqID string, quantity int) error {
	keys := []string{
		fmt.Sprintf("ticket:stock:%s", eventID),
		fmt.Sprintf("ticket:history:%s:%s", eventID, userID),
		fmt.Sprintf("ticket:req_processed:%s:%s:%s", eventID, userID, reqID),
	}

	args := []interface{}{quantity}

	rawResult, err := r.cb.Execute(func() (interface{}, error) {
		return utils.EvalShaWithFallback(ctx, r.rdb, r.rollbackScriptSHA, r.rollbackScriptBody, keys, args...).Int()
	})
	if err != nil {
		return fmt.Errorf("execute rollback script: %w", err)
	}

	result := rawResult.(int)
	if result == 0 {
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
