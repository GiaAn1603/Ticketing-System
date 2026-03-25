package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/repositories"
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

	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.ServerStartupTimeout)
	defer startupCancel()

	tp, err := infrastructure.InitTracer(
		startupCtx,
		"ticket-worker",
		cfg.OtelExporterEndpoint,
		cfg.OtelBatchMaxQueueSize,
		cfg.OtelBatchMaxExportSize,
		cfg.OtelBatchTimeout,
		cfg.OtelExportTimeout,
		cfg.OtelTraceRatio,
	)
	if err != nil {
		return fmt.Errorf("init tracer: %w", err)
	}
	defer func() {
		logger.Info("Tracer connection closing")

		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Warn(
				"Tracer close failed",
				infrastructure.KeyAction, "shutdown",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)
		}
	}()

	pgDB, err := infrastructure.ConnectPostgres(
		startupCtx,
		cfg.PostgresAddr,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
		cfg.DBMaxOpenConns,
		cfg.DBMaxIdleConns,
		cfg.DBConnMaxLifetime,
	)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer func() {
		logger.Info("Postgres connection closing")

		if err := pgDB.Close(); err != nil {
			logger.Warn(
				"Postgres close failed",
				infrastructure.KeyAction, "shutdown",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)
		}
	}()

	pgRepo := repositories.NewPostgresRepo(pgDB)

	kafkaBrokers := []string{cfg.KafkaAddr}
	kafkaConsumer, err := events.NewKafkaConsumer(
		startupCtx,
		pgRepo,
		kafkaBrokers,
		cfg.KafkaTopicOrders,
		cfg.KafkaGroupID,
		cfg.KafkaNumPartitions,
		cfg.KafkaReplicationFactor,
		cfg.KafkaConsumerMinBytes,
		cfg.KafkaConsumerMaxBytes,
		cfg.KafkaTimeout,
		cfg.DBTimeout,
		cfg.KafkaCommitTimeout,
		cfg.DBRetryBackoffBase,
		cfg.DBRetryBackoffJitter,
	)
	if err != nil {
		return fmt.Errorf("init kafka consumer: %w", err)
	}
	defer func() {
		logger.Info("Kafka connection closing")

		if err := kafkaConsumer.Close(); err != nil {
			logger.Warn(
				"Kafka close failed",
				infrastructure.KeyAction, "shutdown",
				infrastructure.KeyStatus, infrastructure.StatusFailed,
				infrastructure.KeyError, err.Error(),
			)
		}
	}()

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
