package middleware

import (
	"log/slog"
	"net/http"
	"time"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// CSRFValidator generates and validates CSRF tokens.
type CSRFValidator interface {
	GenerateCSRF() (string, error)
	ValidateCSRF(token string) bool
}

// CSRFOptions carries cookie settings for the CSRF cookie.
type CSRFOptions interface {
	SetCSRFCookie(w http.ResponseWriter, token string, expiresAt time.Time)
}

// CSRF validates the X-CSRF-Token header or query value against the signed csrf_token
// cookie for unsafe HTTP methods. Safe methods (GET, HEAD, OPTIONS, TRACE) are
// exempt. The middleware also issues a CSRF token cookie if one is not present.
func CSRF(validator CSRFValidator, opts CSRFOptions, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				ensureCSRFCookie(w, r, validator, opts)
				next.ServeHTTP(w, r)
				return
			}

			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				token = r.URL.Query().Get("csrf_token")
			}
			cookieToken := readCookieValue(r, "csrf_token")
			if token == "" || cookieToken == "" || token != cookieToken || !validator.ValidateCSRF(token) {
				apphttp.Error(w, r, logger, apphttp.ErrForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, validator CSRFValidator, opts CSRFOptions) {
	if existing := readCookieValue(r, "csrf_token"); existing != "" {
		return
	}
	token, err := validator.GenerateCSRF()
	if err != nil {
		return
	}
	opts.SetCSRFCookie(w, token, time.Now().UTC().Add(24*time.Hour))
}
