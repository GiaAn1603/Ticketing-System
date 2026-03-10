package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/handlers"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/middlewares"
	"Ticketing-System/internal/repositories"
	"Ticketing-System/internal/services"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func run() error {
	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startupCancel()

	rdb, err := infrastructure.ConnectRedis(startupCtx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}
	defer func() {
		log.Println("[MAIN][INFO] Closing Redis connection | action=close_redis")
		if err := rdb.Close(); err != nil {
			log.Printf("[MAIN][WARN] Redis close error | err=%v", err)
		}
	}()

	kafkaBrokers := []string{cfg.KafkaAddr}
	kafkaTopic := "orders"
	kafkaProducer, err := events.NewKafkaProducer(startupCtx, kafkaBrokers, kafkaTopic)
	if err != nil {
		return fmt.Errorf("failed to init kafka producer: %w", err)
	}
	defer func() {
		log.Println("[MAIN][INFO] Closing Kafka connection | action=close_kafka")
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("[MAIN][WARN] Kafka close error | err=%v", err)
		}
	}()

	rateLimiter, err := middlewares.NewRateLimiter(startupCtx, rdb, 10, 5, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to init rate limiter: %w", err)
	}

	redisRepo, err := repositories.NewRedisRepo(startupCtx, rdb)
	if err != nil {
		return fmt.Errorf("failed to init redis repo: %w", err)
	}

	ticketService := services.NewTicketService(redisRepo, kafkaProducer)
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
		Addr:         cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	srvErrChan := make(chan error, 1)

	go func() {
		log.Printf("[MAIN][INFO] Server started | url=http://localhost%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[MAIN][ERROR] HTTP server crashed | err=%v", err)
			srvErrChan <- err
		}
	}()

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	select {
	case err := <-srvErrChan:
		return fmt.Errorf("server stopped unexpectedly: %w", err)
	case <-signalCtx.Done():
		log.Println("[MAIN][INFO] Received shutdown signal | signal=SIGINT/SIGTERM")
	}

	log.Println("[MAIN][INFO] Shutting down server | status=in_progress")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("[MAIN][INFO] Server exited | status=done")

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("[MAIN][FATAL] Application startup failed | err=%v", err)
		os.Exit(1)
	}
}
