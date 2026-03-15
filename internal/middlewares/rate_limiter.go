package middlewares

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/utils"
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	luaSuccess       = 1
	luaInvalidInput  = -1
	luaLimitExceeded = -2
)

type RateLimiter struct {
	rdb        *redis.Client
	capacity   int
	rate       int
	timeout    time.Duration
	scriptSHA  string
	scriptBody string
	log        *slog.Logger
}

func NewRateLimiter(ctx context.Context, rdb *redis.Client, capacity, rate int, timeout time.Duration) (*RateLimiter, error) {
	logger := infrastructure.GetLogger("MIDDLEWARE_RATE_LIMIT")

	sha, err := rdb.ScriptLoad(ctx, scripts.RateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load rate_limit script: %w", err)
	}

	logger.Info(
		"Lua script loaded successfully",
		"sha", sha,
	)

	return &RateLimiter{
		rdb:        rdb,
		capacity:   capacity,
		rate:       rate,
		timeout:    timeout,
		scriptSHA:  sha,
		scriptBody: scripts.RateLimitScript,
		log:        logger,
	}, nil
}

func (rl *RateLimiter) Limit(c *gin.Context) {
	clientIP := c.ClientIP()
	keys := []string{fmt.Sprintf("ticket:rate_limit:bucket:%s", clientIP)}
	args := []interface{}{rl.capacity, rl.rate, 1}

	ctx, cancel := context.WithTimeout(c.Request.Context(), rl.timeout)
	defer cancel()

	rawResult, err := utils.EvalShaWithFallback(ctx, rl.rdb, rl.scriptSHA, rl.scriptBody, keys, args...).Result()
	if err != nil {
		rl.log.Error(
			"Lua script execution failed",
			"client_ip", clientIP,
			infrastructure.KeyError, err.Error(),
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})

		return
	}

	results, isArray := rawResult.([]interface{})
	if !isArray || len(results) < 2 {
		rl.log.Error(
			"Lua response format validation failed",
			"client_ip", clientIP,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})

		return
	}

	statusCode, isValidStatus := results[0].(int64)
	remainingTokens, isValidTokens := results[1].(int64)

	if !isValidStatus || !isValidTokens {
		rl.log.Error(
			"Lua response types validation failed",
			"client_ip", clientIP,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})

		return
	}

	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.capacity))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remainingTokens))

	switch statusCode {
	case luaSuccess:
		c.Next()
	case luaInvalidInput:
		rl.log.Error(
			"Lua parameters validation failed",
			"client_ip", clientIP,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
	case luaLimitExceeded:
		rl.log.Warn(
			"Rate limit exceeded",
			"client_ip", clientIP,
		)

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"status": "fail",
			"error":  "Too many requests, please try again later",
		})
	default:
		rl.log.Error(
			"Lua result recognized failed",
			"client_ip", clientIP,
			"status_code", statusCode,
		)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
	}
}
