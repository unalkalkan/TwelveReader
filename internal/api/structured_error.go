package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// StructuredError represents the standardized error response format for /api/v1 endpoints.
type StructuredError struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the fields of a structured API error.
type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// Predefined error codes for common API errors.
const (
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeInternal            = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"
)

// WriteStructuredError writes a structured error JSON response with the given status code.
// It reads the request ID from the request context (if present).
func WriteStructuredError(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int) {
	reqID := FromContext(r.Context())
	err := StructuredError{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(err)
}

// WriteStructuredErrorf is like WriteStructuredError but supports fmt-style formatting for the message.
func WriteStructuredErrorf(w http.ResponseWriter, r *http.Request, code string, format string, args []interface{}, statusCode int) {
	message := fmt.Sprintf(format, args...)
	WriteStructuredError(w, r, code, message, statusCode)
}

// HTTPStatusCodeForCode maps a common error code to its appropriate HTTP status.
func HTTPStatusCodeForCode(code string) int {
	switch code {
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrCodeTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// WriteMethodNotAllowedError writes a structured 405 response.
func WriteMethodNotAllowedError(w http.ResponseWriter, r *http.Request) {
	WriteStructuredError(w, r, ErrCodeMethodNotAllowed, "Method not allowed", http.StatusMethodNotAllowed)
}

// WriteNotFoundError writes a structured 404 response.
func WriteNotFoundError(w http.ResponseWriter, r *http.Request, resource string) {
	WriteStructuredError(w, r, ErrCodeNotFound, resource+" not found", http.StatusNotFound)
}
