package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	"github.com/raven/geoguess/backend/internal/session"
)

func TestHandlerGetCurrentProfileRequiresRegisteredSession(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore(), nil), slog.Default())
	req := requestWithSession(http.MethodGet, "/api/v1/profile", nil, &session.Context{Kind: session.KindGuest, GuestID: ptr("guest-1")})
	w := httptest.NewRecorder()

	handler.GetCurrentProfile(w, req)

	assertError(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestHandlerGetCurrentProfileReturnsSafeShape(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{
		UserID:      userID,
		Email:       "owner@example.com",
		DisplayName: "Raven",
		Locale:      "en",
		Preferences: map[string]any{"distance_unit": "km"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	handler := NewHandler(NewService(fs, nil), slog.Default())
	req := requestWithSession(http.MethodGet, "/api/v1/profile", nil, registeredSessionPtr(userID))
	w := httptest.NewRecorder()

	handler.GetCurrentProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "owner@example.com") || strings.Contains(body, "session") || strings.Contains(body, "token") {
		t.Fatalf("unexpected profile response body: %s", body)
	}
}

func TestHandlerUpdateProfileRejectsInvalidBody(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore(), nil), slog.Default())
	req := requestWithSession(http.MethodPatch, "/api/v1/profile", strings.NewReader("{"), registeredSessionPtr(uuid.New()))
	w := httptest.NewRecorder()

	handler.UpdateProfile(w, req)

	assertError(t, w, http.StatusBadRequest, "invalid_json")
}

func TestHandlerUpdateProfileMapsValidationFields(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Raven", Locale: "en", Preferences: map[string]any{}}
	handler := NewHandler(NewService(fs, nil), slog.Default())
	req := requestWithSession(http.MethodPatch, "/api/v1/profile", strings.NewReader(`{"timezone":"Not/AZone"}`), registeredSessionPtr(userID))
	w := httptest.NewRecorder()

	handler.UpdateProfile(w, req)

	assertError(t, w, http.StatusBadRequest, "validation_failed")
	if !strings.Contains(w.Body.String(), `"timezone"`) {
		t.Fatalf("expected timezone field error, got %s", w.Body.String())
	}
}

func TestHandlerPublicStatsErrorsAreStable(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore(), nil), slog.Default())
	router := chi.NewRouter()
	router.Get("/api/v1/users/{userId}/stats", handler.GetPublicProfile)

	for _, path := range []string{"/api/v1/users/not-a-uuid/stats", "/api/v1/users/" + uuid.New().String() + "/stats"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		assertError(t, w, http.StatusNotFound, "not_found")
	}
}

func TestHandlerGameHistoryHandlesPaginationAndErrors(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.public[userID] = &PublicProfileSummary{UserID: userID, DisplayName: "Raven"}
	cursor := "next-cursor"
	fs.history[userID] = &GameHistoryPage{
		Items:      []GameHistoryItem{{GameID: uuid.New(), Mode: "solo", Status: "completed", RoundCount: 5, CreatedAt: time.Now()}},
		Limit:      2,
		NextCursor: &cursor,
	}
	handler := NewHandler(NewService(fs, nil), slog.Default())
	router := chi.NewRouter()
	router.Get("/api/v1/users/{userId}/games", handler.GetGameHistory)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/games?limit=2", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "latitude") || strings.Contains(body, "longitude") || strings.Contains(body, "provider") {
		t.Fatalf("history leaked private fields: %s", body)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/games?limit=101", nil))
	assertError(t, w, http.StatusBadRequest, "validation_failed")
}

func TestHandlerRecordRateLimitedIncrementsMetric(t *testing.T) {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_handler_profile_updates_rate_limited_total",
		Help: "Test profile update rate-limit counter.",
	})
	metrics := &Metrics{ProfileRateLimitedTotal: counter}
	handler := NewHandler(NewService(newFakeStore(), metrics), slog.Default())

	handler.RecordRateLimited(httptest.NewRequest(http.MethodPatch, "/api/v1/profile", nil))

	metric := &dto.Metric{}
	if err := counter.Write(metric); err != nil {
		t.Fatalf("read profile rate-limit metric: %v", err)
	}
	if got := metric.GetCounter().GetValue(); got != 1 {
		t.Fatalf("expected rate-limit metric increment, got %v", got)
	}
}

func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, w.Code, w.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Error.Code != code {
		t.Fatalf("expected code %q, got %q in %s", code, payload.Error.Code, w.Body.String())
	}
}

func ptr(v string) *string {
	return &v
}

func registeredSessionPtr(userID uuid.UUID) *session.Context {
	sc := registeredSession(userID)
	return &sc
}

func requestWithSession(method string, target string, body io.Reader, sess *session.Context) *http.Request {
	req := httptest.NewRequest(method, target, body)
	recorder := httptest.NewRecorder()
	appmiddleware.SessionLoader(staticSessionResolver{session: sess}, "access_token", "guest_session")(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		req = r
	})).ServeHTTP(recorder, req)
	return req
}

type staticSessionResolver struct {
	session *session.Context
}

func (r staticSessionResolver) ResolveSession(context.Context, string) (*session.Context, error) {
	if r.session == nil {
		return &session.Context{Kind: session.KindAnonymous}, nil
	}
	return r.session, nil
}

func (staticSessionResolver) ResolveGuestSession(context.Context, string) (string, error) {
	return "", nil
}
