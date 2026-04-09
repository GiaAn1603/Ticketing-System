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
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func run() error {
	infrastructure.InitLogger()
	logger := infrastructure.GetLogger("MAIN")
	closer := infrastructure.NewAppCloser(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.ServerStartupTimeout)
	defer startupCancel()

	tp, err := infrastructure.InitTracer(startupCtx, "ticket-api", cfg.OtelExporterEndpoint, cfg.ToTracerConfig())
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	closer.Add(func() error { return tp.Shutdown(context.Background()) })

	rdb, err := infrastructure.ConnectRedis(startupCtx, cfg.ToRedisConfig())
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	closer.Add(rdb.Close)

	kafkaProducer, err := events.NewKafkaProducer(startupCtx, cfg.ToProducerConfig())
	if err != nil {
		return fmt.Errorf("init kafka producer: %w", err)
	}
	closer.Add(kafkaProducer.Close)

	rateLimiter, err := middlewares.NewRateLimiter(startupCtx, rdb, cfg.ToRateLimiterConfig())
	if err != nil {
		return fmt.Errorf("init rate limiter: %w", err)
	}

	redisRepo, err := repositories.NewRedisRepo(startupCtx, rdb, cfg.ToRedisRepoConfig())
	if err != nil {
		return fmt.Errorf("init redis repo: %w", err)
	}

	ticketService := services.NewTicketService(redisRepo, kafkaProducer, cfg.ToTicketServiceConfig())
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
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
			"status":  "success",
		})
	})

	r.POST("/api/init-ticket", rateLimiter.Limit, otelgin.Middleware("ticket-api"), ticketHandler.InitTicket)
	r.POST("/api/buy-ticket", rateLimiter.Limit, otelgin.Middleware("ticket-api"), ticketHandler.BuyTicket)

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

	closer.CloseAll()

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
