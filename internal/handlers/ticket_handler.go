package handlers

import "Ticketing-System/internal/services"

type TicketHandler struct {
	service *services.TicketService
}

func NewTicketHandler(service *services.TicketService) *TicketHandler {
	return &TicketHandler{
		service: service,
	}
}
