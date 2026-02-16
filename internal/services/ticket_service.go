package services

import "Ticketing-System/internal/repositories"

type TicketService struct {
	redisRepo *repositories.RedisRepo
}

func NewTicketService(redisRepo *repositories.RedisRepo) *TicketService {
	return &TicketService{
		redisRepo: redisRepo,
	}
}
