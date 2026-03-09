package middlewares

import (
	"Ticketing-System/internal/utils"
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log"
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
}

func NewRateLimiter(ctx context.Context, rdb *redis.Client, capacity, rate int, timeout time.Duration) (*RateLimiter, error) {
	sha, err := rdb.ScriptLoad(ctx, scripts.RateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate_limit lua script: %w", err)
	}

	log.Printf("[MIDDLEWARE][INFO] Lua script loaded successfully | sha=%s", sha)

	return &RateLimiter{
		rdb:        rdb,
		capacity:   capacity,
		rate:       rate,
		timeout:    timeout,
		scriptSHA:  sha,
		scriptBody: scripts.RateLimitScript,
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
		log.Printf("[MIDDLEWARE][ERROR] Lua script execution failed | client_ip=%s | err=%v", clientIP, err)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
		return
	}

	results, isArray := rawResult.([]interface{})
	if !isArray || len(results) < 2 {
		log.Printf("[MIDDLEWARE][ERROR] Invalid Lua response format | client_ip=%s", clientIP)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
		return
	}

	statusCode, isValidStatus := results[0].(int64)
	remainingTokens, isValidTokens := results[1].(int64)

	if !isValidStatus || !isValidTokens {
		log.Printf("[MIDDLEWARE][ERROR] Invalid Lua response types | client_ip=%s", clientIP)

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
		log.Printf("[MIDDLEWARE][ERROR] Invalid Lua parameters | client_ip=%s", clientIP)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
	case luaLimitExceeded:
		log.Printf("[MIDDLEWARE][WARN] Rate limit exceeded | client_ip=%s", clientIP)

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"status": "fail",
			"error":  "Too many requests, please try again later",
		})
	default:
		log.Printf("[MIDDLEWARE][ERROR] Unknown Lua result | client_ip=%s | code=%d", clientIP, statusCode)

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Internal server error",
		})
	}
}
