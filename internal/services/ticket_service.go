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
		return fmt.Errorf("service failed to initialize event %s: %w", eventID, err)
	}

	return nil
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, qty int) error {
	log.Printf("[SERVICE][INFO] Processing purchase | event_id=%s | user_id=%s | req_id=%s | qty=%d", eventID, userID, reqID, qty)

	if err := s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, qty); err != nil {
		return fmt.Errorf("service failed to process purchase for user %s: %w", userID, err)
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
		log.Printf("[SERVICE][ERROR] Failed to publish order event | event_id=%s | user_id=%s | req_id=%s | qty=%d", eventID, userID, reqID, qty)
		return fmt.Errorf("service failed to publish event for user %s: %w", userID, err)
	}

	return nil
}
