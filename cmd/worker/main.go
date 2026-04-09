package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/repositories"
	"Ticketing-System/internal/services"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func run() error {
	infrastructure.InitLogger()
	logger := infrastructure.GetLogger("WORKER")
	closer := infrastructure.NewAppCloser(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.ServerStartupTimeout)
	defer startupCancel()

	tp, err := infrastructure.InitTracer(startupCtx, "ticket-worker", cfg.OtelExporterEndpoint, cfg.ToTracerConfig())
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	closer.Add(func() error { return tp.Shutdown(context.Background()) })

	pgDB, err := infrastructure.ConnectPostgres(startupCtx, cfg.ToDBConfig())
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	closer.Add(pgDB.Close)

	pgRepo := repositories.NewPostgresRepo(pgDB)
	orderService := services.NewOrderService(pgRepo, cfg.ToOrderServiceConfig())

	kafkaConsumer, err := events.NewKafkaConsumer(startupCtx, orderService, cfg.ToConsumerConfig())
	if err != nil {
		return fmt.Errorf("init kafka consumer: %w", err)
	}
	closer.Add(kafkaConsumer.Close)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	workerErrChan := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("Worker started")

		if err := kafkaConsumer.ConsumeOrderEvent(workerCtx); err != nil {
			logger.Error(
				"Consumer loop crashed",
				infrastructure.KeyAction, "run_consumer_loop",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)

			workerErrChan <- err
		}
	}()

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	select {
	case err := <-workerErrChan:
		return fmt.Errorf("run worker: %w", err)
	case <-signalCtx.Done():
		logger.Info(
			"Shutdown signal received",
			"signal", "SIGINT/SIGTERM",
		)
	}

	logger.Info("Worker shutdown started")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer shutdownCancel()

	workerCancel()

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case <-doneChan:
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown worker: %w", shutdownCtx.Err())
	}

	closer.CloseAll()

	logger.Info("Worker exited")

	return nil
}

func main() {
	if err := run(); err != nil {
		slog.Error(
			"Worker startup failed",
			"layer", "WORKER",
			infrastructure.KeyAction, "run_worker",
			infrastructure.KeyStatus, infrastructure.StatusFailed,
			infrastructure.KeyError, err.Error(),
		)

		os.Exit(1)
	}
}
