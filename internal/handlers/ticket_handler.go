package handlers

import (
	"Ticketing-System/internal/handlers/requests"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/pkg/apperrors"
	"Ticketing-System/internal/pkg/responses"
	"context"
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
)

type TicketService interface {
	InitializeEvent(ctx context.Context, eventID string, stock, maxLimit int) error
	ProcessPurchase(ctx context.Context, eventID, userID, reqID string, quantity int) error
}

type TicketHandler struct {
	service TicketService
	reqPool sync.Pool
	log     *slog.Logger
}

func NewTicketHandler(service TicketService) *TicketHandler {
	logger := infrastructure.GetLogger("HANDLER")

	return &TicketHandler{
		service: service,
		reqPool: sync.Pool{New: func() interface{} { return new(requests.BuyRequest) }},
		log:     logger,
	}
}

func (h *TicketHandler) InitTicket(c *gin.Context) {
	var req requests.InitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		responses.RespondError(c, err, "init_ticket", "", h.log)
		return
	}

	if err := h.service.InitializeEvent(c.Request.Context(), req.EventID, req.Stock, req.MaxLimit); err != nil {
		responses.RespondError(c, err, "init_ticket", "", h.log)
		return
	}

	h.log.Info(
		"Event initialized successfully",
		"event_id", req.EventID,
		"stock", req.Stock,
		"max_limit", req.MaxLimit,
	)

	responses.RespondSuccess(c, gin.H{
		"status":   "success",
		"event_id": req.EventID,
	})
}

func (h *TicketHandler) BuyTicket(c *gin.Context) {
	req := h.reqPool.Get().(*requests.BuyRequest)
	*req = requests.BuyRequest{}
	defer h.reqPool.Put(req)

	if err := c.ShouldBindJSON(req); err != nil {
		responses.RespondError(c, err, "buy_ticket", "", h.log)
		return
	}

	reqID := c.GetHeader("X-Request-ID")
	if reqID == "" {
		responses.RespondError(c, apperrors.MissingRequestID, "buy_ticket", "", h.log)
		return
	}

	if err := h.service.ProcessPurchase(c.Request.Context(), req.EventID, req.UserID, reqID, req.Quantity); err != nil {
		responses.RespondError(c, err, "buy_ticket", reqID, h.log)
		return
	}

	infrastructure.TicketSales.WithLabelValues(req.EventID, "success").Inc()

	h.log.Info(
		"Purchase completed successfully",
		"event_id", req.EventID,
		"user_id", req.UserID,
		"request_id", reqID,
		"quantity", req.Quantity,
	)

	responses.RespondSuccess(c, gin.H{
		"status":  "success",
		"message": "Ticket purchased successfully",
	})
}
