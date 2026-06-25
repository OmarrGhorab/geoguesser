package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/raven/geoguess/backend/internal/config"
	appmiddleware "github.com/raven/geoguess/backend/internal/middleware"
	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Handler handles authentication HTTP endpoints.
type Handler struct {
	service *Service
	cfg     config.Config
	opts    CookieOptions
	logger  *slog.Logger
}

// NewHandler returns a new auth handler.
func NewHandler(service *Service, cfg config.Config, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
		opts:    NewCookieOptions(cfg),
		logger:  logger,
	}
}

// Service returns the underlying auth service for middleware wiring.
func (h *Handler) Service() *Service {
	return h.service
}

// RegisterRoutes mounts auth routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(auth chi.Router) {
		auth.Post("/register", h.Register)
		auth.Post("/login", h.Login)
		auth.Post("/logout", h.Logout)
		auth.Post("/refresh", h.Refresh)
		auth.Post("/forgot-password", h.ForgotPassword)
		auth.Post("/reset-password", h.ResetPassword)
		auth.Get("/me", h.Me)
		auth.Get("/oauth/{provider}", h.OAuthInitiate)
		auth.Get("/oauth/{provider}/callback", h.OAuthCallback)
	})
}

// Register handles POST /auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	resp, tokens, err := h.service.Register(r.Context(), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	apphttp.Created(w, r, resp)
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	resp, tokens, err := h.service.Login(r.Context(), req)
	if err != nil {
		h.mapError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	apphttp.OK(w, r, resp)
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := ReadCookieValue(r, RefreshTokenCookieName)
	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		h.mapError(w, r, err)
		return
	}
	ClearAuthCookies(w, h.opts)
	ClearCSRFCookie(w, h.opts)
	apphttp.NoContent(w)
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := ReadCookieValue(r, RefreshTokenCookieName)
	resp, tokens, err := h.service.Refresh(r.Context(), refreshToken)
	if err != nil {
		ClearAuthCookies(w, h.opts)
		h.mapError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	apphttp.OK(w, r, resp)
}

// Me handles GET /auth/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	sc := appmiddleware.SessionFromContext(r.Context())
	resp, guestToken, err := h.service.Me(r.Context(), sc)
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	if guestToken != nil {
		SetGuestCookie(w, h.opts, guestToken.Raw, guestToken.ExpiresAt)
	}
	apphttp.OK(w, r, resp)
}

// ForgotPassword handles POST /auth/forgot-password.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	if err := h.service.RequestPasswordReset(r.Context(), req.Email); err != nil {
		h.mapError(w, r, err)
		return
	}

	apphttp.NoContent(w)
}

// ResetPassword handles POST /auth/reset-password.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := apphttp.DecodeJSON(w, r, &req); err != nil {
		apphttp.Error(w, r, h.logger, err)
		return
	}

	if err := h.service.ResetPassword(r.Context(), req); err != nil {
		h.mapError(w, r, err)
		return
	}

	apphttp.NoContent(w)
}

// OAuthInitiate handles GET /auth/oauth/{provider}.
func (h *Handler) OAuthInitiate(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !IsValidOAuthProvider(provider) {
		apphttp.Error(w, r, h.logger, ErrInvalidOAuthProvider)
		return
	}
	authURL, _, err := h.service.OAuthInitiate(r.Context(), OAuthProvider(provider))
	if err != nil {
		h.mapError(w, r, err)
		return
	}
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// OAuthCallback handles GET /auth/oauth/{provider}/callback.
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !IsValidOAuthProvider(provider) {
		apphttp.Error(w, r, h.logger, ErrInvalidOAuthProvider)
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		apphttp.Error(w, r, h.logger, ErrOAuthStateMismatch)
		return
	}

	resp, tokens, err := h.service.OAuthCallback(r.Context(), OAuthProvider(provider), code, state)
	if err != nil {
		h.mapError(w, r, err)
		return
	}

	h.setAuthCookies(w, tokens)
	apphttp.OK(w, r, resp)
}

func (h *Handler) setAuthCookies(w http.ResponseWriter, tokens *TokenPair) {
	SetAuthCookies(w, h.opts, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt, tokens.RefreshExpiresAt)
}

func (h *Handler) mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail),
		errors.Is(err, ErrPasswordTooShort),
		errors.Is(err, ErrDisplayNameLength),
		errors.Is(err, ErrPasswordRequired),
		errors.Is(err, ErrDisplayNameRequired):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	case errors.Is(err, ErrEmailAlreadyExists):
		apphttp.Error(w, r, h.logger, apphttp.ErrConflict.WithCause(err))
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrInvalidRefreshToken),
		errors.Is(err, ErrSessionExpired),
		errors.Is(err, ErrSessionRevoked),
		errors.Is(err, ErrTokenReuseDetected),
		errors.Is(err, ErrInvalidAccessToken),
		errors.Is(err, ErrUnauthorized):
		apphttp.Error(w, r, h.logger, apphttp.ErrUnauthorized.WithCause(err))
	case errors.Is(err, ErrInvalidCSRFToken):
		apphttp.Error(w, r, h.logger, apphttp.ErrForbidden.WithCause(err))
	case errors.Is(err, ErrInvalidOAuthProvider),
		errors.Is(err, ErrOAuthStateMismatch),
		errors.Is(err, ErrInvalidOTP),
		errors.Is(err, ErrOTPRateLimited),
		errors.Is(err, ErrSamePassword):
		apphttp.Error(w, r, h.logger, apphttp.ErrValidationFailed.WithCause(err))
	default:
		apphttp.Error(w, r, h.logger, err)
	}
}
