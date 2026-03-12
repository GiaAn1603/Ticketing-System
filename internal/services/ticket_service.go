package services

import (
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/repositories"
	"context"
	"fmt"
	"log"
	"time"
)

type TicketService struct {
	redisRepo *repositories.RedisRepo
	producer  *events.KafkaProducer
}

func NewTicketService(redisRepo *repositories.RedisRepo, producer *events.KafkaProducer) *TicketService {
	return &TicketService{
		redisRepo: redisRepo,
		producer:  producer,
	}
}

func (s *TicketService) InitializeEvent(ctx context.Context, eventID string, stock, limit int) error {
	log.Printf("[SERVICE][INFO] Processing init event | event_id=%s | stock=%d | limit=%d", eventID, stock, limit)

	if err := s.redisRepo.InitializeEvent(ctx, eventID, stock, limit); err != nil {
		return fmt.Errorf("failed to initialize event %s in redis: %w", eventID, err)
	}

	return nil
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, qty int) error {
	log.Printf("[SERVICE][INFO] Processing purchase | event_id=%s | user_id=%s | req_id=%s | qty=%d", eventID, userID, reqID, qty)

	if err := s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, qty); err != nil {
		return fmt.Errorf("failed to process purchase for request %s in redis: %w", reqID, err)
	}

	event := models.OrderEvent{
		EventID:   eventID,
		UserID:    userID,
		RequestID: reqID,
		Quantity:  qty,
		Status:    "Processing",
		Timestamp: time.Now(),
	}

	if err := s.producer.PublishOrderEvent(ctx, event); err != nil {
		log.Printf("[SERVICE][ERROR] Kafka publish failed, initiating rollback | event_id=%s | user_id=%s | req_id=%s | qty=%d | err=%v", eventID, userID, reqID, qty, err)

		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rollbackCancel()

		if rbErr := s.redisRepo.RollbackPurchase(rollbackCtx, eventID, userID, reqID, qty); rbErr != nil {
			log.Printf("[SERVICE][CRITICAL] Fatal dual-write! Failed to rollback Redis | event_id=%s | user_id=%s | req_id=%s | qty=%d | rollback_err=%v", eventID, userID, reqID, qty, rbErr)
		}

		return fmt.Errorf("failed to publish event to kafka for request %s: %w", reqID, err)
	}

	return nil
}
