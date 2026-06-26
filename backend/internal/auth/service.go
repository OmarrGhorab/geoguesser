package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/config"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/platform/email"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"github.com/raven/geoguess/backend/internal/session"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	roleUser                  = "user"
	statusActive              = "active"
	statusPendingVerification = "pending_verification"
	oauthStatePrefix          = "oauth:state"
	oauthStateTTL             = 10 * time.Minute
)

// Service implements authentication business logic.
type Service struct {
	repo         *Repository
	hasher       PasswordHasher
	tokenManager *TokenManager
	guestManager *GuestSessionManager
	csrfManager  *CSRFManager
	oauthManager *OAuthManager
	sessionStore SessionStore
	otpStore     *OTPStore
	emailSender  email.Sender
	redis        *redis.Client
	cfg          config.Config
	clock        clock.Clock
}

// NewService returns a new auth service.
func NewService(
	repo *Repository,
	hasher PasswordHasher,
	tokenManager *TokenManager,
	guestManager *GuestSessionManager,
	csrfManager *CSRFManager,
	oauthManager *OAuthManager,
	sessionStore SessionStore,
	otpStore *OTPStore,
	emailSender email.Sender,
	redisClient *redis.Client,
	cfg config.Config,
	clock clock.Clock,
) *Service {
	return &Service{
		repo:         repo,
		hasher:       hasher,
		tokenManager: tokenManager,
		guestManager: guestManager,
		csrfManager:  csrfManager,
		oauthManager: oauthManager,
		sessionStore: sessionStore,
		otpStore:     otpStore,
		emailSender:  emailSender,
		redis:        redisClient,
		cfg:          cfg,
		clock:        clock,
	}
}

// Register creates a new registered account with email and password.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, *TokenPair, error) {
	if err := validateEmail(req.Email); err != nil {
		return nil, nil, ErrInvalidEmail
	}
	if len(req.Password) < 12 {
		return nil, nil, ErrPasswordTooShort
	}
	if len(req.DisplayName) < 2 || len(req.DisplayName) > 32 {
		return nil, nil, ErrDisplayNameLength
	}

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrEmailAlreadyExists
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, nil, err
	}

	now := s.clock.Now()
	user := &User{
		Email:        req.Email,
		PasswordHash: &hash,
		Role:         roleUser,
		Status:       statusPendingVerification,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	profile := &UserProfile{
		DisplayName: req.DisplayName,
		Locale:      "en",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var tokens *TokenPair
	if err := postgres.RunInTransaction(ctx, s.repo.db, func(tx *gorm.DB) error {
		if err := s.repo.CreateUser(ctx, tx, user); err != nil {
			return err
		}
		profile.UserID = user.ID
		if err := s.repo.CreateProfile(ctx, tx, profile); err != nil {
			return err
		}
		if err := s.repo.UpdateUserLastLogin(ctx, tx, user.ID, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	tokens, err = s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return &AuthResponse{User: toUserDTO(user, profile.DisplayName)}, tokens, nil
}

// Login authenticates a user with email and password.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, *TokenPair, error) {
	if err := validateEmail(req.Email); err != nil {
		return nil, nil, ErrInvalidEmail
	}
	if req.Password == "" {
		return nil, nil, ErrPasswordRequired
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || user.PasswordHash == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := s.hasher.Verify(req.Password, *user.PasswordHash); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	now := s.clock.Now()
	if err := s.repo.UpdateUserLastLogin(ctx, nil, user.ID, now); err != nil {
		return nil, nil, err
	}
	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return &AuthResponse{User: toUserDTO(user, profile.DisplayName)}, tokens, nil
}

// RequestPasswordReset generates an OTP and sends it to the user's email.
func (s *Service) RequestPasswordReset(ctx context.Context, emailAddress string) error {
	if err := validateEmail(emailAddress); err != nil {
		return ErrInvalidEmail
	}

	allowed, err := s.otpStore.AllowRequest(ctx, emailAddress)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrOTPRateLimited
	}

	user, err := s.repo.GetUserByEmail(ctx, emailAddress)
	if err != nil {
		return err
	}

	code, err := s.otpStore.Generate(ctx, emailAddress)
	if err != nil {
		return err
	}

	if user != nil {
		if err := s.emailSender.Send(ctx, email.Message{
			To:      user.Email,
			Subject: "Password reset code",
			Text:    fmt.Sprintf("Your password reset code is: %s", code),
		}); err != nil {
			return fmt.Errorf("failed to send password reset email: %w", err)
		}
	}

	return nil
}

// ResetPassword validates the OTP and updates the user's password.
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	if err := validateEmail(req.Email); err != nil {
		return ErrInvalidEmail
	}
	if len(req.NewPassword) < 12 {
		return ErrPasswordTooShort
	}
	if len(req.OTP) != otpLength {
		return ErrInvalidOTP
	}

	valid, err := s.otpStore.Validate(ctx, req.Email, req.OTP)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidOTP
	}

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrInvalidOTP
	}

	if user.PasswordHash != nil {
		if err := s.hasher.Verify(req.NewPassword, *user.PasswordHash); err == nil {
			return ErrSamePassword
		}
	}

	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}

	now := s.clock.Now()
	if err := postgres.RunInTransaction(ctx, s.repo.db, func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"password_hash": hash,
			"updated_at":    now,
		}).Error; err != nil {
			return fmt.Errorf("failed to update password: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := s.sessionStore.RevokeAll(ctx, user.ID); err != nil {
		return err
	}

	return nil
}

