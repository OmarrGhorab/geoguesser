package leaderboards

import (
	"errors"
	"net/http"
	"testing"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

func TestToAPIErrorMapsDependencyFailureToServiceUnavailable(t *testing.T) {
	err := ToAPIError(ErrDependencyFailure)
	var apiErr *apphttp.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ToAPIError returned %T, want *http.APIError", err)
	}
	if apiErr.Status != http.StatusServiceUnavailable || apiErr.Code != "leaderboards_unavailable" {
		t.Fatalf("API error = status %d code %q, want 503 leaderboards_unavailable", apiErr.Status, apiErr.Code)
	}
}
