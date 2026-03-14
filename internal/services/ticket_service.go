package services

import (
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/repositories"
	"context"
	"fmt"
	"log/slog"
	"time"
)

type TicketService struct {
	redisRepo       *repositories.RedisRepo
	producer        *events.KafkaProducer
	rollbackTimeout time.Duration
	log             *slog.Logger
}

func NewTicketService(redisRepo *repositories.RedisRepo, producer *events.KafkaProducer, rollbackTimeout time.Duration) *TicketService {
	logger := infrastructure.GetLogger("SERVICE")

	return &TicketService{
		redisRepo:       redisRepo,
		producer:        producer,
		rollbackTimeout: rollbackTimeout,
		log:             logger,
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
		return fmt.Errorf("failed to initialize event %s in redis: %w", eventID, err)
	}

	return nil
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, quantity int) error {
	s.log.Debug(
		"Purchase processing",
		"event_id", eventID,
		"user_id", userID,
		"request_id", reqID,
		"quantity", quantity,
	)

	if err := s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, quantity); err != nil {
		return fmt.Errorf("failed to process purchase for request %s in redis: %w", reqID, err)
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

		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), s.rollbackTimeout)
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
		}

		return fmt.Errorf("failed to publish event to kafka for request %s: %w", reqID, err)
	}

	return nil
}