// Logout revokes the session identified by the refresh token hash.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := HashRefreshToken(refreshToken)
	return s.sessionStore.Revoke(ctx, hash)
}

// Refresh rotates the refresh token and issues a new access token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, *TokenPair, error) {
	hash := HashRefreshToken(refreshToken)

	if userID, reused, err := s.sessionStore.RevokedUserID(ctx, hash); err != nil {
		return nil, nil, err
	} else if reused {
		if err := s.sessionStore.RevokeAll(ctx, userID); err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrTokenReuseDetected
	}

	session, err := s.sessionStore.Get(ctx, hash)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrInvalidRefreshToken
	}

	now := s.clock.Now()
	if session.ExpiresAt.Before(now) {
		_ = s.sessionStore.Revoke(ctx, hash)
		return nil, nil, ErrSessionExpired
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrUserNotFound
	}

	profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// Rotate: revoke the current token and keep a short-lived revoked record
	// so that reuse attempts can be detected.
	if err := s.sessionStore.Revoke(ctx, hash); err != nil {
		return nil, nil, err
	}
	if err := s.markRevoked(ctx, hash, session.UserID); err != nil {
		return nil, nil, err
	}

	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return &AuthResponse{User: toUserDTO(user, profile.DisplayName)}, tokens, nil
}

func (s *Service) markRevoked(ctx context.Context, tokenHash string, userID uuid.UUID) error {
	// Keep revoked records for the same duration as refresh tokens so that
	// rotation chains remain detectable.
	return s.sessionStore.MarkRevoked(ctx, tokenHash, userID, s.cfg.RefreshTokenTTL)
}

// ResolveSession parses the access token and returns the session context.
func (s *Service) ResolveSession(ctx context.Context, accessToken string) (*session.Context, error) {
	if accessToken == "" {
		return &session.Context{Kind: session.KindAnonymous}, nil
	}
	claims, err := s.tokenManager.VerifyAccessToken(accessToken)
	if err != nil {
		return &session.Context{Kind: session.KindAnonymous}, nil
	}
	return &session.Context{
		Kind:   session.KindUser,
		UserID: &claims.UserID,
		Role:   claims.Role,
	}, nil
}

// ResolveGuestSession validates a guest session cookie and returns the guest id.
func (s *Service) ResolveGuestSession(ctx context.Context, signed string) (string, error) {
	if signed == "" {
		return "", ErrGuestSessionInvalid
	}
	raw, err := s.guestManager.Validate(signed)
	if err != nil {
		return "", err
	}
	return HashGuestID(raw), nil
}

