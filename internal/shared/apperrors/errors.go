package apperrors

type ErrorCode string

const (
	CodeAlreadyProcessed   ErrorCode = "ALREADY_PROCESSED"
	CodeInvalidInput       ErrorCode = "INVALID_INPUT"
	CodePurchaseLimit      ErrorCode = "PURCHASE_LIMIT"
	CodeOutOfStock         ErrorCode = "OUT_OF_STOCK"
	CodeEventNotFound      ErrorCode = "EVENT_NOT_FOUND"
	CodeRateLimit          ErrorCode = "RATE_LIMIT"
	CodeEventAlreadyExists ErrorCode = "EVENT_ALREADY_EXISTS"
	CodeMissingRequestID   ErrorCode = "MISSING_REQUEST_ID"
	CodeInternal           ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
	Code    ErrorCode
	Status  string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrAlreadyProcessed      = &AppError{Code: CodeAlreadyProcessed, Status: "success", Message: "Request already processed"}
	ErrInvalidInput          = &AppError{Code: CodeInvalidInput, Status: "fail", Message: "Invalid input parameters"}
	ErrPurchaseLimitExceeded = &AppError{Code: CodePurchaseLimit, Status: "fail", Message: "Purchase limit exceeded"}
	ErrOutOfStock            = &AppError{Code: CodeOutOfStock, Status: "fail", Message: "Sold out"}
	ErrEventNotFound         = &AppError{Code: CodeEventNotFound, Status: "fail", Message: "Event not found"}
	ErrRateLimitExceeded     = &AppError{Code: CodeRateLimit, Status: "fail", Message: "Too many requests, please try again later"}
	ErrEventAlreadyExists    = &AppError{Code: CodeEventAlreadyExists, Status: "fail", Message: "Event already exists"}
	ErrMissingRequestID      = &AppError{Code: CodeMissingRequestID, Status: "fail", Message: "Missing X-Request-ID header"}
	ErrInternal              = &AppError{Code: CodeInternal, Status: "error", Message: "Internal server error"}
)
