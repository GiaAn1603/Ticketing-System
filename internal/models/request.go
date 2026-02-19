package models

type InitRequest struct {
	EventID string `json:"event_id" binding:"required"`
	Stock   int    `json:"stock" binding:"required,gte=1"`
}

type BuyRequest struct {
	EventID  string `json:"event_id" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
	MaxLimit int    `json:"max_limit" binding:"required,gt=0"`
}
