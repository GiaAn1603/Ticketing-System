package repositories

import (
	"Ticketing-System/internal/infrastructure"
	"Ticketing-System/internal/models"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

type PostgresRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	logger := infrastructure.GetLogger("POSTGRES_REPO")

	return &PostgresRepo{
		db:  db,
		log: logger,
	}
}

func (r *PostgresRepo) InsertOrderIfNotExists(ctx context.Context, order models.OrderEvent) error {
	query := `
    INSERT INTO orders (event_id, user_id, request_id, quantity, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id, user_id, request_id) DO NOTHING
	`

	result, err := r.db.ExecContext(ctx, query, order.EventID, order.UserID, order.RequestID, order.Quantity, order.Status, order.Timestamp)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.log.Debug(
			"Order already exists",
			"event_id", order.EventID,
			"user_id", order.UserID,
			"request_id", order.RequestID,
			"quantity", order.Quantity,
			"order_status", order.Status,
		)
		return nil
	}

	r.log.Debug(
		"Order saved successfully",
		"event_id", order.EventID,
		"user_id", order.UserID,
		"request_id", order.RequestID,
		"quantity", order.Quantity,
		"order_status", order.Status,
	)

	return nil
}
