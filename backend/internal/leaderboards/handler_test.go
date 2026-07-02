package leaderboards

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandlerRoutesReturnJSON(t *testing.T) {
	handler := NewHandler(handlerServiceStub{}, nil)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	for _, path := range []string{
		"/leaderboards/global",
		"/leaderboards/daily",
		"/leaderboards/maps/not-a-uuid",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("GET %s content-type = %q body=%s", path, got, rec.Body.String())
		}
	}
}

func TestHandlerRejectsMalformedLimit(t *testing.T) {
	handler := NewHandler(handlerServiceStub{}, nil)
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	for _, path := range []string{
		"/leaderboards/global?limit=abc",
		"/leaderboards/daily?limit=0",
		"/leaderboards/maps/00000000-0000-0000-0000-000000000001?limit=101",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d body=%s, want %d", path, rec.Code, rec.Body.String(), http.StatusBadRequest)
		}
		if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("GET %s content-type = %q body=%s", path, got, rec.Body.String())
		}
	}
}

type handlerServiceStub struct{}

func (handlerServiceStub) GetGlobal(context.Context, int, string) (*Response, error) {
	return &Response{}, nil
}

func (handlerServiceStub) GetDaily(context.Context, int, string, string) (*Response, error) {
	return &Response{}, nil
}

func (handlerServiceStub) GetMap(context.Context, string, int, string) (*Response, error) {
	return nil, ErrInvalidMapID
}
