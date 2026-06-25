package http

import (
	"encoding/json"
	"net/http"
)

// Response is a generic wrapper for single-resource API responses.
type Response[T any] struct {
	Data T `json:"data"`
}

// ListResponse is a generic wrapper for list API responses with pagination.
type ListResponse[T any] struct {
	Data []T      `json:"data"`
	Page PageInfo `json:"page"`
}

// PageInfo carries cursor pagination metadata.
type PageInfo struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

// JSON writes a JSON response with the given status code. It encodes failures
// as a generic internal error envelope so callers are not left with a blank
// response on encoding failures.
func JSON(w http.ResponseWriter, r *http.Request, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Encoding failures are unexpected; write a minimal safe envelope.
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Code:    ErrCodeInternal,
				Message: MsgInternalError,
			},
		})
	}
}

// OK writes a 200 OK response with the payload.
func OK(w http.ResponseWriter, r *http.Request, payload any) {
	JSON(w, r, http.StatusOK, payload)
}

// Created writes a 201 Created response with the payload.
func Created(w http.ResponseWriter, r *http.Request, payload any) {
	JSON(w, r, http.StatusCreated, payload)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
