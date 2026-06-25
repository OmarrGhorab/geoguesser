package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raven/geoguess/backend/internal/middleware"
)

func TestMetricsAuthAllowsRequestsWhenTokenUnset(t *testing.T) {
	handler := middleware.MetricsAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMetricsAuthRequiresBearerToken(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", authorization: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "missing bearer scheme", authorization: "secret-token", wantStatus: http.StatusUnauthorized},
		{name: "correct token", authorization: "Bearer secret-token", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tt.authorization != "" {
				r.Header.Set("Authorization", tt.authorization)
			}

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
