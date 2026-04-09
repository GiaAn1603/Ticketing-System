package services

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/repositories"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type TicketService struct {
	redisRepo    *repositories.RedisRepo
	producer     *events.KafkaProducer
	soldOutCache *expirable.LRU[string, bool]
	cfg          config.TicketServiceConfig
	log          *slog.Logger
}

func NewTicketService(redisRepo *repositories.RedisRepo, producer *events.KafkaProducer, cfg config.TicketServiceConfig) *TicketService {
	logger := infrastructure.GetLogger("SERVICE")

	cache := expirable.NewLRU[string, bool](cfg.SoldOutMaxSize, nil, cfg.SoldOutTTL)

	return &TicketService{
		redisRepo:    redisRepo,
		producer:     producer,
		soldOutCache: cache,
		cfg:          cfg,
		log:          logger,
	}
}

func (s *TicketService) InitializeEvent(ctx context.Context, eventID string, stock, maxLimit int) error {
	s.log.Debug(
		"Init event processing",
		"event_id", eventID,
		"stock", stock,
		"max_limit", maxLimit,
	)

	if err := s.redisRepo.InitializeEvent(ctx, eventID, stock, maxLimit); err != nil {
		return fmt.Errorf("initialize event: %w", err)
	}

	return nil
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, quantity int) error {
	tr := otel.Tracer("ticket-service")
	ctx, span := tr.Start(ctx, "process_purchase")
	span.SetAttributes(
		attribute.String("event_id", eventID),
		attribute.String("user_id", userID),
		attribute.String("request_id", reqID),
	)
	defer span.End()

	if _, isSoldOut := s.soldOutCache.Get(eventID); isSoldOut {
		s.log.Warn(
			"Stock availability validation failed",
			"event_id", eventID,
		)
		return models.ErrOutOfStock
	}

	s.log.Debug(
		"Purchase processing",
		"event_id", eventID,
		"user_id", userID,
		"request_id", reqID,
		"quantity", quantity,
	)

	if err := s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, quantity); err != nil {
		if errors.Is(err, models.ErrOutOfStock) {
			s.soldOutCache.Add(eventID, true)
		}
		return fmt.Errorf("process purchase: %w", err)
	}

	event := models.OrderEvent{
		EventID:   eventID,
		UserID:    userID,
		RequestID: reqID,
		Quantity:  quantity,
		Status:    "Processing",
		Timestamp: time.Now(),
	}

	if err := s.producer.PublishOrderEvent(ctx, event); err != nil {
		s.log.Error(
			"Kafka publish failed",
			"event_id", eventID,
			"user_id", userID,
			"request_id", reqID,
			"quantity", quantity,
			infrastructure.KeyError, err.Error(),
		)

		detachedCtx := context.WithoutCancel(ctx)
		rollbackCtx, rollbackCancel := context.WithTimeout(detachedCtx, s.cfg.RollbackTimeout)
		defer rollbackCancel()

		if rbErr := s.redisRepo.RollbackPurchase(rollbackCtx, eventID, userID, reqID, quantity); rbErr != nil {
			s.log.Error(
				"Dual write rollback to Redis failed",
				"event_id", eventID,
				"user_id", userID,
				"request_id", reqID,
				"quantity", quantity,
				infrastructure.KeyError, rbErr.Error(),
			)
		} else {
			s.soldOutCache.Remove(eventID)

			s.log.Info(
				"Sold-out cache removed successfully",
				"event_id", eventID,
				"user_id", userID,
				"request_id", reqID,
				"quantity", quantity,
			)
		}

		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}
