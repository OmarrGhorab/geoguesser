package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// RateLimiter is the interface for rate limit decisions.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)
}

// RateLimitConfig defines a rate limit rule.
type RateLimitConfig struct {
	Limit  int
	Window time.Duration
}

// RateLimit applies a sliding-window rate limit using a key extractor.
func RateLimit(limiter RateLimiter, config RateLimitConfig, keyFunc func(r *http.Request) string, logger *slog.Logger) func(http.Handler) http.Handler {
	return RateLimitWithObserver(limiter, config, keyFunc, logger, nil)
}

// RateLimitWithObserver applies a sliding-window rate limit and calls
// onRejected after a request is rejected. The hook must stay privacy-safe:
// callers should record counters or static log fields, not key material.
func RateLimitWithObserver(limiter RateLimiter, config RateLimitConfig, keyFunc func(r *http.Request) string, logger *slog.Logger, onRejected func(*http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			allowed, _, err := limiter.Allow(r.Context(), key, config.Limit, config.Window)
			if err != nil {
				logger.ErrorContext(r.Context(), "rate limit check failed", slog.Any("error", err))
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				if onRejected != nil {
					onRejected(r)
				}
				w.Header().Set("Retry-After", strconv.Itoa(int(config.Window.Seconds())))
				apphttp.Error(w, r, logger, apphttp.ErrRateLimited)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByIP returns a key extractor using the request IP.
func RateLimitByIP(prefix string) func(r *http.Request) string {
	return func(r *http.Request) string {
		return fmt.Sprintf("%s:%s", prefix, remoteHost(r))
	}
}

// RateLimitByCookie returns a key extractor using a cookie value.
func RateLimitByCookie(prefix, cookieName string) func(r *http.Request) string {
	return func(r *http.Request) string {
		c, err := r.Cookie(cookieName)
		if err != nil || c.Value == "" {
			return fmt.Sprintf("%s:%s", prefix, remoteHost(r))
		}
		return fmt.Sprintf("%s:%s", prefix, c.Value)
	}
}

func remoteHost(r *http.Request) string {
	host := strings.TrimSpace(r.RemoteAddr)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	return host
}
