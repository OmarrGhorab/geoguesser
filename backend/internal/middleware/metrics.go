package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/raven/geoguess/backend/internal/platform/observability"
)

// Metrics records Prometheus HTTP request duration and total counters.
func Metrics(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(startedAt).Seconds()
			path := chi.RouteContext(r.Context()).RoutePattern()
			if path == "" {
				path = r.URL.Path
			}
			labels := []string{r.Method, path, strconv.Itoa(ww.Status())}
			metrics.HTTPRequestDuration.WithLabelValues(labels...).Observe(duration)
			metrics.HTTPRequestsTotal.WithLabelValues(labels...).Inc()
		})
	}
}
