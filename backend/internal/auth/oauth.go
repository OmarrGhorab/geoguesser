package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/raven/geoguess/backend/internal/config"
)

// OAuthProvider identifies supported OAuth providers.
type OAuthProvider string

const (
	OAuthProviderGoogle  OAuthProvider = "google"
	OAuthProviderDiscord OAuthProvider = "discord"
)

// IsValidOAuthProvider returns true for supported providers.
func IsValidOAuthProvider(p string) bool {
	switch OAuthProvider(p) {
	case OAuthProviderGoogle, OAuthProviderDiscord:
		return true
	}
	return false
}

// OAuthUserInfo is the normalized profile returned by an OAuth provider.
type OAuthUserInfo struct {
	Provider          OAuthProvider
	ProviderAccountID string
	Email             *string
	DisplayName       *string
	AvatarURL         *string
	VerifiedEmail     bool
}

// OAuthClient handles the authorization-code flow for an OAuth provider.
type OAuthClient interface {
	AuthURL(state string) string
	ExchangeCode(ctxCode string) (*OAuthUserInfo, error)
}

// OAuthManager holds configured OAuth clients.
type OAuthManager struct {
	clients map[OAuthProvider]OAuthClient
}

// NewOAuthManager creates an OAuth manager from config. Missing provider
// configuration leaves that provider unavailable.
func NewOAuthManager(cfg config.Config) *OAuthManager {
	clients := make(map[OAuthProvider]OAuthClient)
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" && cfg.GoogleRedirectURL != "" {
		clients[OAuthProviderGoogle] = newGoogleClient(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	}
	if cfg.DiscordClientID != "" && cfg.DiscordClientSecret != "" && cfg.DiscordRedirectURL != "" {
		clients[OAuthProviderDiscord] = newDiscordClient(cfg.DiscordClientID, cfg.DiscordClientSecret, cfg.DiscordRedirectURL)
	}
	return &OAuthManager{clients: clients}
}

// Client returns the OAuth client for the provider, if configured.
func (o *OAuthManager) Client(provider OAuthProvider) (OAuthClient, bool) {
	c, ok := o.clients[provider]
	return c, ok
}

// GenerateOAuthState returns a random state token for OAuth flows.
func GenerateOAuthState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// googleClient implements OAuthClient for Google.
type googleClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

func newGoogleClient(clientID, clientSecret, redirectURL string) *googleClient {
	return &googleClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *googleClient) AuthURL(state string) string {
	u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	q := u.Query()
	q.Set("client_id", g.clientID)
	q.Set("redirect_uri", g.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("access_type", "offline")
	u.RawQuery = q.Encode()
	return u.String()
}

func (g *googleClient) ExchangeCode(code string) (*OAuthUserInfo, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", g.clientID)
	data.Set("client_secret", g.clientSecret)
	data.Set("redirect_uri", g.redirectURL)
	data.Set("grant_type", "authorization_code")

	resp, err := g.httpClient.Post("https://oauth2.googleapis.com/token", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("google token exchange failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read google token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token exchange returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode google token response: %w", err)
	}

	userReq, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build google userinfo request: %w", err)
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := g.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("google userinfo request failed: %w", err)
	}
	defer func() { _ = userResp.Body.Close() }()

	userBody, err := io.ReadAll(userResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read google userinfo response: %w", err)
	}
	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d: %s", userResp.StatusCode, string(userBody))
	}

	var user struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := json.Unmarshal(userBody, &user); err != nil {
		return nil, fmt.Errorf("failed to decode google userinfo: %w", err)
	}

	info := &OAuthUserInfo{
		Provider:          OAuthProviderGoogle,
		ProviderAccountID: user.ID,
		VerifiedEmail:     user.VerifiedEmail,
	}
	if user.Email != "" {
		info.Email = &user.Email
	}
	if user.Name != "" {
		info.DisplayName = &user.Name
	}
	if user.Picture != "" {
		info.AvatarURL = &user.Picture
	}
	return info, nil
}

// discordClient implements OAuthClient for Discord.
type discordClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

func newDiscordClient(clientID, clientSecret, redirectURL string) *discordClient {
	return &discordClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *discordClient) AuthURL(state string) string {
	u, _ := url.Parse("https://discord.com/oauth2/authorize")
	q := u.Query()
	q.Set("client_id", d.clientID)
	q.Set("redirect_uri", d.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "identify email")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String()
}

func (d *discordClient) ExchangeCode(code string) (*OAuthUserInfo, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", d.clientID)
	data.Set("client_secret", d.clientSecret)
	data.Set("redirect_uri", d.redirectURL)
	data.Set("grant_type", "authorization_code")

	resp, err := d.httpClient.Post("https://discord.com/api/oauth2/token", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("discord token exchange failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read discord token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord token exchange returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode discord token response: %w", err)
	}

	userReq, err := http.NewRequest(http.MethodGet, "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build discord user request: %w", err)
	}
	userReq.Header.Set("Authorization", tokenResp.TokenType+" "+tokenResp.AccessToken)

	userResp, err := d.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("discord user request failed: %w", err)
	}
	defer func() { _ = userResp.Body.Close() }()

	userBody, err := io.ReadAll(userResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read discord user response: %w", err)
	}
	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user returned status %d: %s", userResp.StatusCode, string(userBody))
	}

	var user struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		Verified   bool   `json:"verified"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
		Avatar     string `json:"avatar"`
	}
	if err := json.Unmarshal(userBody, &user); err != nil {
		return nil, fmt.Errorf("failed to decode discord user: %w", err)
	}

	displayName := user.GlobalName
	if displayName == "" {
		displayName = user.Username
	}

	info := &OAuthUserInfo{
		Provider:          OAuthProviderDiscord,
		ProviderAccountID: user.ID,
		VerifiedEmail:     user.Verified,
	}
	if user.Email != "" {
		info.Email = &user.Email
	}
	if displayName != "" {
		info.DisplayName = &displayName
	}
	if user.Avatar != "" {
		avatarURL := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
		info.AvatarURL = &avatarURL
	}
	return info, nil
}
