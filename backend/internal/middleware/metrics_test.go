package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

func TestMetricsMiddleware(t *testing.T) {
	metrics, err := observability.NewMetrics("geoguess-test")
	if err != nil {
		t.Fatalf("metrics setup failed: %v", err)
	}

	mw := middleware.Metrics(metrics)
	r := chi.NewRouter()
	r.Use(mw)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
}
