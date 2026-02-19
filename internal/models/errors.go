package models

import "errors"

var (
	ErrAlreadyProcessed = errors.New("request already processed")
	ErrInvalidInput     = errors.New("invalid input")
	ErrLimitExceeded    = errors.New("limit exceeded")
	ErrOutOfStock       = errors.New("out of stock")
	ErrEventNotFound    = errors.New("event not found")
	ErrInternal         = errors.New("internal error")
)
