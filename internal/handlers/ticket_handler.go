package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"Ticketing-System/internal/handlers/requests"
	"Ticketing-System/internal/infrastructure/observability"
	"Ticketing-System/pkg/apperrors"
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
	logger := observability.GetLogger("HANDLER")

	return &TicketHandler{
		service: service,
		reqPool: sync.Pool{New: func() interface{} { return new(requests.BuyRequest) }},
		log:     logger,
	}
}

func (h *TicketHandler) InitTicket(c *gin.Context) {
	var req requests.InitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn(
			"InitRequest payload validation failed",
			observability.KeyAction, "init_ticket",
			observability.KeyStatus, observability.StatusFailed,
			observability.KeyError, err.Error(),
		)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Invalid request payload",
		})

		return
	}

	if err := h.service.InitializeEvent(c.Request.Context(), req.EventID, req.Stock, req.MaxLimit); err != nil {
		h.log.Error(
			"Event initialization failed",
			observability.KeyAction, "init_ticket",
			observability.KeyStatus, observability.StatusFailed,
			"event_id", req.EventID,
			"stock", req.Stock,
			"max_limit", req.MaxLimit,
			observability.KeyError, err.Error(),
		)

		if strings.Contains(err.Error(), "event already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"status": "fail",
				"error":  "Event already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})

		return
	}

	h.log.Info(
		"Event initialized successfully",
		"event_id", req.EventID,
		"stock", req.Stock,
		"max_limit", req.MaxLimit,
	)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"event_id": req.EventID,
	})
}

func (h *TicketHandler) BuyTicket(c *gin.Context) {
	req := h.reqPool.Get().(*requests.BuyRequest)
	*req = requests.BuyRequest{}
	defer h.reqPool.Put(req)

	if err := c.ShouldBindJSON(req); err != nil {
		h.log.Warn(
			"BuyRequest payload validation failed",
			observability.KeyAction, "buy_ticket",
			observability.KeyStatus, observability.StatusFailed,
			observability.KeyError, err.Error(),
		)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Invalid request payload",
		})

		return
	}

	reqID := c.GetHeader("X-Request-ID")
	if reqID == "" {
		h.log.Warn(
			"X-Request-ID header missing",
			observability.KeyAction, "buy_ticket",
			observability.KeyStatus, observability.StatusFailed,
			"event_id", req.EventID,
			"user_id", req.UserID,
			"quantity", req.Quantity,
			observability.KeyError, "missing_request_id_header",
		)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Missing X-Request-ID header",
		})

		return
	}

	if err := h.service.ProcessPurchase(c.Request.Context(), req.EventID, req.UserID, reqID, req.Quantity); err != nil {
		h.log.Warn(
			"Purchase failed",
			observability.KeyAction, "buy_ticket",
			observability.KeyStatus, observability.StatusFailed,
			"event_id", req.EventID,
			"user_id", req.UserID,
			"request_id", reqID,
			"quantity", req.Quantity,
			observability.KeyError, err.Error(),
		)

		switch {
		case errors.Is(err, apperrors.ErrAlreadyProcessed):
			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"message": "Request already processed",
			})
		case errors.Is(err, apperrors.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "fail",
				"error":  "Invalid input parameters",
			})
		case errors.Is(err, apperrors.ErrEventNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status": "fail",
				"error":  "Event not found",
			})
		case errors.Is(err, apperrors.ErrOutOfStock):
			observability.TicketSales.WithLabelValues(req.EventID, "sold_out").Inc()

			c.JSON(http.StatusConflict, gin.H{
				"status": "fail",
				"error":  "Sold out",
			})
		case errors.Is(err, apperrors.ErrPurchaseLimitExceeded):
			c.JSON(http.StatusConflict, gin.H{
				"status": "fail",
				"error":  "Purchase limit exceeded",
			})
		case errors.Is(err, apperrors.ErrInternal):
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  "Internal database error",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  "Internal server error",
			})
		}

		return
	}

	observability.TicketSales.WithLabelValues(req.EventID, "success").Inc()

	h.log.Info(
		"Purchase completed successfully",
		"event_id", req.EventID,
		"user_id", req.UserID,
		"request_id", reqID,
		"quantity", req.Quantity,
	)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ticket purchased successfully",
	})
}
