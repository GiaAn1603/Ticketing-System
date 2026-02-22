package services

import (
	"Ticketing-System/internal/repositories"
	"context"
	"log"
)

type TicketService struct {
	redisRepo *repositories.RedisRepo
}

func NewTicketService(redisRepo *repositories.RedisRepo) *TicketService {
	return &TicketService{
		redisRepo: redisRepo,
	}
}

func (s *TicketService) InitializeEvent(ctx context.Context, eventID string, stock int) error {
	log.Printf("[SERVICE][INFO] Processing init event | event_id=%s | stock=%d", eventID, stock)
	return s.redisRepo.InitializeEvent(ctx, eventID, stock)
}

func (s *TicketService) ProcessPurchase(ctx context.Context, eventID, userID, reqID string, qty, limit int) error {
	log.Printf("[SERVICE][INFO] Processing purchase | req_id=%s | event_id=%s | user_id=%s", reqID, eventID, userID)
	return s.redisRepo.PurchaseTicket(ctx, eventID, userID, reqID, qty, limit)
}
