package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

func autoMigrate(ctx context.Context, db *sql.DB) error {
	schemaPath := filepath.Join("migrations", "init_schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Printf("[INFRA][WARN] Could not find schema file at %s | action=skip_migration | err=%v", schemaPath, err)
		return nil
	}

	if _, err = db.ExecContext(ctx, string(schemaBytes)); err != nil {
		return fmt.Errorf("failed to execute schema script: %w", err)
	}

	log.Println("[INFRA][INFO] Database schema verified/initialized automatically | action=auto_migrate")

	return nil
}

func ConnectPostgres(ctx context.Context, addr, user, password, dbname string, maxOpen, maxIdle int, maxLifetime time.Duration) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, addr, dbname)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping failed at %s: %w", addr, err)
	}

	log.Printf("[INFRA][INFO] Connected to Postgres successfully | addr=%s | db=%s", addr, dbname)

	if err := autoMigrate(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to run auto migration: %w", err)
	}

	return db, nil
}
