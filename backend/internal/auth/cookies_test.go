package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSetAuthCookiesAreHTTPOnly(t *testing.T) {
	opts := CookieOptions{Secure: true, SameSite: http.SameSiteLaxMode}
	w := httptest.NewRecorder()
	expiresAt := time.Now().UTC().Add(time.Hour)

	SetAuthCookies(w, opts, "access", "refresh", expiresAt, expiresAt)

	for _, name := range []string{AccessTokenCookieName, RefreshTokenCookieName} {
		cookie := findCookie(t, w.Result(), name)
		if !cookie.HttpOnly {
			t.Fatalf("%s cookie must be HttpOnly", name)
		}
		if !cookie.Secure {
			t.Fatalf("%s cookie must preserve Secure option", name)
		}
	}
}

func TestSetGuestCookieIsHTTPOnly(t *testing.T) {
	opts := CookieOptions{SameSite: http.SameSiteLaxMode}
	w := httptest.NewRecorder()

	SetGuestCookie(w, opts, "guest", time.Now().UTC().Add(time.Hour))

	cookie := findCookie(t, w.Result(), GuestSessionCookieName)
	if !cookie.HttpOnly {
		t.Fatal("guest session cookie must be HttpOnly")
	}
}

func TestSetCSRFCookieIsReadable(t *testing.T) {
	opts := CookieOptions{SameSite: http.SameSiteLaxMode}
	w := httptest.NewRecorder()

	SetCSRFCookie(w, opts, "csrf", time.Now().UTC().Add(time.Hour))

	cookie := findCookie(t, w.Result(), CSRFTokenCookieName)
	if cookie.HttpOnly {
		t.Fatal("csrf cookie must be readable by browser JavaScript")
	}
}

func findCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s not found", name)
	return nil
}
