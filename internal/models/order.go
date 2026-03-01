package models

import "time"

type OrderEvent struct {
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	RequestID string    `json:"request_id"`
	Quantity  int       `json:"quantity"`
	MaxLimit  int       `json:"max_limit"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
