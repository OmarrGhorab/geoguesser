package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// MetricsAuth protects the Prometheus scrape endpoint when a token is set.
func MetricsAuth(token string) func(http.Handler) http.Handler {
	token = strings.TrimSpace(token)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
