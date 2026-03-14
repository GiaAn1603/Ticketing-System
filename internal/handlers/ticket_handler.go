package handlers

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/services"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	service *services.TicketService
	log     *slog.Logger
}

func NewTicketHandler(service *services.TicketService) *TicketHandler {
	logger := infrastructure.GetLogger("HANDLER")

	return &TicketHandler{
		service: service,
		log:     logger,
	}
}

func (h *TicketHandler) InitTicket(c *gin.Context) {
	var req models.InitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn(
			"InitRequest payload validation failed",
			infrastructure.KeyAction, "init_ticket",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			infrastructure.KeyError, err.Error(),
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
			infrastructure.KeyAction, "init_ticket",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			"event_id", req.EventID,
			"stock", req.Stock,
			"max_limit", req.MaxLimit,
			infrastructure.KeyError, err.Error(),
		)

		if strings.Contains(err.Error(), "failed to set stock in redis") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error":  "Failed to set event stock",
			})
			return
		}

		if strings.Contains(err.Error(), "already exists") {
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
		infrastructure.KeyAction, "init_ticket",
		infrastructure.KeyStatus, infrastructure.StatusSuccess,
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
	var req models.BuyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn(
			"BuyRequest payload validation failed",
			infrastructure.KeyAction, "buy_ticket",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			infrastructure.KeyError, err.Error(),
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
			infrastructure.KeyAction, "buy_ticket",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			"event_id", req.EventID,
			"user_id", req.UserID,
			"quantity", req.Quantity,
			infrastructure.KeyError, "missing_request_id_header",
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
			infrastructure.KeyAction, "buy_ticket",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			"event_id", req.EventID,
			"user_id", req.UserID,
			"request_id", reqID,
			"quantity", req.Quantity,
			infrastructure.KeyError, err.Error(),
		)

		switch {
		case errors.Is(err, models.ErrAlreadyProcessed):
			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"message": "Request already processed",
			})
		case errors.Is(err, models.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "fail",
				"error":  "Invalid input parameters",
			})
		case errors.Is(err, models.ErrLimitExceeded):
			c.JSON(http.StatusConflict, gin.H{
				"status": "fail",
				"error":  "Purchase limit exceeded",
			})
		case errors.Is(err, models.ErrOutOfStock):
			c.JSON(http.StatusConflict, gin.H{
				"status": "fail",
				"error":  "Sold out",
			})
		case errors.Is(err, models.ErrEventNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status": "fail",
				"error":  "Event not found",
			})
		case errors.Is(err, models.ErrInternal):
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

	h.log.Info(
		"Purchase completed successfully",
		infrastructure.KeyAction, "buy_ticket",
		infrastructure.KeyStatus, infrastructure.StatusSuccess,
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
