package handlers

import (
	"Ticketing-System/internal/models"
	"Ticketing-System/internal/services"
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
		log.Printf("[HANDLER][ERROR] Init event failed | event_id=%s | err=%v", req.EventID, err)

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

	log.Printf("[HANDLER][INFO] Init event successful | event_id=%s", req.EventID)

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"event_id": req.EventID,
	})
}
