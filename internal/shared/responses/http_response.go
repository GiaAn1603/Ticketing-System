package responses

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/shared/apperrors"
)

var errorCodeToHTTP = map[apperrors.ErrorCode]int{
	apperrors.CodeAlreadyProcessed:   http.StatusOK,
	apperrors.CodeInvalidInput:       http.StatusBadRequest,
	apperrors.CodePurchaseLimit:      http.StatusConflict,
	apperrors.CodeOutOfStock:         http.StatusConflict,
	apperrors.CodeEventNotFound:      http.StatusNotFound,
	apperrors.CodeRateLimit:          http.StatusTooManyRequests,
	apperrors.CodeEventAlreadyExists: http.StatusConflict,
	apperrors.CodeMissingRequestID:   http.StatusBadRequest,
	apperrors.CodeInternal:           http.StatusInternalServerError,
}

func Error(c *gin.Context, err error, action, reqID string, log *slog.Logger) {
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		appErr = apperrors.ErrInternal
	}

	httpCode, exists := errorCodeToHTTP[appErr.Code]
	if !exists {
		httpCode = http.StatusInternalServerError
	}

	logArgs := []interface{}{
		infrastructure.KeyAction, action,
		infrastructure.KeyError, err.Error(),
	}

	if reqID != "" {
		logArgs = append(logArgs, "request_id", reqID)
	}

	log.Warn("Request failed", logArgs...)

	c.JSON(httpCode, gin.H{
		"status": appErr.Status,
		"error":  appErr.Message,
	})
}

func Abort(c *gin.Context, err error, action string, log *slog.Logger) {
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		appErr = apperrors.ErrInternal
	}

	httpCode, exists := errorCodeToHTTP[appErr.Code]
	if !exists {
		httpCode = http.StatusInternalServerError
	}

	log.Warn("Request aborted",
		infrastructure.KeyAction, action,
		"client_ip", c.ClientIP(),
		infrastructure.KeyError, err.Error(),
	)

	c.AbortWithStatusJSON(httpCode, gin.H{
		"status": appErr.Status,
		"error":  appErr.Message,
	})
}

func Success(c *gin.Context, payload gin.H) {
	c.JSON(http.StatusOK, payload)
}
