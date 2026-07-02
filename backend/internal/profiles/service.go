package profiles

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

// supportedLocales mirrors the frontend's supported locale set.
var supportedLocales = map[string]bool{
	"en": true,
	"ar": true,
}

var allowedPreferenceValues = map[string]map[string]bool{
	"distance_unit": {"km": true, "mi": true},
	"theme":         {"system": true, "light": true, "dark": true},
}

const (
	displayNameMinLen = 2
	displayNameMaxLen = 32
	defaultHistoryLim = 20
	maxHistoryLimit   = 100

	iso3166Alpha2Codes = " AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW "
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
	logger  *slog.Logger
}

// NewService returns a new profiles service.
func NewService(repo store, metrics *Metrics) *Service {
	return NewServiceWithLogger(repo, metrics, nil)
}

// NewServiceWithLogger returns a profiles service with privacy-safe logging.
func NewServiceWithLogger(repo store, metrics *Metrics, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, metrics: metrics, logger: logger}
}

// GetCurrentProfile returns the profile, stats, and recent progress for the
// currently authenticated registered user.
func (s *Service) GetCurrentProfile(ctx context.Context, sess session.Context) (*ProfileResponse, error) {
	if !sess.IsRegistered() {
		s.logger.InfoContext(ctx, "profile read denied", slog.String("reason", "registered_session_required"))
		return nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(*sess.UserID)
	if err != nil {
		s.logger.InfoContext(ctx, "profile read denied", slog.String("reason", "invalid_session_user"))
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
	s.logger.InfoContext(ctx, "profile read completed", slog.String("user_id", userID.String()))

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
		s.logger.InfoContext(ctx, "profile update denied", slog.String("reason", "registered_session_required"))
		return nil, ErrUnauthorized
	}
	userID, err := uuid.Parse(*sess.UserID)
	if err != nil {
		s.logger.InfoContext(ctx, "profile update denied", slog.String("reason", "invalid_session_user"))
		return nil, ErrUnauthorized
	}

	update, err := validateUpdate(req)
	if err != nil {
		s.metrics.RecordValidationFailure()
		s.metrics.RecordProfileUpdate("validation_failed")
		var verr *ValidationError
		fieldCount := 0
		if errors.As(err, &verr) {
			fieldCount = len(verr.Fields)
		}
		s.logger.InfoContext(ctx, "profile update validation failed",
			slog.String("user_id", userID.String()),
			slog.Int("field_count", fieldCount),
		)
		return nil, err
	}

	profile, err := s.repo.UpdateProfile(ctx, userID, update)
	if err != nil {
		s.metrics.RecordProfileUpdate("error")
		s.logger.InfoContext(ctx, "profile update failed", slog.String("user_id", userID.String()), slog.String("outcome", "error"))
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
	s.logger.InfoContext(ctx, "profile update completed", slog.String("user_id", userID.String()), slog.String("outcome", "success"))

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
		s.logger.InfoContext(ctx, "public stats read rejected", slog.String("reason", "invalid_user_id"))
		return nil, ErrInvalidUserID
	}

	profile, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		s.logger.InfoContext(ctx, "public stats read not found", slog.String("user_id", userID.String()))
		return nil, ErrUserNotFound
	}

	stats, err := s.repo.GetStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.metrics.RecordPublicStatsRead()
	s.logger.InfoContext(ctx, "public stats read completed", slog.String("user_id", userID.String()))

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
		s.logger.InfoContext(ctx, "game history read rejected", slog.String("reason", "invalid_user_id"))
		return nil, ErrInvalidUserID
	}

	profile, err := s.repo.GetPublicProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		s.logger.InfoContext(ctx, "game history read not found", slog.String("user_id", userID.String()))
		return nil, ErrUserNotFound
	}

	if limit <= 0 {
		limit = defaultHistoryLim
	}
	if limit > maxHistoryLimit {
		s.logger.InfoContext(ctx, "game history read rejected", slog.String("user_id", userID.String()), slog.String("reason", "invalid_limit"))
		return nil, ErrInvalidLimit
	}

	page, err := s.repo.ListGameHistory(ctx, userID, limit, cursor)
	if err != nil {
		return nil, err
	}

	s.metrics.RecordGameHistoryRead()
	s.logger.InfoContext(ctx, "game history read completed", slog.String("user_id", userID.String()), slog.Int("limit", page.Limit))

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
		nameLen := len([]rune(name))
		if nameLen < displayNameMinLen || nameLen > displayNameMaxLen {
			fields = append(fields, apphttp.FieldError{
				Name:    "display_name",
				Code:    "invalid_length",
				Message: "display_name must be between 2 and 32 characters",
			})
		} else if containsControl(name) {
			fields = append(fields, apphttp.FieldError{
				Name:    "display_name",
				Code:    "invalid_format",
				Message: "display_name must not contain control characters",
			})
		} else {
			update.HasDisplayName = true
			update.DisplayName = &name
		}
	}

	if req.AvatarURL.Set {
		if req.AvatarURL.Value != nil {
			avatar := strings.TrimSpace(*req.AvatarURL.Value)
			if !isSafeAvatarURL(avatar) {
				fields = append(fields, apphttp.FieldError{
					Name:    "avatar_url",
					Code:    "invalid_format",
					Message: "avatar_url must be an http(s) image reference",
				})
			} else {
				update.HasAvatarURL = true
				update.AvatarURL = &req.AvatarURL.Value
				*update.AvatarURL = &avatar
			}
		} else {
			update.HasAvatarURL = true
			update.AvatarURL = &req.AvatarURL.Value
		}
	}

	if req.CountryCode.Set {
		if req.CountryCode.Value != nil {
			country := strings.ToUpper(strings.TrimSpace(*req.CountryCode.Value))
			if !isCountryCode(country) {
				fields = append(fields, apphttp.FieldError{
					Name:    "country_code",
					Code:    "invalid_format",
					Message: "country_code must be a 2-letter ISO code or null",
				})
			} else {
				update.HasCountryCode = true
				update.CountryCode = &req.CountryCode.Value
				*update.CountryCode = &country
			}
		} else {
			update.HasCountryCode = true
			update.CountryCode = &req.CountryCode.Value
		}
	}

	if req.Locale != nil {
		locale := strings.TrimSpace(*req.Locale)
		if !supportedLocales[locale] {
			fields = append(fields, apphttp.FieldError{
				Name:    "locale",
				Code:    "unsupported_locale",
				Message: "locale must be one of: en, ar",
			})
		} else {
			update.HasLocale = true
			update.Locale = &locale
		}
	}

	if req.Timezone.Set {
		if req.Timezone.Value != nil {
			tz := strings.TrimSpace(*req.Timezone.Value)
			if !isIanaTimezone(tz) {
				fields = append(fields, apphttp.FieldError{
					Name:    "timezone",
					Code:    "invalid_format",
					Message: "timezone must be a valid IANA timezone identifier or null",
				})
			} else {
				update.HasTimezone = true
				update.Timezone = &req.Timezone.Value
				*update.Timezone = &tz
			}
		} else {
			update.HasTimezone = true
			update.Timezone = &req.Timezone.Value
		}
	}

	if req.Preferences.Set {
		prefs := req.Preferences.Value
		if prefs != nil {
			if prefFields := validatePreferences(prefs); len(prefFields) > 0 {
				fields = append(fields, prefFields...)
			} else {
				update.HasPreferences = true
				prefsPtr := &prefs
				update.Preferences = &prefsPtr
			}
		} else {
			update.HasPreferences = true
			prefsPtr := &prefs
			update.Preferences = &prefsPtr
		}
	}

	if len(fields) > 0 {
		return ProfileUpdate{}, &ValidationError{Fields: fields}
	}

	return update, nil
}

func containsControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func isCountryCode(country string) bool {
	if len(country) != 2 {
		return false
	}
	return strings.Contains(iso3166Alpha2Codes, " "+country+" ")
}

func isIanaTimezone(tz string) bool {
	if tz == "" || containsControl(tz) {
		return false
	}
	if tz != "UTC" && !strings.Contains(tz, "/") {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}

func isSafeAvatarURL(raw string) bool {
	if raw == "" || len(raw) > 2048 || containsControl(raw) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return false
	}
	lowerPath := strings.ToLower(u.Path)
	return strings.HasSuffix(lowerPath, ".jpg") ||
		strings.HasSuffix(lowerPath, ".jpeg") ||
		strings.HasSuffix(lowerPath, ".png") ||
		strings.HasSuffix(lowerPath, ".webp") ||
		strings.HasSuffix(lowerPath, ".gif")
}

func validatePreferences(prefs map[string]any) []apphttp.FieldError {
	fields := make([]apphttp.FieldError, 0)
	for key, value := range prefs {
		allowed, ok := allowedPreferenceValues[key]
		if !ok {
			fields = append(fields, apphttp.FieldError{
				Name:    "preferences." + key,
				Code:    "unsupported_preference",
				Message: "preference is not supported",
			})
			continue
		}
		str, ok := value.(string)
		if !ok || !allowed[str] {
			fields = append(fields, apphttp.FieldError{
				Name:    "preferences." + key,
				Code:    "invalid_value",
				Message: "preference value is not supported",
			})
		}
	}
	return fields
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
