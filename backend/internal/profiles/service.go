package profiles

import (
	"context"
	"strings"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

// supportedLocales mirrors the frontend's supported locale set.
var supportedLocales = map[string]bool{
	"en": true,
	"ar": true,
}

const (
	displayNameMinLen = 2
	displayNameMaxLen = 32
	defaultHistoryLim = 20
	maxHistoryLimit   = 100
)

// store is the persistence contract the service depends on. *Repository
// satisfies this implicitly; tests can supply a fake.
type store interface {
	GetCurrentProfile(ctx context.Context, userID uuid.UUID) (*RegisteredProfile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, update ProfileUpdate) (*RegisteredProfile, error)
	GetPublicProfile(ctx context.Context, userID uuid.UUID) (*PublicProfileSummary, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*StatsSummary, error)
	ListGameHistory(ctx context.Context, userID uuid.UUID, limit int, cursor string) (*GameHistoryPage, error)
}

// Service implements profiles business logic.
type Service struct {
	repo    store
	metrics *Metrics
}

// NewService returns a new profiles service.
func NewService(repo store, metrics *Metrics) *Service {
	return &Service{repo: repo, metrics: metrics}
}

// GetCurrentProfile returns the profile, stats, and recent progress for the
// currently authenticated registered user.
func (s *Service) GetCurrentProfile(ctx context.Context, sess session.Context) (*ProfileResponse, error) {
	if !sess.IsRegistered() {
		return nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	profile, err := s.repo.GetCurrentProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}

	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	history, err := s.repo.ListGameHistory(ctx, userID, defaultHistoryLim, "")
	if err != nil {
		return nil, err
	}

	s.metrics.RecordProfileRead()

	return &ProfileResponse{
		Profile:  toProfileDTO(profile),
		Stats:    toStatsDTO(stats),
		Progress: ProgressDTO{RecentGames: toGameHistoryDTOs(history.Items), Page: PageDTO{Limit: history.Limit, NextCursor: history.NextCursor}},
	}, nil
}

// UpdateProfile validates and applies a partial update to the current
// registered user's profile, then returns the same full shape as
// GetCurrentProfile (profile, stats, and saved-progress summary).
func (s *Service) UpdateProfile(ctx context.Context, sess session.Context, req UpdateProfileRequest) (*ProfileResponse, error) {
	if !sess.IsRegistered() {
		return nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	update, verr := validateUpdate(req)
	if verr != nil {
		s.metrics.RecordValidationFailure()
		s.metrics.RecordProfileUpdate("validation_failed")
		return nil, verr
	}

	profile, err := s.repo.UpdateProfile(ctx, userID, update)
	if err != nil {
		s.metrics.RecordProfileUpdate("error")
		return nil, err
	}

	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	history, err := s.repo.ListGameHistory(ctx, userID, defaultHistoryLim, "")
	if err != nil {
		return nil, err
	}

	s.metrics.RecordProfileUpdate("success")

	return &ProfileResponse{
		Profile:  toProfileDTO(profile),
		Stats:    toStatsDTO(stats),
		Progress: ProgressDTO{RecentGames: toGameHistoryDTOs(history.Items), Page: PageDTO{Limit: history.Limit, NextCursor: history.NextCursor}},
	}, nil
}

// GetPublicProfile returns the public-safe profile and stats for any user.
// It returns ErrUserNotFound uniformly for missing, disabled, deleted, and
// otherwise unavailable users so callers cannot distinguish account state.
func (s *Service) GetPublicProfile(ctx context.Context, rawUserID string) (*PublicProfileResponse, error) {
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	profile, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}

	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.metrics.RecordPublicStatsRead()

	return &PublicProfileResponse{
		Profile: PublicProfileDTO{
			UserID:      profile.UserID,
			DisplayName: profile.DisplayName,
			AvatarURL:   profile.AvatarURL,
			CountryCode: profile.CountryCode,
		},
		Stats: toStatsDTO(stats),
	}, nil
}

// GetGameHistory returns a cursor-paginated, public-safe game history for a
// user. Same not-found semantics as GetPublicProfile.
func (s *Service) GetGameHistory(ctx context.Context, rawUserID string, limit int, cursor string) (*GameHistoryResponse, error) {
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	profile, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrUserNotFound
	}

	if limit <= 0 {
		limit = defaultHistoryLim
	}
	if limit > maxHistoryLimit {
		return nil, ErrInvalidLimit
	}

	page, err := s.repo.ListGameHistory(ctx, userID, limit, cursor)
	if err != nil {
		return nil, err
	}

	s.metrics.RecordGameHistoryRead()

	return &GameHistoryResponse{
		Games: toGameHistoryDTOs(page.Items),
		Page:  PageDTO{Limit: page.Limit, NextCursor: page.NextCursor},
	}, nil
}

