package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

func autoMigrate(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	schemaPath := filepath.Join("migrations", "init_schema.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		logger.Warn(
			"Schema file found failed",
			"path", schemaPath,
			KeyError, err.Error(),
		)

		return nil
	}

	if _, err = db.ExecContext(ctx, string(schemaBytes)); err != nil {
		return fmt.Errorf("execute schema script: %w", err)
	}

	logger.Info("Database schema initialized successfully")

	return nil
}

func ConnectPostgres(
	ctx context.Context,
	addr, user, password, dbName string,
	maxOpen, maxIdle int,
	maxLifetime time.Duration,
) (*sql.DB, error) {
	logger := GetLogger("INFRA_POSTGRES")

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, addr, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Info(
		"Postgres connected successfully",
		"addr", addr,
		"db_name", dbName,
	)

	if err := autoMigrate(ctx, db, logger); err != nil {
		return nil, fmt.Errorf("run auto migration: %w", err)
	}

	return db, nil
}
