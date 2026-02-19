package apperrors

import "errors"

var (
	ErrAlreadyProcessed      = errors.New("request already processed")
	ErrInvalidInput          = errors.New("invalid input parameters")
	ErrEventNotFound         = errors.New("event not found")
	ErrOutOfStock            = errors.New("sold out")
	ErrPurchaseLimitExceeded = errors.New("purchase limit exceeded")
	ErrRateLimitExceeded     = errors.New("too many requests")
	ErrInternal              = errors.New("internal server error")
)
