package http_test

import (
	"context"
	"testing"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

func TestRequestIDContext(t *testing.T) {
	ctx := apphttp.WithRequestID(context.Background(), "req_123")
	if got := apphttp.RequestIDFromContext(ctx); got != "req_123" {
		t.Errorf("RequestIDFromContext = %q, want req_123", got)
	}
}

func TestRequestIDFromContextMissing(t *testing.T) {
	if got := apphttp.RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext = %q, want empty", got)
	}
}
