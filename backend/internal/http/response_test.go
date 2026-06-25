package http_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

func TestJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	apphttp.OK(w, r, map[string]string{"status": "ok"})

	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("body does not contain expected payload: %s", body)
	}
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	apiErr := apphttp.ErrValidationFailed.WithFields(apphttp.FieldError{
		Name:    "email",
		Code:    "invalid_email",
		Message: "Email must be valid.",
	})

	apphttp.Error(w, r, logger, apiErr)

	res := w.Result()
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}

	var payload apphttp.ErrorResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if payload.Error.Code != apphttp.ErrCodeValidationFailed {
		t.Errorf("error code = %q, want %q", payload.Error.Code, apphttp.ErrCodeValidationFailed)
	}
	if len(payload.Error.Fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(payload.Error.Fields))
	}
	if payload.Error.Fields[0].Name != "email" {
		t.Errorf("field name = %q, want email", payload.Error.Fields[0].Name)
	}
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"name":"test"}`, false},
		{"invalid json", `{"name":}`, true},
		{"multiple values", `{"name":"a"}{"name":"b"}`, true},
		{"unknown field", `{"name":"test","extra":"value"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			var dst struct {
				Name string `json:"name"`
			}
			err := apphttp.DecodeJSON(r, &dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJSON error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