// EnsureGuestSession returns an existing guest id or creates a new one.
func (s *Service) EnsureGuestSession(ctx context.Context, signed string) (string, *TokenPairGuest, error) {
	if signed != "" {
		raw, err := s.guestManager.Validate(signed)
		if err == nil {
			return HashGuestID(raw), nil, nil
		}
	}
	raw, hash, err := s.guestManager.Generate()
	if err != nil {
		return "", nil, err
	}
	return hash, &TokenPairGuest{Raw: raw, ExpiresAt: s.clock.Now().Add(GuestSessionTTL)}, nil
}

// TokenPairGuest holds a guest session token.
type TokenPairGuest struct {
	Raw       string
	ExpiresAt time.Time
}

// Me returns the session response for the resolved session context.
func (s *Service) Me(ctx context.Context, sc *session.Context) (*SessionResponse, *TokenPairGuest, error) {
	if sc == nil || sc.Kind == session.KindAnonymous {
		guestID, guestToken, err := s.EnsureGuestSession(ctx, "")
		if err != nil {
			return nil, nil, err
		}
		return &SessionResponse{
			Kind:          session.KindGuest,
			Authenticated: false,
			Guest:         &GuestDTO{ID: guestID, DisplayName: "Guest"},
		}, guestToken, nil
	}

	if sc.Kind == session.KindGuest && sc.GuestID != nil {
		return &SessionResponse{
			Kind:          session.KindGuest,
			Authenticated: false,
			Guest:         &GuestDTO{ID: *sc.GuestID, DisplayName: "Guest"},
		}, nil, nil
	}

	if sc.Kind == session.KindUser && sc.UserID != nil {
		userID, err := uuid.Parse(*sc.UserID)
		if err != nil {
			return nil, nil, ErrInvalidAccessToken
		}
		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return nil, nil, err
		}
		if user == nil {
			return nil, nil, ErrUserNotFound
		}
		profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
		if err != nil {
			return nil, nil, err
		}
		userDTO := toUserDTO(user, profile.DisplayName)
		return &SessionResponse{
			Kind:          session.KindUser,
			Authenticated: true,
			User:          &userDTO,
		}, nil, nil
	}

	return nil, nil, ErrUnauthorized
}

// GenerateCSRF returns a new CSRF token.
func (s *Service) GenerateCSRF() (string, error) {
	return s.csrfManager.Generate()
}

// ValidateCSRF validates a CSRF token.
func (s *Service) ValidateCSRF(token string) bool {
	return s.csrfManager.Validate(token)
}

// OAuthInitiate returns the provider authorization URL and stores the state in Redis.
func (s *Service) OAuthInitiate(ctx context.Context, provider OAuthProvider) (string, string, error) {
	client, ok := s.oauthManager.Client(provider)
	if !ok {
		return "", "", ErrInvalidOAuthProvider
	}
	state := GenerateOAuthState()
	if err := s.redis.Set(ctx, fmt.Sprintf("%s:%s", oauthStatePrefix, state), string(provider), oauthStateTTL).Err(); err != nil {
		return "", "", fmt.Errorf("failed to store oauth state: %w", err)
	}
	return client.AuthURL(state), state, nil
}

