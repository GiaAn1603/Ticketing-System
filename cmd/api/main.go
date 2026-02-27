package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/handlers"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/middlewares"
	"Ticketing-System/internal/repositories"
	"Ticketing-System/internal/services"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startupCancel()

	rdb, err := infrastructure.ConnectRedis(startupCtx, cfg.RedisAddr)
	if err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to connect Redis | err=%v", err)
	}
	defer func() {
		log.Println("[MAIN][INFO] Closing Redis connection")
		rdb.Close()
	}()

	rateLimiter, err := middlewares.NewRateLimiter(startupCtx, rdb, 10, 5, 500*time.Millisecond)
	if err != nil {
		log.Fatalf("[MAIN][FATAL] Failed to init RateLimiter | err=%v", err)
	}

	redisRepo, err := repositories.NewRedisRepo(startupCtx, rdb)
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

	srv := &http.Server{
		Addr:    cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("[MAIN][INFO] Server started | url=http://localhost%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[MAIN][FATAL] Failed to start server | err=%v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("[MAIN][INFO] Shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[MAIN][FATAL] Server forced to shutdown | err=%v", err)
	}

	log.Println("[MAIN][INFO] Server exited")
}
