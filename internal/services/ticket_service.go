package services

import (
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/repositories"
	"context"
	"log"
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
	return s.redisRepo.InitializeEvent(ctx, eventID, stock, limit)
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, qty int) error {
	log.Printf("[SERVICE][INFO] Processing purchase | event_id=%s | user_id=%s | req_id=%s | qty=%d", eventID, userID, reqID, qty)
	return s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, qty)
}