// OAuthCallback completes the OAuth flow and returns an authenticated session.
func (s *Service) OAuthCallback(ctx context.Context, provider OAuthProvider, code, state string) (*AuthResponse, *TokenPair, error) {
	stored, err := s.redis.Get(ctx, fmt.Sprintf("%s:%s", oauthStatePrefix, state)).Result()
	if errors.Is(err, redis.Nil) || stored != string(provider) {
		return nil, nil, ErrOAuthStateMismatch
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read oauth state: %w", err)
	}
	_ = s.redis.Del(ctx, fmt.Sprintf("%s:%s", oauthStatePrefix, state))

	client, ok := s.oauthManager.Client(provider)
	if !ok {
		return nil, nil, ErrInvalidOAuthProvider
	}

	info, err := client.ExchangeCode(code)
	if err != nil {
		return nil, nil, err
	}

	existing, err := s.repo.GetOAuthAccount(ctx, provider, info.ProviderAccountID)
	if err != nil {
		return nil, nil, err
	}

	now := s.clock.Now()
	if existing != nil {
		user, err := s.repo.GetUserByID(ctx, existing.UserID)
		if err != nil {
			return nil, nil, err
		}
		if user == nil {
			return nil, nil, ErrUserNotFound
		}
		profile, err := s.repo.GetProfileByUserID(ctx, user.ID)
		if err != nil {
			return nil, nil, err
		}
		var tokens *TokenPair
		if err := s.repo.UpdateUserLastLogin(ctx, nil, user.ID, now); err != nil {
			return nil, nil, err
		}
		tokens, err = s.createSession(ctx, user)
		if err != nil {
			return nil, nil, err
		}
		return &AuthResponse{User: toUserDTO(user, profile.DisplayName)}, tokens, nil
	}

	displayName := ""
	if info.DisplayName != nil {
		displayName = *info.DisplayName
	}
	if displayName == "" {
		displayName = "Player"
	}
	if info.Email == nil || *info.Email == "" {
		return nil, nil, ErrInvalidEmail
	}
	email := *info.Email

	var user *User
	var profile *UserProfile
	var tokens *TokenPair
	if err := postgres.RunInTransaction(ctx, s.repo.db, func(tx *gorm.DB) error {
		existingUser, err := s.repo.GetUserByEmail(ctx, email)
		if err != nil {
			return err
		}
		if existingUser != nil {
			user = existingUser
		} else {
			user = &User{
				Email:  email,
				Role:   roleUser,
				Status: statusActive,
			}
			if err := s.repo.CreateUser(ctx, tx, user); err != nil {
				return err
			}
			profile = &UserProfile{
				UserID:      user.ID,
				DisplayName: displayName,
				Locale:      "en",
			}
			if err := s.repo.CreateProfile(ctx, tx, profile); err != nil {
				return err
			}
		}

		if profile == nil {
			p, err := s.repo.GetProfileByUserID(ctx, user.ID)
			if err != nil {
				return err
			}
			profile = p
		}

		account := &UserOAuthAccount{
			UserID:            user.ID,
			Provider:          string(provider),
			ProviderAccountID: info.ProviderAccountID,
		}
		if info.Email != nil {
			account.Email = info.Email
		}
		if info.DisplayName != nil {
			account.DisplayName = info.DisplayName
		}
		if info.AvatarURL != nil {
			account.AvatarURL = info.AvatarURL
		}
		if err := s.repo.CreateOAuthAccount(ctx, tx, account); err != nil {
			return err
		}

		if err := s.repo.UpdateUserLastLogin(ctx, tx, user.ID, now); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, nil, err
	}

	tokens, err = s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return &AuthResponse{User: toUserDTO(user, profile.DisplayName)}, tokens, nil
}

func (s *Service) createSession(ctx context.Context, user *User) (*TokenPair, error) {
	now := s.clock.Now()
	rawRefresh, hash, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	accessToken, accessExpiresAt, err := s.tokenManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := now.Add(s.cfg.RefreshTokenTTL)

	session := &RefreshSession{
		UserID:    user.ID,
		Role:      user.Role,
		CreatedAt: now,
		ExpiresAt: refreshExpiresAt,
	}
	if err := s.sessionStore.Create(ctx, hash, session, s.cfg.RefreshTokenTTL); err != nil {
		return nil, err
	}

	tokens := &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     rawRefresh,
		ExpiresAt:        accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}
	return tokens, nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrEmailRequired
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func toUserDTO(user *User, displayName string) UserDTO {
	return UserDTO{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: displayName,
		Role:        user.Role,
	}
}
