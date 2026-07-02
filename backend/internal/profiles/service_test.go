package profiles

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/session"
)

// fakeStore is an in-memory stand-in for *Repository used in service unit
// tests. Only the behavior each test exercises is implemented; everything
// else returns zero values.
type fakeStore struct {
	profiles map[uuid.UUID]*RegisteredProfile
	public   map[uuid.UUID]*PublicProfileSummary
	stats    map[uuid.UUID]*StatsSummary
	history  map[uuid.UUID]*GameHistoryPage

	updateErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		profiles: map[uuid.UUID]*RegisteredProfile{},
		public:   map[uuid.UUID]*PublicProfileSummary{},
		stats:    map[uuid.UUID]*StatsSummary{},
		history:  map[uuid.UUID]*GameHistoryPage{},
	}
}

func (f *fakeStore) GetCurrentProfile(_ context.Context, userID uuid.UUID) (*RegisteredProfile, error) {
	return f.profiles[userID], nil
}

func (f *fakeStore) UpdateProfile(_ context.Context, userID uuid.UUID, update ProfileUpdate) (*RegisteredProfile, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	p, ok := f.profiles[userID]
	if !ok {
		return nil, ErrProfileNotFound
	}
	cpy := *p
	if update.HasDisplayName && update.DisplayName != nil {
		cpy.DisplayName = *update.DisplayName
	}
	if update.HasAvatarURL {
		cpy.AvatarURL = *update.AvatarURL
	}
	if update.HasCountryCode {
		cpy.CountryCode = *update.CountryCode
	}
	if update.HasLocale && update.Locale != nil {
		cpy.Locale = *update.Locale
	}
	if update.HasTimezone {
		cpy.Timezone = *update.Timezone
	}
	if update.HasPreferences {
		if *update.Preferences == nil {
			cpy.Preferences = map[string]any{}
		} else {
			cpy.Preferences = **update.Preferences
		}
	}
	f.profiles[userID] = &cpy
	return &cpy, nil
}

func (f *fakeStore) GetPublicProfile(_ context.Context, userID uuid.UUID) (*PublicProfileSummary, error) {
	return f.public[userID], nil
}

func (f *fakeStore) GetStats(_ context.Context, userID uuid.UUID) (*StatsSummary, error) {
	if s, ok := f.stats[userID]; ok {
		return s, nil
	}
	return &StatsSummary{}, nil
}

func (f *fakeStore) ListGameHistory(_ context.Context, userID uuid.UUID, limit int, _ string) (*GameHistoryPage, error) {
	if p, ok := f.history[userID]; ok {
		return p, nil
	}
	return &GameHistoryPage{Limit: limit}, nil
}

func registeredSession(userID uuid.UUID) session.Context {
	id := userID.String()
	return session.Context{Kind: session.KindUser, UserID: &id}
}

func TestServiceGetCurrentProfileRequiresRegisteredSession(t *testing.T) {
	s := NewService(newFakeStore(), nil)
	_, err := s.GetCurrentProfile(context.Background(), session.Context{Kind: session.KindGuest})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestServiceGetCurrentProfileNotFound(t *testing.T) {
	s := NewService(newFakeStore(), nil)
	userID := uuid.New()
	_, err := s.GetCurrentProfile(context.Background(), registeredSession(userID))
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestServiceGetCurrentProfileReturnsProfileStatsAndProgress(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{
		UserID:      userID,
		Email:       "a@example.com",
		DisplayName: "Ana",
		Locale:      "en",
		Preferences: map[string]any{},
	}
	fs.stats[userID] = &StatsSummary{GamesPlayed: 3, TotalScore: 300, AverageScore: 100, BestScore: 150}
	fs.history[userID] = &GameHistoryPage{
		Items: []GameHistoryItem{{GameID: uuid.New(), Mode: "solo", Status: "completed", CreatedAt: time.Now()}},
		Limit: 20,
	}

	s := NewService(fs, nil)
	resp, err := s.GetCurrentProfile(context.Background(), registeredSession(userID))
	if err != nil {
		t.Fatalf("GetCurrentProfile failed: %v", err)
	}
	if resp.Profile.Email != "a@example.com" {
		t.Fatalf("expected email to be included for owner, got %q", resp.Profile.Email)
	}
	if resp.Stats.GamesPlayed != 3 {
		t.Fatalf("expected games played 3, got %d", resp.Stats.GamesPlayed)
	}
	if len(resp.Progress.RecentGames) != 1 {
		t.Fatalf("expected 1 recent game, got %d", len(resp.Progress.RecentGames))
	}
}

func TestServiceLogsAvoidEmailAndPreferences(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{
		UserID:      userID,
		Email:       "private@example.com",
		DisplayName: "Ana",
		Locale:      "en",
		Preferences: map[string]any{"distance_unit": "km"},
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	s := NewServiceWithLogger(fs, nil, logger)
	_, err := s.GetCurrentProfile(context.Background(), registeredSession(userID))
	if err != nil {
		t.Fatalf("GetCurrentProfile failed: %v", err)
	}

	got := logs.String()
	if strings.Contains(got, "private@example.com") || strings.Contains(got, "distance_unit") {
		t.Fatalf("privacy-sensitive profile data appeared in logs: %s", got)
	}
	if !strings.Contains(got, userID.String()) {
		t.Fatalf("expected safe user id in logs, got %s", got)
	}
}

func TestServiceUpdateProfileValidatesDisplayNameLength(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	tooShort := "a"
	_, err := s.UpdateProfile(context.Background(), registeredSession(userID), UpdateProfileRequest{DisplayName: &tooShort})

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %v (%T)", err, err)
	}
	if len(verr.Fields) != 1 || verr.Fields[0].Name != "display_name" {
		t.Fatalf("expected display_name field error, got %+v", verr.Fields)
	}
}

func TestServiceUpdateProfileValidatesDisplayNameControlCharacters(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	bad := "Ana\nNour"
	_, err := s.UpdateProfile(context.Background(), registeredSession(userID), UpdateProfileRequest{DisplayName: &bad})

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %v (%T)", err, err)
	}
	if verr.Fields[0].Name != "display_name" || verr.Fields[0].Code != "invalid_format" {
		t.Fatalf("expected display_name invalid_format error, got %+v", verr.Fields)
	}
}

func TestServiceUpdateProfileValidatesLocale(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	bad := "fr"
	_, err := s.UpdateProfile(context.Background(), registeredSession(userID), UpdateProfileRequest{Locale: &bad})

	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %v (%T)", err, err)
	}
	if verr.Fields[0].Name != "locale" {
		t.Fatalf("expected locale field error, got %+v", verr.Fields)
	}
}

