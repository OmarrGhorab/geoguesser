package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/raven/geoguess/backend/internal/config"
)

const (
	// AccessTokenCookieName is the HTTP-only access token cookie name.
	AccessTokenCookieName = "access_token"
	// RefreshTokenCookieName is the HTTP-only refresh token cookie name.
	RefreshTokenCookieName = "refresh_token"
	// CSRFTokenCookieName is the readable CSRF token cookie name.
	CSRFTokenCookieName = "csrf_token"
	// GuestSessionCookieName is the signed guest session cookie name.
	GuestSessionCookieName = "guest_session"
)

// CookieOptions holds cookie settings derived from config.
type CookieOptions struct {
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

// NewCookieOptions builds cookie options from configuration.
func NewCookieOptions(cfg config.Config) CookieOptions {
	sameSite := http.SameSiteLaxMode
	switch strings.ToLower(cfg.CookieSameSite) {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	case "lax":
		sameSite = http.SameSiteLaxMode
	}
	return CookieOptions{
		Domain:   cfg.CookieDomain,
		Secure:   cfg.CookieSecure,
		SameSite: sameSite,
	}
}

// SetAuthCookies writes access and refresh token cookies.
func SetAuthCookies(w http.ResponseWriter, opts CookieOptions, accessToken, refreshToken string, accessExpiresAt, refreshExpiresAt time.Time) {
	setCookie(w, opts, AccessTokenCookieName, accessToken, accessExpiresAt, "/")
	setCookie(w, opts, RefreshTokenCookieName, refreshToken, refreshExpiresAt, "/api/v1/auth/refresh")
}

// ClearAuthCookies clears access and refresh token cookies.
func ClearAuthCookies(w http.ResponseWriter, opts CookieOptions) {
	clearCookie(w, opts, AccessTokenCookieName, "/")
	clearCookie(w, opts, RefreshTokenCookieName, "/api/v1/auth/refresh")
}

// SetCSRFCookie writes the readable CSRF token cookie.
func (opts CookieOptions) SetCSRFCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	setCookie(w, opts, CSRFTokenCookieName, token, expiresAt, "/")
}

// SetCSRFCookieWithOptions is the standalone version for callers that prefer it.
func SetCSRFCookie(w http.ResponseWriter, opts CookieOptions, token string, expiresAt time.Time) {
	opts.SetCSRFCookie(w, token, expiresAt)
}

// ClearCSRFCookie clears the CSRF token cookie.
func ClearCSRFCookie(w http.ResponseWriter, opts CookieOptions) {
	clearCookie(w, opts, CSRFTokenCookieName, "/")
}

// SetGuestCookie writes the signed guest session cookie.
func SetGuestCookie(w http.ResponseWriter, opts CookieOptions, token string, expiresAt time.Time) {
	setCookie(w, opts, GuestSessionCookieName, token, expiresAt, "/")
}

// ClearGuestCookie clears the guest session cookie.
func ClearGuestCookie(w http.ResponseWriter, opts CookieOptions) {
	clearCookie(w, opts, GuestSessionCookieName, "/")
}

func setCookie(w http.ResponseWriter, opts CookieOptions, name, value string, expiresAt time.Time, path string) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   opts.Domain,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: name != CSRFTokenCookieName,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	}
	if name == CSRFTokenCookieName {
		cookie.HttpOnly = false
	}
	http.SetCookie(w, cookie)
}

func clearCookie(w http.ResponseWriter, opts CookieOptions, name, path string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   opts.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: name != CSRFTokenCookieName,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	}
	http.SetCookie(w, cookie)
}

// ReadCookieValue returns the value of the named cookie or an empty string.
func ReadCookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
