package apperrors

import "errors"

var (
	AlreadyProcessed   = errors.New("request already processed")
	InvalidInput       = errors.New("invalid input")
	LimitExceeded      = errors.New("limit exceeded")
	OutOfStock         = errors.New("out of stock")
	EventNotFound      = errors.New("event not found")
	EventAlreadyExists = errors.New("event already exists")
	MissingRequestID   = errors.New("missing request id")
	Internal           = errors.New("internal error")
)