func TestServiceUpdateProfileValidatesCountryCode(t *testing.T) {
	for _, bad := range []string{"EGY", "ZZ"} {
		t.Run(bad, func(t *testing.T) {
			fs := newFakeStore()
			userID := uuid.New()
			fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

			s := NewService(fs, nil)
			req := UpdateProfileRequest{}
			if err := req.CountryCode.UnmarshalJSON([]byte(`"` + bad + `"`)); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			_, err := s.UpdateProfile(context.Background(), registeredSession(userID), req)

			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *ValidationError, got %v (%T)", err, err)
			}
			if verr.Fields[0].Name != "country_code" {
				t.Fatalf("expected country_code field error, got %+v", verr.Fields)
			}
		})
	}
}

func TestServiceUpdateProfileNormalizesCountryCode(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	req := UpdateProfileRequest{}
	if err := req.CountryCode.UnmarshalJSON([]byte(`"eg"`)); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	resp, err := s.UpdateProfile(context.Background(), registeredSession(userID), req)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if resp.Profile.CountryCode == nil || *resp.Profile.CountryCode != "EG" {
		t.Fatalf("expected country EG, got %v", resp.Profile.CountryCode)
	}
}

func TestServiceUpdateProfileValidatesTimezoneAvatarAndPreferences(t *testing.T) {
	tests := []struct {
		name      string
		request   func() UpdateProfileRequest
		wantField string
	}{
		{
			name: "timezone_unknown",
			request: func() UpdateProfileRequest {
				req := UpdateProfileRequest{}
				if err := req.Timezone.UnmarshalJSON([]byte(`"Not/AZone"`)); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				return req
			},
			wantField: "timezone",
		},
		{
			name: "timezone_local",
			request: func() UpdateProfileRequest {
				req := UpdateProfileRequest{}
				if err := req.Timezone.UnmarshalJSON([]byte(`"Local"`)); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				return req
			},
			wantField: "timezone",
		},
		{
			name: "avatar",
			request: func() UpdateProfileRequest {
				req := UpdateProfileRequest{}
				if err := req.AvatarURL.UnmarshalJSON([]byte(`"javascript:alert(1)"`)); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				return req
			},
			wantField: "avatar_url",
		},
		{
			name: "preferences",
			request: func() UpdateProfileRequest {
				req := UpdateProfileRequest{}
				if err := req.Preferences.UnmarshalJSON([]byte(`{"admin":true}`)); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				return req
			},
			wantField: "preferences.admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := newFakeStore()
			userID := uuid.New()
			fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

			s := NewService(fs, nil)
			_, err := s.UpdateProfile(context.Background(), registeredSession(userID), tt.request())

			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *ValidationError, got %v (%T)", err, err)
			}
			if verr.Fields[0].Name != tt.wantField {
				t.Fatalf("expected %s field error, got %+v", tt.wantField, verr.Fields)
			}
		})
	}
}

