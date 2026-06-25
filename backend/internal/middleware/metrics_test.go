package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

func TestMetricsMiddleware(t *testing.T) {
	metrics, err := observability.NewMetrics("geoguess-test")
	if err != nil {
		t.Fatalf("metrics setup failed: %v", err)
	}

	mw := middleware.Metrics(metrics)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTeapot)
	}
}
