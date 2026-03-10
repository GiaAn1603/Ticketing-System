package main

import (
	"Ticketing-System/internal/config"
	"Ticketing-System/internal/infrastructure"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func run() error {
	cfg := config.LoadConfig()

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startupCancel()

	pgDB, err := infrastructure.ConnectPostgres(startupCtx, cfg.PostgresAddr, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB)
	if err != nil {
		return fmt.Errorf("failed to connect postgres: %w", err)
	}
	defer func() {
		log.Println("[WORKER][INFO] Closing Postgres connection | action=close_postgres")
		if err := pgDB.Close(); err != nil {
			log.Printf("[WORKER][WARN] Postgres close error | err=%v", err)
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
		<-workerCtx.Done()
		time.Sleep(2 * time.Second)
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
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
