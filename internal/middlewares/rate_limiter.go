package middlewares

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/utils"
	"Ticketing-System/scripts"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
)

const (
	luaSuccess       = 1
	luaInvalidInput  = -1
	luaLimitExceeded = -2
)

type RateLimiter struct {
	rdb           *redis.Client
	cb            *gobreaker.CircuitBreaker
	bannedIPCache *expirable.LRU[string, struct{}]
	scriptSHA     string
	scriptBody    string
	cfg           config.RateLimiterConfig
	log           *slog.Logger
}

func NewRateLimiter(ctx context.Context, rdb *redis.Client, cfg config.RateLimiterConfig) (*RateLimiter, error) {
	logger := infrastructure.GetLogger("MIDDLEWARE_RATE_LIMIT")

	sha, err := rdb.ScriptLoad(ctx, scripts.RateLimitScript).Result()
	if err != nil {
		return nil, fmt.Errorf("load rate_limit script: %w", err)
	}

	logger.Info(
		"Lua script loaded successfully",
		"sha", sha,
	)

	cb := infrastructure.NewCircuitBreaker(logger, "RateLimit_CB", cfg.CBConfig)
	cache := expirable.NewLRU[string, struct{}](cfg.BannedMaxSize, nil, cfg.BannedTTL)

	return &RateLimiter{
		rdb:           rdb,
		cb:            cb,
		bannedIPCache: cache,
		scriptSHA:     sha,
		scriptBody:    scripts.RateLimitScript,
		cfg:           cfg,
		log:           logger,
	}, nil
}

func (rl *RateLimiter) Limit(c *gin.Context) {
	clientIP := c.ClientIP()

	if _, isBanned := rl.bannedIPCache.Get(clientIP); isBanned {
		infrastructure.RateLimitRejections.Inc()

		rl.log.Warn(
			"Rate limit cache validation failed",
			"client_ip", clientIP,
		)

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"status": "fail",
			"error":  "Too many requests, please try again later",
		})

		return
	}

	keys := []string{fmt.Sprintf("ticket:rate_limit:bucket:%s", clientIP)}
	args := []interface{}{rl.cfg.Capacity, rl.cfg.Rate, 1}

	ctx, cancel := context.WithTimeout(c.Request.Context(), rl.cfg.Timeout)
	defer cancel()

	rawResult, err := rl.cb.Execute(func() (interface{}, error) {
		return utils.EvalShaWithFallback(ctx, rl.rdb, rl.scriptSHA, rl.scriptBody, keys, args...).Result()
	})
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

	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.cfg.Capacity))
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
		infrastructure.RateLimitRejections.Inc()
		rl.bannedIPCache.Add(clientIP, struct{}{})

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
