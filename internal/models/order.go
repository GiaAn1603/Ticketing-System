package models

import "time"

type OrderEvent struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	RequestID string    `json:"request_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderDB struct {
	ID        int       `db:"id"`
	EventID   string    `db:"event_id"`
	UserID    string    `db:"user_id"`
	RequestID string    `db:"request_id"`
	Quantity  int       `db:"quantity"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
