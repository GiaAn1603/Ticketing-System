package handlers

import (
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/services"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	service *services.TicketService
}

func NewTicketHandler(service *services.TicketService) *TicketHandler {
	return &TicketHandler{
		service: service,
	}
}

func (h *TicketHandler) InitTicket(c *gin.Context) {
	var req models.InitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[HANDLER][WARN] Invalid InitRequest payload | err=%v", err)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Invalid request payload",
		})
		return
	}

	if err := h.service.InitializeEvent(c.Request.Context(), req.EventID, req.Stock); err != nil {
		log.Printf("[HANDLER][ERROR] Init event failed | event_id=%s | stock=%d | err=%v", req.EventID, req.Stock, err)

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

	log.Printf("[HANDLER][INFO] Init event successful | event_id=%s | stock=%d", req.EventID, req.Stock)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"event_id": req.EventID,
	})
}

func (h *TicketHandler) BuyTicket(c *gin.Context) {
	var req models.BuyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[HANDLER][WARN] Invalid BuyRequest payload | err=%v", err)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Invalid request payload",
		})
		return
	}

	reqID := c.GetHeader("X-Request-ID")
	if reqID == "" {
		log.Printf("[HANDLER][WARN] Missing X-Request-ID | event_id=%s | user_id=%s | qty=%d | limit=%d", req.EventID, req.UserID, req.Quantity, req.MaxLimit)

		c.JSON(http.StatusBadRequest, gin.H{
			"status": "fail",
			"error":  "Missing X-Request-ID header",
		})
		return
	}

	if err := h.service.ProcessPurchase(c.Request.Context(), req.EventID, req.UserID, reqID, req.Quantity, req.MaxLimit); err != nil {
		log.Printf("[HANDLER][WARN] Purchase failed | event_id=%s | user_id=%s | req_id=%s | qty=%d | limit=%d | err=%v", req.EventID, req.UserID, reqID, req.Quantity, req.MaxLimit, err)

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

	log.Printf("[HANDLER][INFO] Purchase successful | event_id=%s | user_id=%s | req_id=%s | qty=%d | limit=%d", req.EventID, req.UserID, reqID, req.Quantity, req.MaxLimit)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ticket purchased successfully",
	})
}
