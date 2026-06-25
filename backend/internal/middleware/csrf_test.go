package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/auth"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
)

func TestCSRFRequiresMatchingCookieAndHeader(t *testing.T) {
	manager, err := auth.NewCSRFManager("test-csrf-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("csrf manager setup failed: %v", err)
	}
	opts := auth.CookieOptions{SameSite: http.SameSiteLaxMode}
	handler := appmiddleware.CSRF(testCSRFValidator{manager}, opts, slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	validToken, err := manager.Generate()
	if err != nil {
		t.Fatalf("csrf token generation failed: %v", err)
	}
	otherToken, err := manager.Generate()
	if err != nil {
		t.Fatalf("csrf token generation failed: %v", err)
	}

	cases := []struct {
		name       string
		header     string
		cookie     string
		wantStatus int
	}{
		{name: "matching cookie and header", header: validToken, cookie: validToken, wantStatus: http.StatusNoContent},
		{name: "missing cookie", header: validToken, wantStatus: http.StatusForbidden},
		{name: "missing header", cookie: validToken, wantStatus: http.StatusForbidden},
		{name: "mismatched signed token", header: validToken, cookie: otherToken, wantStatus: http.StatusForbidden},
		{name: "invalid token", header: "invalid", cookie: "invalid", wantStatus: http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/unsafe", nil)
			if tc.header != "" {
				r.Header.Set("X-CSRF-Token", tc.header)
			}
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: auth.CSRFTokenCookieName, Value: tc.cookie})
			}

			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestCSRFIssuesCookieForSafeRequests(t *testing.T) {
	manager, err := auth.NewCSRFManager("test-csrf-secret-at-least-32-bytes-long")
	if err != nil {
		t.Fatalf("csrf manager setup failed: %v", err)
	}
	opts := auth.CookieOptions{SameSite: http.SameSiteLaxMode}
	handler := appmiddleware.CSRF(testCSRFValidator{manager}, opts, slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/safe", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	cookie := findSetCookie(w.Result(), auth.CSRFTokenCookieName)
	if cookie == nil {
		t.Fatal("expected csrf cookie to be issued")
	}
	if cookie.HttpOnly {
		t.Fatal("csrf cookie must be readable by the browser")
	}
	if !manager.Validate(cookie.Value) {
		t.Fatal("issued csrf cookie is not valid")
	}
	if cookie.Expires.Before(time.Now().UTC()) {
		t.Fatal("csrf cookie is already expired")
	}
}

func findSetCookie(resp *http.Response, name string) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

type testCSRFValidator struct {
	manager *auth.CSRFManager
}

func (t testCSRFValidator) GenerateCSRF() (string, error) {
	return t.manager.Generate()
}

func (t testCSRFValidator) ValidateCSRF(token string) bool {
	return t.manager.Validate(token)
}
