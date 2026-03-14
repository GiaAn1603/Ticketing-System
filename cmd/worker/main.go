package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/events"
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/repositories"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func run() error {
	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.ServerStartupTimeout)
	defer startupCancel()

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
		return fmt.Errorf("failed to connect postgres: %w", err)
	}
	defer func() {
		log.Println("[WORKER][INFO] Closing Postgres connection | action=close_postgres")
		if err := pgDB.Close(); err != nil {
			log.Printf("[WORKER][WARN] Postgres close error | err=%v", err)
		}
	}()

	pgRepo := repositories.NewPostgresRepo(pgDB)

	kafkaBrokers := []string{cfg.KafkaAddr}
	kafkaConsumer, err := events.NewKafkaConsumer(
		startupCtx,
		kafkaBrokers,
		cfg.KafkaTopicOrders,
		cfg.KafkaGroupID,
		cfg.KafkaNumPartitions,
		cfg.KafkaReplicationFactor,
		pgRepo,
		cfg.KafkaConsumerMinBytes,
		cfg.KafkaConsumerMaxBytes,
		cfg.KafkaTimeout,
		cfg.DBTimeout,
		cfg.KafkaCommitTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to init kafka consumer: %w", err)
	}
	defer func() {
		log.Println("[WORKER][INFO] Closing Kafka connection | action=close_kafka")
		if err := kafkaConsumer.Close(); err != nil {
			log.Printf("[WORKER][WARN] Kafka close error | err=%v", err)
		}
	}()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	workerErrChan := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("[WORKER][INFO] Worker started | status=running")
		if err := kafkaConsumer.ConsumeOrderEvent(workerCtx); err != nil {
			log.Printf("[WORKER][ERROR] Consumer loop crashed | err=%v", err)
			workerErrChan <- err
		}
	}()

	signalCtx, signalCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer signalCancel()

	select {
	case err := <-workerErrChan:
		return fmt.Errorf("worker stopped unexpectedly: %w", err)
	case <-signalCtx.Done():
		log.Println("[WORKER][INFO] Received shutdown signal | signal=SIGINT/SIGTERM")
	}

	log.Println("[WORKER][INFO] Shutting down worker | status=in_progress")

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
		return fmt.Errorf("worker forced to shutdown: %w", shutdownCtx.Err())
	}

	log.Println("[WORKER][INFO] Worker exited | status=done")

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Printf("[WORKER][FATAL] Worker startup failed | err=%v", err)
		os.Exit(1)
	}
}
