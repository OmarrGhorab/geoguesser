package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/raven/geoguess/backend/internal/session"
	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// sessionContextKey is the request context key for the resolved session.
type sessionContextKey struct{}

// SessionResolver resolves an access token into a session context.
type SessionResolver interface {
	ResolveSession(ctx context.Context, accessToken string) (*session.Context, error)
	ResolveGuestSession(ctx context.Context, signed string) (string, error)
}

// SessionLoader loads registered and guest sessions from cookies and attaches
// a session.Context to the request context. It never rejects requests; endpoints
// that require auth should check the session context.
func SessionLoader(resolver SessionResolver, accessCookieName, guestCookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc, err := resolver.ResolveSession(r.Context(), readCookieValue(r, accessCookieName))
			if err != nil || sc.Kind == session.KindAnonymous {
				guestToken := readCookieValue(r, guestCookieName)
				if guestToken != "" {
					guestID, err := resolver.ResolveGuestSession(r.Context(), guestToken)
					if err == nil {
						sc = &session.Context{
							Kind:    session.KindGuest,
							GuestID: &guestID,
						}
					}
				}
			}
			r = r.WithContext(context.WithValue(r.Context(), sessionContextKey{}, sc))
			next.ServeHTTP(w, r)
		})
	}
}

// SessionFromContext returns the session context from the request context.
func SessionFromContext(ctx context.Context) *session.Context {
	if sc, ok := ctx.Value(sessionContextKey{}).(*session.Context); ok {
		return sc
	}
	return &session.Context{Kind: session.KindAnonymous}
}

// RequireAuth rejects requests without a registered user session.
func RequireAuth(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc := SessionFromContext(r.Context())
			if !sc.IsRegistered() {
				apphttp.Error(w, r, logger, apphttp.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func readCookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
