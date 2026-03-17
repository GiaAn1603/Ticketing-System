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
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func run() error {
	infrastructure.InitLogger()
	logger := infrastructure.GetLogger("MAIN")

	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.ServerStartupTimeout)
	defer startupCancel()

	rdb, err := infrastructure.ConnectRedis(startupCtx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		logger.Info("Redis connection closing")

		if err := rdb.Close(); err != nil {
			logger.Warn(
				"Redis close failed",
				infrastructure.KeyAction, "shutdown",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)
		}
	}()

	kafkaBrokers := []string{cfg.KafkaAddr}
	kafkaProducer, err := events.NewKafkaProducer(
		startupCtx,
		kafkaBrokers,
		cfg.KafkaTopicOrders,
		cfg.KafkaNumPartitions,
		cfg.KafkaReplicationFactor,
		cfg.KafkaProducerBatchSize,
		cfg.KafkaProducerBatchTimeout,
		cfg.KafkaTimeout,
	)
	if err != nil {
		return fmt.Errorf("init kafka producer: %w", err)
	}
	defer func() {
		logger.Info("Kafka connection closing")

		if err := kafkaProducer.Close(); err != nil {
			logger.Warn(
				"Kafka close failed",
				infrastructure.KeyAction, "shutdown",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)
		}
	}()

	rateLimiter, err := middlewares.NewRateLimiter(
		startupCtx,
		rdb,
		cfg.RateLimitCapacity,
		cfg.RateLimitRate,
		cfg.CBMaxRequests,
		cfg.RateLimitTimeout,
		cfg.CBInterval,
		cfg.CBTimeout,
	)
	if err != nil {
		return fmt.Errorf("init rate limiter: %w", err)
	}

	redisRepo, err := repositories.NewRedisRepo(
		startupCtx,
		rdb,
		cfg.HistoryTTLSeconds,
		cfg.CBMaxRequests,
		cfg.CBInterval,
		cfg.CBTimeout,
	)
	if err != nil {
		return fmt.Errorf("init redis repo: %w", err)
	}

	ticketService := services.NewTicketService(redisRepo, kafkaProducer, cfg.RedisTimeout)
	ticketHandler := handlers.NewTicketHandler(ticketService)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	err = r.SetTrustedProxies([]string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1",
	})
	if err != nil {
		return fmt.Errorf("set trusted proxies: %w", err)
	}

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

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
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
	}

	srvErrChan := make(chan error, 1)

	go func() {
		logger.Info(
			"Server started",
			"url", fmt.Sprintf("http://localhost%s", cfg.ServerPort),
		)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(
				"HTTP server crashed",
				infrastructure.KeyAction, "run_http_server",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)

			srvErrChan <- err
		}
	}()

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	select {
	case err := <-srvErrChan:
		return fmt.Errorf("run server: %w", err)
	case <-signalCtx.Done():
		logger.Info(
			"Shutdown signal received",
			"signal", "SIGINT/SIGTERM",
		)
	}

	logger.Info("Server shutdown started")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	logger.Info("Server exited")

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error(
			"Application startup failed",
			"layer", "MAIN",
			infrastructure.KeyAction, "run_application",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			infrastructure.KeyError, err.Error(),
		)

		os.Exit(1)
	}
}
