package domain

import "net/http"

// APIError is the structured error the API returns for every failure. The
// frontend maps Code → a user-facing message; Message is a developer-facing
// fallback and is never localized.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	status  int
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// HTTPStatus returns the HTTP status to send with this error.
func (e *APIError) HTTPStatus() int { return e.status }

// NewAPIError builds an APIError with an explicit HTTP status.
func NewAPIError(status int, code, message string) *APIError {
	return &APIError{Code: code, Message: message, status: status}
}

// Predefined errors. Codes are stable — the frontend keys off them.
var (
	ErrInvalidCredentials = NewAPIError(http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "invalid email or password")
	ErrTokenMissing       = NewAPIError(http.StatusUnauthorized, "AUTH_TOKEN_MISSING", "authorization token missing")
	ErrTokenInvalid       = NewAPIError(http.StatusUnauthorized, "AUTH_TOKEN_INVALID", "authorization token invalid")
	ErrTokenExpired       = NewAPIError(http.StatusUnauthorized, "AUTH_TOKEN_EXPIRED", "authorization token expired")
	ErrForbidden          = NewAPIError(http.StatusForbidden, "AUTH_FORBIDDEN", "insufficient permissions")
	ErrValidation         = NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "invalid request")
	ErrRateLimited        = NewAPIError(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
	ErrPayloadTooLarge    = NewAPIError(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
	ErrInternal           = NewAPIError(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
)
