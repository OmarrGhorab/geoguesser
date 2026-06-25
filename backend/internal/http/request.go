package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// DefaultMaxBodyBytes is the maximum request body size for JSON endpoints.
const DefaultMaxBodyBytes = 1 << 20 // 1 MiB

// DecodeJSON reads and validates a JSON request body into dst. It enforces a
// maximum body size and returns stable API errors for malformed or oversized
// input.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
	defer func() { _ = body.Close() }()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return ErrRequestTooLarge.WithCause(err)
		}
		return ErrInvalidJSON.WithCause(err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidJSON.WithCause(errors.New("request body must contain a single JSON object"))
	}

	return nil
}

// RequestIDContextKey is the context key for request IDs.
type requestIDContextKey struct{}

// WithRequestID returns a context with the request ID attached.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext returns the request ID stored in the context, if any.
// It checks both the application key and the chi middleware key.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return v
	}
	return middleware.GetReqID(ctx)
}