func validateUpdate(req UpdateProfileRequest) (ProfileUpdate, error) {
	var fields []apphttp.FieldError
	update := ProfileUpdate{}

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if len(name) < displayNameMinLen || len(name) > displayNameMaxLen {
			fields = append(fields, apphttp.FieldError{
				Name:    "display_name",
				Code:    "invalid_length",
				Message: "display_name must be between 2 and 32 characters",
			})
		} else {
			update.HasDisplayName = true
			update.DisplayName = &name
		}
	}

	if req.AvatarURL.Set {
		update.HasAvatarURL = true
		update.AvatarURL = &req.AvatarURL.Value
	}

	if req.CountryCode.Set {
		if req.CountryCode.Value != nil && len(*req.CountryCode.Value) != 2 {
			fields = append(fields, apphttp.FieldError{
				Name:    "country_code",
				Code:    "invalid_format",
				Message: "country_code must be a 2-letter ISO code or null",
			})
		} else {
			update.HasCountryCode = true
			update.CountryCode = &req.CountryCode.Value
		}
	}

	if req.Locale != nil {
		if !supportedLocales[*req.Locale] {
			fields = append(fields, apphttp.FieldError{
				Name:    "locale",
				Code:    "unsupported_locale",
				Message: "locale must be one of: en, ar",
			})
		} else {
			update.HasLocale = true
			update.Locale = req.Locale
		}
	}

	if req.Timezone.Set {
		update.HasTimezone = true
		update.Timezone = &req.Timezone.Value
	}

	if req.Preferences.Set {
		update.HasPreferences = true
		prefs := req.Preferences.Value
		prefsPtr := &prefs
		update.Preferences = &prefsPtr
	}

	if len(fields) > 0 {
		return ProfileUpdate{}, &ValidationError{Fields: fields}
	}

	return update, nil
}

func toProfileDTO(p *RegisteredProfile) ProfileDTO {
	return ProfileDTO{
		UserID:      p.UserID,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarURL,
		CountryCode: p.CountryCode,
		Locale:      p.Locale,
		Timezone:    p.Timezone,
		Preferences: p.Preferences,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toStatsDTO(s *StatsSummary) StatsDTO {
	return StatsDTO{
		GamesPlayed:  s.GamesPlayed,
		TotalScore:   s.TotalScore,
		AverageScore: s.AverageScore,
		BestScore:    s.BestScore,
		LastPlayedAt: s.LastPlayedAt,
	}
}

func toGameHistoryDTOs(items []GameHistoryItem) []GameHistoryItemDTO {
	dtos := make([]GameHistoryItemDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, GameHistoryItemDTO{
			ID:                 it.GameID,
			MapID:              it.MapID,
			Mode:               it.Mode,
			Status:             it.Status,
			RoundCount:         it.RoundCount,
			CurrentRoundNumber: it.CurrentRoundNumber,
			TotalScore:         it.TotalScore,
			StartedAt:          it.StartedAt,
			CompletedAt:        it.CompletedAt,
			CreatedAt:          it.CreatedAt,
		})
	}
	return dtos
}
