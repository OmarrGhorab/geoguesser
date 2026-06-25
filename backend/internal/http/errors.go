package http

import (
	"errors"
	"log/slog"
	"net/http"
)

// FieldError describes a single validation failure for a request field.
type FieldError struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorDetail is the standard API error payload.
type ErrorDetail struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	RequestID string       `json:"request_id,omitempty"`
	Fields    []FieldError `json:"fields,omitempty"`
}

// ErrorResponse wraps ErrorDetail in the documented envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// Common stable error codes used across the API.
const (
	ErrCodeInternal         = "internal_error"
	ErrCodeValidationFailed = "validation_failed"
	ErrCodeInvalidJSON      = "invalid_json"
	ErrCodeUnauthorized     = "unauthorized"
	ErrCodeForbidden        = "forbidden"
	ErrCodeNotFound         = "not_found"
	ErrCodeConflict         = "conflict"
	ErrCodeUnprocessable    = "unprocessable_entity"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeMethodNotAllowed = "method_not_allowed"
	ErrCodeRequestTooLarge  = "request_too_large"
	ErrCodeUnsupportedMedia = "unsupported_media_type"
)

// Common safe user-facing messages. Callers should localize on the frontend.
const (
	MsgInternalError        = "An unexpected error occurred."
	MsgValidationFailed     = "The request is invalid."
	MsgInvalidJSON          = "The request body is not valid JSON."
	MsgUnauthorized         = "Authentication is required."
	MsgForbidden            = "You are not allowed to perform this action."
	MsgNotFound             = "The requested resource was not found."
	MsgConflict             = "The request conflicts with the current state."
	MsgUnprocessable        = "The request cannot be processed in the current state."
	MsgRateLimited          = "Too many requests. Please try again later."
	MsgMethodNotAllowed     = "The requested method is not allowed."
	MsgRequestTooLarge      = "The request body is too large."
	MsgUnsupportedMediaType = "The request content type is not supported."
)

// APIError is a domain-aware HTTP error that handlers can return.
type APIError struct {
	Status  int
	Code    string
	Message string
	Fields  []FieldError
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *APIError) Unwrap() error {
	return e.Cause
}

// NewAPIError builds an APIError with the given status, code, and message.
func NewAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// WithFields returns a copy of the APIError with field errors attached.
func (e *APIError) WithFields(fields ...FieldError) *APIError {
	cpy := *e
	cpy.Fields = append(cpy.Fields, fields...)
	return &cpy
}

// WithCause returns a copy of the APIError with a wrapped cause.
func (e *APIError) WithCause(cause error) *APIError {
	cpy := *e
	cpy.Cause = cause
	return &cpy
}

// Predefined API errors. Use WithFields/WithCause to add detail.
var (
	ErrInternal         = NewAPIError(http.StatusInternalServerError, ErrCodeInternal, MsgInternalError)
	ErrValidationFailed = NewAPIError(http.StatusBadRequest, ErrCodeValidationFailed, MsgValidationFailed)
	ErrInvalidJSON      = NewAPIError(http.StatusBadRequest, ErrCodeInvalidJSON, MsgInvalidJSON)
	ErrUnauthorized     = NewAPIError(http.StatusUnauthorized, ErrCodeUnauthorized, MsgUnauthorized)
	ErrForbidden        = NewAPIError(http.StatusForbidden, ErrCodeForbidden, MsgForbidden)
	ErrNotFound         = NewAPIError(http.StatusNotFound, ErrCodeNotFound, MsgNotFound)
	ErrConflict         = NewAPIError(http.StatusConflict, ErrCodeConflict, MsgConflict)
	ErrUnprocessable    = NewAPIError(http.StatusUnprocessableEntity, ErrCodeUnprocessable, MsgUnprocessable)
	ErrRateLimited      = NewAPIError(http.StatusTooManyRequests, ErrCodeRateLimited, MsgRateLimited)
	ErrMethodNotAllowed = NewAPIError(http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, MsgMethodNotAllowed)
	ErrRequestTooLarge  = NewAPIError(http.StatusRequestEntityTooLarge, ErrCodeRequestTooLarge, MsgRequestTooLarge)
	ErrUnsupportedMedia = NewAPIError(http.StatusUnsupportedMediaType, ErrCodeUnsupportedMedia, MsgUnsupportedMediaType)
)

// Error writes a standardized error response. It extracts a request ID from the
// request context when available and logs unexpected server errors.
func Error(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		apiErr = ErrInternal.WithCause(err)
	}

	if apiErr.Status >= http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "http server error",
			slog.String("code", apiErr.Code),
			slog.String("path", r.URL.Path),
			slog.String("method", r.Method),
			slog.Any("error", err),
		)
	}

	WriteError(w, r, apiErr.Status, apiErr.Code, apiErr.Message, apiErr.Fields)
}

// WriteError writes the documented error envelope with the supplied fields.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, fields []FieldError) {
	detail := ErrorDetail{
		Code:    code,
		Message: message,
		Fields:  fields,
	}
	if reqID := RequestIDFromContext(r.Context()); reqID != "" {
		detail.RequestID = reqID
	}

	JSON(w, r, status, ErrorResponse{Error: detail})
}
