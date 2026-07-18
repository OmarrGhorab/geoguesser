package matchplay_test

import (
	"errors"
	"net/http"
	"testing"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/matchplay"
)

func TestMapErrorPrivacySafeNotFound(t *testing.T) {
	t.Parallel()

	for _, err := range []error{matchplay.ErrNotFound, matchplay.ErrForbiddenOpponent} {
		mapped := matchplay.MapError(err)
		var apiErr *apphttp.APIError
		if !errors.As(mapped, &apiErr) {
			t.Fatalf("expected APIError for %v", err)
		}
		if apiErr.Status != http.StatusNotFound || apiErr.Code != apphttp.ErrCodeNotFound {
			t.Fatalf("mapped = %+v, want privacy-safe not_found", apiErr)
		}
	}
}

func TestMapErrorStableCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err    error
		status int
		code   string
	}{
		{matchplay.ErrUnauthorized, http.StatusUnauthorized, apphttp.ErrCodeUnauthorized},
		{matchplay.ErrMatchNotActive, http.StatusConflict, matchplay.CodeMatchNotActive},
		{matchplay.ErrInvalidLeave, http.StatusUnprocessableEntity, apphttp.ErrCodeUnprocessable},
		{matchplay.ErrSpectateForbidden, http.StatusForbidden, matchplay.CodeSpectateForbidden},
		{matchplay.ErrProgressionPending, http.StatusAccepted, matchplay.CodeProgressionPending},
		{matchplay.ErrUnavailable, http.StatusServiceUnavailable, matchplay.CodeUnavailable},
		{matchplay.ErrInvalidJSON, http.StatusBadRequest, apphttp.ErrCodeInvalidJSON},
	}

	for _, tc := range cases {
		mapped := matchplay.MapError(tc.err)
		var apiErr *apphttp.APIError
		if !errors.As(mapped, &apiErr) {
			t.Fatalf("expected APIError for %v, got %T", tc.err, mapped)
		}
		if apiErr.Status != tc.status || apiErr.Code != tc.code {
			t.Fatalf("%v mapped to status=%d code=%q, want status=%d code=%q",
				tc.err, apiErr.Status, apiErr.Code, tc.status, tc.code)
		}
	}
}
