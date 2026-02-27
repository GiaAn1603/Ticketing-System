package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/handlers"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/middlewares"
	"Ticketing-System/internal/repositories"
	"Ticketing-System/internal/services"
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rdb, err := infrastructure.ConnectRedis(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to connect Redis | err=%v", err)
	}
	defer rdb.Close()

	rateLimiter, err := middlewares.NewRateLimiter(ctx, rdb, 10, 5, 500*time.Millisecond)
	if err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to init RateLimiter | err=%v", err)
	}

	redisRepo, err := repositories.NewRedisRepo(ctx, rdb)
	if err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to init RedisRepo | err=%v", err)
	}

	ticketService := services.NewTicketService(redisRepo)
	ticketHandler := handlers.NewTicketHandler(ticketService)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "success",
		})
	})

	r.POST("/init-ticket", rateLimiter.Limit, ticketHandler.InitTicket)
	r.POST("/buy-ticket", rateLimiter.Limit, ticketHandler.BuyTicket)

	log.Printf("[MAIN][INFO] Server started | url=http://localhost%s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to start server | err=%v", err)
	}
}
