package home

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

type serviceStub struct {
	response *Response
	err      error
}

func (s serviceStub) Get(context.Context, *session.Context) (*Response, error) {
	return s.response, s.err
}

func TestHandlerGetWritesHomeSnapshot(t *testing.T) {
	userID := uuid.New()
	handler := NewHandler(serviceStub{response: &Response{Viewer: ViewerDTO{UserID: userID, DisplayName: "Radiant"}}}, slog.Default(), nil)
	request := httptest.NewRequest(http.MethodGet, "/home", nil)
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var payload Response
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Viewer.UserID != userID {
		t.Fatalf("viewer = %+v", payload.Viewer)
	}
}

func TestHandlerGetMapsUnavailableTo503(t *testing.T) {
	handler := NewHandler(serviceStub{err: ErrUnavailable}, slog.Default(), nil)
	request := httptest.NewRequest(http.MethodGet, "/home", nil)
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var payload apphttp.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != CodeUnavailable {
		t.Fatalf("error code = %q, want %q", payload.Error.Code, CodeUnavailable)
	}
}

func TestToAPIErrorMapsUnauthorized(t *testing.T) {
	apiErr := ToAPIError(ErrUnauthorized)
	if apiErr.Status != http.StatusUnauthorized || !errors.Is(apiErr, ErrUnauthorized) {
		t.Fatalf("ToAPIError() = %+v", apiErr)
	}
}