func TestServiceUpdateProfileAcceptsSafeTimezoneAvatarAndPreferences(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	req := UpdateProfileRequest{}
	if err := req.Timezone.UnmarshalJSON([]byte(`"Africa/Cairo"`)); err != nil {
		t.Fatalf("unmarshal timezone failed: %v", err)
	}
	if err := req.AvatarURL.UnmarshalJSON([]byte(`"https://example.com/avatar.webp"`)); err != nil {
		t.Fatalf("unmarshal avatar failed: %v", err)
	}
	if err := req.Preferences.UnmarshalJSON([]byte(`{"distance_unit":"km","theme":"dark"}`)); err != nil {
		t.Fatalf("unmarshal preferences failed: %v", err)
	}

	resp, err := s.UpdateProfile(context.Background(), registeredSession(userID), req)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if resp.Profile.Timezone == nil || *resp.Profile.Timezone != "Africa/Cairo" {
		t.Fatalf("expected timezone Africa/Cairo, got %v", resp.Profile.Timezone)
	}
	if resp.Profile.AvatarURL == nil || *resp.Profile.AvatarURL != "https://example.com/avatar.webp" {
		t.Fatalf("expected avatar URL to persist, got %v", resp.Profile.AvatarURL)
	}
	if resp.Profile.Preferences["distance_unit"] != "km" || resp.Profile.Preferences["theme"] != "dark" {
		t.Fatalf("expected safe preferences to persist, got %+v", resp.Profile.Preferences)
	}
}

func TestServiceUpdateProfileAcceptsUTCTimezone(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{}}

	s := NewService(fs, nil)
	req := UpdateProfileRequest{}
	if err := req.Timezone.UnmarshalJSON([]byte(`"UTC"`)); err != nil {
		t.Fatalf("unmarshal timezone failed: %v", err)
	}

	resp, err := s.UpdateProfile(context.Background(), registeredSession(userID), req)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if resp.Profile.Timezone == nil || *resp.Profile.Timezone != "UTC" {
		t.Fatalf("expected timezone UTC, got %v", resp.Profile.Timezone)
	}
}

func TestServiceUpdateProfileAppliesPartialUpdate(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", Preferences: map[string]any{"a": 1}}

	s := NewService(fs, nil)
	newName := "Nour"
	resp, err := s.UpdateProfile(context.Background(), registeredSession(userID), UpdateProfileRequest{DisplayName: &newName})
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if resp.Profile.DisplayName != newName {
		t.Fatalf("expected display name %q, got %q", newName, resp.Profile.DisplayName)
	}
	if resp.Profile.Locale != "en" {
		t.Fatalf("expected locale to remain unchanged (en), got %q", resp.Profile.Locale)
	}
}

func TestServiceUpdateProfileClearsOptionalFieldOnExplicitNull(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	avatar := "https://example.com/a.png"
	fs.profiles[userID] = &RegisteredProfile{UserID: userID, DisplayName: "Ana", Locale: "en", AvatarURL: &avatar, Preferences: map[string]any{}}

	s := NewService(fs, nil)
	req := UpdateProfileRequest{}
	if err := req.AvatarURL.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	resp, err := s.UpdateProfile(context.Background(), registeredSession(userID), req)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}
	if resp.Profile.AvatarURL != nil {
		t.Fatalf("expected avatar_url to be cleared, got %v", *resp.Profile.AvatarURL)
	}
}

func TestServiceGetPublicProfileUniformNotFound(t *testing.T) {
	s := NewService(newFakeStore(), nil)
	_, err := s.GetPublicProfile(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestServiceGetPublicProfileInvalidID(t *testing.T) {
	s := NewService(newFakeStore(), nil)
	_, err := s.GetPublicProfile(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestServiceGetPublicProfileReturnsSummaryAndStats(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.public[userID] = &PublicProfileSummary{UserID: userID, DisplayName: "Ana"}
	fs.stats[userID] = &StatsSummary{GamesPlayed: 5}

	s := NewService(fs, nil)
	resp, err := s.GetPublicProfile(context.Background(), userID.String())
	if err != nil {
		t.Fatalf("GetPublicProfile failed: %v", err)
	}
	if resp.Profile.DisplayName != "Ana" {
		t.Fatalf("expected display name Ana, got %q", resp.Profile.DisplayName)
	}
	if resp.Stats.GamesPlayed != 5 {
		t.Fatalf("expected games played 5, got %d", resp.Stats.GamesPlayed)
	}
}

func TestServiceGetGameHistoryRejectsOversizedLimit(t *testing.T) {
	fs := newFakeStore()
	userID := uuid.New()
	fs.public[userID] = &PublicProfileSummary{UserID: userID}

	s := NewService(fs, nil)
	_, err := s.GetGameHistory(context.Background(), userID.String(), 500, "")
	if !errors.Is(err, ErrInvalidLimit) {
		t.Fatalf("expected ErrInvalidLimit, got %v", err)
	}
}

func TestServiceGetGameHistoryNotFoundForMissingUser(t *testing.T) {
	s := NewService(newFakeStore(), nil)
	_, err := s.GetGameHistory(context.Background(), uuid.New().String(), 10, "")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
