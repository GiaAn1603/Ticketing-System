package services

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"Ticketing-System/internal/config"
	"Ticketing-System/internal/infrastructure/observability"
	"Ticketing-System/internal/models"
)

type PostgresRepository interface {
	CreateOrder(ctx context.Context, order models.OrderEvent) error
}

type OrderService struct {
	pgRepo PostgresRepository
	cfg    config.OrderServiceConfig
	log    *slog.Logger
}

func NewOrderService(pgRepo PostgresRepository, cfg config.OrderServiceConfig) *OrderService {
	logger := observability.GetLogger("ORDER_SERVICE")

	return &OrderService{
		pgRepo: pgRepo,
		cfg:    cfg,
		log:    logger,
	}
}

func (s *OrderService) ProcessOrder(ctx context.Context, event models.OrderEvent) error {
	s.log.Debug(
		"Order processing",
		"event_id", event.EventID,
		"user_id", event.UserID,
		"request_id", event.RequestID,
		"quantity", event.Quantity,
		"order_status", event.Status,
	)

	event.Status = "Success"

	for {
		dbCtx, dbCancel := context.WithTimeout(ctx, s.cfg.DBTimeout)
		err := s.pgRepo.CreateOrder(dbCtx, event)
		dbCancel()

		if err == nil {
			return nil
		}

		s.log.Error(
			"Order insertion failed",
			"event_id", event.EventID,
			"user_id", event.UserID,
			"request_id", event.RequestID,
			"quantity", event.Quantity,
			"order_status", event.Status,
			observability.KeyError, err.Error(),
		)

		var jitter time.Duration
		if s.cfg.BackoffJitter > 0 {
			jitter = time.Duration(rand.Intn(s.cfg.BackoffJitter)) * time.Millisecond
		}

		select {
		case <-ctx.Done():
			s.log.Warn(
				"Database retry loop failed",
				"event_id", event.EventID,
				"user_id", event.UserID,
				"request_id", event.RequestID,
				"quantity", event.Quantity,
				"order_status", event.Status,
				observability.KeyError, ctx.Err().Error(),
			)
			return ctx.Err()
		case <-time.After(s.cfg.BackoffBase + jitter):
		}
	}
}
