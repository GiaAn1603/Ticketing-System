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
