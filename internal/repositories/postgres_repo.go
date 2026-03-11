package repositories

import (
	"Ticketing-System/internal/models"
	"context"
	"database/sql"
	"fmt"
	"log"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{
		db: db,
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
		return fmt.Errorf("failed to insert order for request %s: %w", order.RequestID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for request %s: %w", order.RequestID, err)
	}

	if rowsAffected == 0 {
		log.Printf("[REPO][WARN] Order already exists, skipped (Idempotency check) | req_id=%s", order.RequestID)
		return nil
	}

	log.Printf("[REPO][INFO] Order saved to database | event_id=%s | user_id=%s | req_id=%s | qty=%d", order.EventID, order.UserID, order.RequestID, order.Quantity)

	return nil
}
