package responses

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"Ticketing-System/pkg/apperrors"
)

func getHTTPError(err error) (int, string, string) {
	switch {
	case errors.Is(err, apperrors.ErrAlreadyProcessed):
		return http.StatusOK, "success", "Request already processed"
	case errors.Is(err, apperrors.ErrInvalidInput):
		return http.StatusBadRequest, "fail", "Invalid input parameters"
	case errors.Is(err, apperrors.ErrEventNotFound):
		return http.StatusNotFound, "fail", "Event not found"
	case errors.Is(err, apperrors.ErrOutOfStock):
		return http.StatusConflict, "fail", "Sold out"
	case errors.Is(err, apperrors.ErrPurchaseLimitExceeded):
		return http.StatusConflict, "fail", "Purchase limit exceeded"
	case errors.Is(err, apperrors.ErrRateLimitExceeded):
		return http.StatusTooManyRequests, "fail", "Too many requests, please try again later"
	case errors.Is(err, apperrors.ErrEventAlreadyExists):
		return http.StatusConflict, "fail", "Event already exists"
	case errors.Is(err, apperrors.ErrMissingRequestID):
		return http.StatusBadRequest, "fail", "Missing X-Request-ID header"
	case errors.Is(err, apperrors.ErrInternal):
		return http.StatusInternalServerError, "error", "Internal database error"
	default:
		return http.StatusInternalServerError, "error", "Internal server error"
	}
}

func Abort(c *gin.Context, err error) {
	httpCode, status, msg := getHTTPError(err)

	c.AbortWithStatusJSON(httpCode, gin.H{
		"status": status,
		"error":  msg,
	})
}

func Error(c *gin.Context, err error) {
	httpCode, status, msg := getHTTPError(err)

	if status == "success" {
		c.JSON(httpCode, gin.H{
			"status":  status,
			"message": msg,
		})
		return
	}

	c.JSON(httpCode, gin.H{
		"status": status,
		"error":  msg,
	})
}

func Success(c *gin.Context, payload gin.H) {
	payload["status"] = "success"
	c.JSON(http.StatusOK, payload)
}
