package responses

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/pkg/apperrors"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RespondError(c *gin.Context, err error, action, reqID string, log *slog.Logger) {
	status := "fail"
	code := http.StatusInternalServerError
	var message string

	switch {
	case errors.Is(err, apperrors.AlreadyProcessed):
		code = http.StatusOK
		status = "success"
		message = "Request already processed"
	case errors.Is(err, apperrors.InvalidInput):
		code = http.StatusBadRequest
		message = "Invalid input parameters"
	case errors.Is(err, apperrors.LimitExceeded):
		code = http.StatusConflict
		message = "Purchase limit exceeded"
	case errors.Is(err, apperrors.OutOfStock):
		code = http.StatusConflict
		message = "Sold out"
	case errors.Is(err, apperrors.EventNotFound):
		code = http.StatusNotFound
		message = "Event not found"
	case errors.Is(err, apperrors.EventAlreadyExists):
		code = http.StatusConflict
		message = "Event already exists"
	case errors.Is(err, apperrors.MissingRequestID):
		code = http.StatusBadRequest
		message = "Missing X-Request-ID header"
	case errors.Is(err, apperrors.Internal):
		status = "error"
		message = "Internal database error"
	default:
		status = "error"
		message = err.Error()
	}

	logArgs := []interface{}{
		infrastructure.KeyAction, action,
		infrastructure.KeyError, err.Error(),
	}

	if reqID != "" {
		logArgs = append(logArgs, "request_id", reqID)
	}

	log.Warn("Request failed", logArgs...)

	c.JSON(code, gin.H{
		"status": status,
		"error":  message,
	})
}

func AbortWithError(c *gin.Context, err error, action string, log *slog.Logger) {
	status := "fail"
	code := http.StatusInternalServerError
	var message string

	switch {
	case errors.Is(err, apperrors.LimitExceeded):
		code = http.StatusTooManyRequests
		message = "Too many requests, please try again later"
	case errors.Is(err, apperrors.Internal):
		status = "error"
		message = "Internal server error"
	default:
		status = "error"
		message = err.Error()
	}

	log.Warn("Request aborted",
		infrastructure.KeyAction, action,
		"client_ip", c.ClientIP(),
		infrastructure.KeyError, err.Error(),
	)

	c.AbortWithStatusJSON(code, gin.H{
		"status": status,
		"error":  message,
	})
}

func RespondSuccess(c *gin.Context, payload gin.H) {
	c.JSON(http.StatusOK, payload)
}
