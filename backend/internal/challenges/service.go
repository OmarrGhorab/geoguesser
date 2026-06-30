package challenges

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/clock"
	"github.com/raven/geoguess/backend/internal/session"
)

type LocationSelector interface {
	SelectLocations(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error)
}

type MetricsRecorder interface{}

type Service struct {
	repo         *Repository
	selector     LocationSelector
	clock        clock.Clock
	logger       *slog.Logger
	resetHourUTC int
	defaultMapID uuid.UUID
	metrics      MetricsRecorder
}

func NewService(repo *Repository, selector LocationSelector, clk clock.Clock, logger *slog.Logger, resetHourUTC int, defaultMapID uuid.UUID, metrics MetricsRecorder) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, selector: selector, clock: clk, logger: logger, resetHourUTC: resetHourUTC, defaultMapID: defaultMapID, metrics: metrics}
}

func (s *Service) GetDaily(ctx context.Context, sess *session.Context, dateOverride string) (*ChallengeMetadataResponse, error) {
	now := s.clock.Now()
	date, startsAt, endsAt := DailyWindow(now, s.resetHourUTC)
	if strings.TrimSpace(dateOverride) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(dateOverride))
		if err != nil {
			return nil, ErrInvalidChallengeInput
		}
		date = parsed
		startsAt = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), s.resetHourUTC, 0, 0, 0, time.UTC)
		endsAt = startsAt.AddDate(0, 0, 1)
	}
	challenge, err := s.repo.GetDailyByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		challenge, err = s.materializeDaily(ctx, date, startsAt, endsAt)
		if err != nil {
			return nil, err
		}
	}
	return s.metadata(ctx, sess, *challenge)
}

func (s *Service) StartDailyAttempt(ctx context.Context, sess *session.Context) (*ChallengeAttemptResponse, error) {
	now := s.clock.Now()
	date, _, _ := DailyWindow(now, s.resetHourUTC)
	challenge, err := s.repo.GetDailyByDate(ctx, date)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		meta, err := s.GetDaily(ctx, sess, "")
		if err != nil {
			return nil, err
		}
		challenge, err = s.repo.GetChallengeByID(ctx, meta.Challenge.ID)
		if err != nil {
			return nil, err
		}
	}
	if challenge == nil {
		return nil, ErrChallengeNotFound
	}
	return s.startAttempt(ctx, sess, *challenge)
}

func (s *Service) CreateShared(ctx context.Context, sess *session.Context, req CreateSharedChallengeRequest) (*ChallengeMetadataResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	settings, err := normalizeSettings(req.RoundCount, req.TimerSeconds)
	if err != nil {
		return nil, err
	}
	if req.MapID == uuid.Nil {
		return nil, ErrInvalidChallengeInput
	}
	selected, err := s.selectUnique(ctx, req.MapID, settings.RoundCount)
	if err != nil {
		return nil, err
	}
	seed, err := SharedSeed()
	if err != nil {
		return nil, err
	}
	code, err := SharedCode()
	if err != nil {
		return nil, err
	}
	rawSettings, err := encodeSettings(settings)
	if err != nil {
		return nil, err
	}
	challenge := &Challenge{
		Type:             TypeShared,
		SlugOrCode:       &code,
		Seed:             seed,
		MapID:            req.MapID,
		SettingsSnapshot: rawSettings,
		Status:           StatusActive,
		CreatedByUserID:  owner.userID,
	}
	locations := challengeLocationsFromSelected(selected)
	if err := s.repo.CreateChallengeWithLocations(ctx, challenge, locations); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "shared challenge created", slog.String("challenge_id", challenge.ID.String()), slog.String("map_id", req.MapID.String()))
	return s.metadata(ctx, sess, *challenge)
}

func (s *Service) GetShared(ctx context.Context, sess *session.Context, code string) (*ChallengeMetadataResponse, error) {
	challenge, err := s.repo.GetSharedByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, ErrChallengeNotFound
	}
	return s.metadata(ctx, sess, *challenge)
}

func (s *Service) StartChallengeAttempt(ctx context.Context, sess *session.Context, challengeID string) (*ChallengeAttemptResponse, error) {
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	challenge, err := s.repo.GetChallengeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, ErrChallengeNotFound
	}
	return s.startAttempt(ctx, sess, *challenge)
}

func (s *Service) GetResults(ctx context.Context, sess *session.Context, challengeID string) (*ResultResponse, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	challenge, err := s.repo.GetChallengeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, ErrChallengeNotFound
	}
	attempt, err := s.repo.GetAttemptForOwner(ctx, id, owner)
	if err != nil {
		return nil, err
	}
	if attempt == nil {
		return nil, ErrResultsNotReady
	}
	if attempt.Status != AttemptStatusCompleted {
		s.logger.InfoContext(ctx, "challenge result spoiler protected", slog.String("challenge_id", id.String()), slog.String("attempt_id", attempt.ID.String()))
	}
	settings, err := decodeSettings(challenge.SettingsSnapshot)
	if err != nil {
		return nil, err
	}
	return &ResultResponse{
		Challenge: toChallengeSummary(*challenge, settings),
		Attempt:   toAttemptSummary(*attempt, nil),
		Visible:   attempt.Status == AttemptStatusCompleted,
		Message:   "Results are available after completing the challenge.",
	}, nil
}

func (s *Service) GetLeaderboard(ctx context.Context, sess *session.Context, challengeID string, limit int) (*LeaderboardResponse, error) {
	id, err := uuid.Parse(challengeID)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	challenge, err := s.repo.GetChallengeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		return nil, ErrChallengeNotFound
	}
	settings, err := decodeSettings(challenge.SettingsSnapshot)
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListLeaderboardEntries(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	dtos := make([]LeaderboardEntryDTO, len(entries))
	for i, entry := range entries {
		dtos[i] = LeaderboardEntryDTO{Rank: entry.Rank, DisplayName: entry.DisplayNameSnapshot, Score: entry.Score, CompletionDurationMS: entry.CompletionDurationMS, CompletedAt: entry.CompletedAt}
	}
	s.logger.InfoContext(ctx, "challenge leaderboard read", slog.String("challenge_id", id.String()), slog.Int("entries", len(entries)))
	return &LeaderboardResponse{Challenge: toChallengeSummary(*challenge, settings), Entries: dtos, Page: PageInfo{Limit: limit}}, nil
}

func (s *Service) GetDailyStreak(ctx context.Context, sess *session.Context) (*StreakSummary, error) {
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	streak, err := s.repo.GetStreakForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	summary := toStreakSummary(streak, owner.guestHash != nil)
	return &summary, nil
}

func (s *Service) GetMissions(ctx context.Context, sess *session.Context) ([]MissionSummary, error) {
	if _, err := ownerFromSession(sess); err != nil {
		return nil, err
	}
	return DefaultMissionSummaries(s.clock.Now()), nil
}

func (s *Service) ClaimMission(ctx context.Context, sess *session.Context, missionID string) (*MissionSummary, error) {
	if _, err := ownerFromSession(sess); err != nil {
		return nil, err
	}
	missions := DefaultMissionSummaries(s.clock.Now())
	if len(missions) == 0 {
		return nil, ErrChallengeNotFound
	}
	missions[0].Status = "claimed"
	s.logger.InfoContext(ctx, "challenge mission claimed", slog.String("mission_id", missionID))
	return &missions[0], nil
}

func (s *Service) FinalizeAttemptResult(ctx context.Context, attempt ChallengeAttempt, roundResults any, completedAt time.Time, displayName string) error {
	if attempt.Status != AttemptStatusCompleted {
		attempt.Status = AttemptStatusCompleted
	}
	snapshot, err := json.Marshal(roundResults)
	if err != nil {
		return err
	}
	result := ChallengeResult{
		AttemptID:            attempt.ID,
		ChallengeID:          attempt.ChallengeID,
		TotalScore:           attempt.TotalScore,
		TotalDistanceMeters:  attempt.TotalDistanceMeters,
		RoundResultsSnapshot: snapshot,
		CompletedAt:          completedAt,
	}
	if err := s.repo.CreateResultAndLeaderboardEntry(ctx, attempt, result, displayName); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "challenge result finalized", slog.String("challenge_id", attempt.ChallengeID.String()), slog.String("attempt_id", attempt.ID.String()), slog.Int("score", result.TotalScore))
	return nil
}

func (s *Service) materializeDaily(ctx context.Context, date, startsAt, endsAt time.Time) (*Challenge, error) {
	mapID, err := s.firstActiveMapID(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := normalizeSettings(0, nil)
	if err != nil {
		return nil, err
	}
	selected, err := s.selectUnique(ctx, mapID, settings.RoundCount)
	if err != nil {
		return nil, err
	}
	rawSettings, err := encodeSettings(settings)
	if err != nil {
		return nil, err
	}
	challenge := &Challenge{
		Type:             TypeDaily,
		Seed:             DailySeed(date),
		ChallengeDate:    &date,
		ResetStartsAt:    &startsAt,
		ResetEndsAt:      &endsAt,
		MapID:            mapID,
		SettingsSnapshot: rawSettings,
		Status:           StatusActive,
	}
	if err := s.repo.CreateChallengeWithLocations(ctx, challenge, challengeLocationsFromSelected(selected)); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "daily challenge materialized", slog.String("challenge_id", challenge.ID.String()), slog.String("challenge_date", date.Format("2006-01-02")))
	return challenge, nil
}

func (s *Service) firstActiveMapID(ctx context.Context) (uuid.UUID, error) {
	if s.defaultMapID != uuid.Nil {
		return s.defaultMapID, nil
	}
	return s.repo.GetDefaultActiveMapID(ctx)
}

func (s *Service) selectUnique(ctx context.Context, mapID uuid.UUID, count int) ([]maps.SelectedLocation, error) {
	selected, err := s.selector.SelectLocations(ctx, mapID, count)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{}, count)
	unique := make([]maps.SelectedLocation, 0, count)
	for _, location := range selected {
		if location.ID == uuid.Nil {
			continue
		}
		if _, ok := seen[location.ID]; ok {
			continue
		}
		seen[location.ID] = struct{}{}
		unique = append(unique, location)
		if len(unique) == count {
			return unique, nil
		}
	}
	return nil, ErrNotEnoughLocations
}

func (s *Service) metadata(ctx context.Context, sess *session.Context, challenge Challenge) (*ChallengeMetadataResponse, error) {
	settings, err := decodeSettings(challenge.SettingsSnapshot)
	if err != nil {
		return nil, err
	}
	var attemptSummary *AttemptSummary
	owner, ownerErr := ownerFromSession(sess)
	if ownerErr == nil {
		attempt, err := s.repo.GetAttemptForOwner(ctx, challenge.ID, owner)
		if err != nil {
			return nil, err
		}
		if attempt != nil {
			summary := toAttemptSummary(*attempt, nil)
			attemptSummary = &summary
		}
	}
	participants, err := s.repo.CountLeaderboardEntries(ctx, challenge.ID)
	if err != nil {
		return nil, err
	}
	streak := EmptyStreakSummary(ownerErr == nil && owner.guestHash != nil)
	if ownerErr == nil {
		loaded, err := s.repo.GetStreakForOwner(ctx, owner)
		if err != nil {
			return nil, err
		}
		streak = toStreakSummary(loaded, owner.guestHash != nil)
	}
	var countdown *CountdownSummary
	if challenge.ResetEndsAt != nil {
		remaining := int64(time.Until(*challenge.ResetEndsAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		countdown = &CountdownSummary{ResetEndsAt: *challenge.ResetEndsAt, SecondsRemaining: remaining}
	}
	return &ChallengeMetadataResponse{
		Challenge:          toChallengeSummary(challenge, settings),
		AttemptState:       attemptSummary,
		Streak:             streak,
		MissionsSummary:    DefaultMissionSummaries(s.clock.Now()),
		LeaderboardSummary: LeaderboardSummary{Participants: participants},
		Countdown:          countdown,
	}, nil
}

func (s *Service) startAttempt(ctx context.Context, sess *session.Context, challenge Challenge) (*ChallengeAttemptResponse, error) {
	if challenge.Status != StatusActive {
		return nil, ErrChallengeUnavailable
	}
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	settings, err := decodeSettings(challenge.SettingsSnapshot)
	if err != nil {
		return nil, err
	}
	locationRows, err := s.repo.ListChallengeLocations(ctx, challenge.ID)
	if err != nil {
		return nil, err
	}
	if len(locationRows) < settings.RoundCount {
		return nil, ErrNotEnoughLocations
	}
	selected := make([]maps.SelectedLocation, settings.RoundCount)
	for i := 0; i < settings.RoundCount; i++ {
		selected[i] = maps.SelectedLocation{ID: locationRows[i].LocationID}
	}
	attempt, game, err := s.repo.CreateAttemptWithGame(ctx, challenge, owner, selected, settings, s.clock.Now())
	if err != nil {
		return nil, err
	}
	attempt, game, err = s.repo.StartAttemptGame(ctx, attempt.ID, s.clock.Now())
	if err != nil {
		return nil, err
	}
	gameDTO := games.GameDTO{ID: game.ID, Mode: game.Mode, Status: game.Status, MapID: game.MapID, RoundCount: game.RoundCount, TimerSeconds: game.TimerSeconds, ScoringVersion: game.ScoringVersion, CurrentRoundNumber: game.CurrentRoundNumber, TotalScore: game.TotalScore, StartedAt: game.StartedAt, CompletedAt: game.CompletedAt}
	s.logger.InfoContext(ctx, "challenge attempt started", slog.String("challenge_id", challenge.ID.String()), slog.String("attempt_id", attempt.ID.String()), slog.String("type", challenge.Type))
	return &ChallengeAttemptResponse{Challenge: toChallengeSummary(challenge, settings), Attempt: toAttemptSummary(*attempt, game.CurrentRoundNumber), Game: &gameDTO}, nil
}

func normalizeSettings(roundCount int, timerSeconds *int) (SettingsSnapshot, error) {
	if roundCount == 0 {
		roundCount = DefaultRoundCount
	}
	if roundCount < 1 || roundCount > 10 {
		return SettingsSnapshot{}, ErrInvalidChallengeInput
	}
	if timerSeconds != nil && (*timerSeconds < 10 || *timerSeconds > 600) {
		return SettingsSnapshot{}, ErrInvalidChallengeInput
	}
	return SettingsSnapshot{RoundCount: roundCount, TimerSeconds: timerSeconds, MovementRules: "standard", ScoringVersion: DefaultScoringVersion}, nil
}

func challengeLocationsFromSelected(selected []maps.SelectedLocation) []ChallengeLocation {
	rows := make([]ChallengeLocation, len(selected))
	for i, location := range selected {
		rows[i] = ChallengeLocation{RoundNumber: i + 1, LocationID: location.ID, SelectionVersion: 1}
	}
	return rows
}

func toChallengeSummary(challenge Challenge, settings SettingsSnapshot) ChallengeSummary {
	var challengeDate *string
	if challenge.ChallengeDate != nil {
		v := challenge.ChallengeDate.UTC().Format("2006-01-02")
		challengeDate = &v
	}
	return ChallengeSummary{ID: challenge.ID, Type: challenge.Type, Seed: challenge.Seed, ChallengeDate: challengeDate, ResetStartsAt: challenge.ResetStartsAt, ResetEndsAt: challenge.ResetEndsAt, Map: MapSummary{ID: challenge.MapID}, Settings: settings, Status: challenge.Status, ShareCode: challenge.SlugOrCode}
}

func toAttemptSummary(attempt ChallengeAttempt, currentRound *int) AttemptSummary {
	return AttemptSummary{ID: attempt.ID, ChallengeID: attempt.ChallengeID, Status: attempt.Status, LeaderboardEligible: attempt.LeaderboardEligible, StartedAt: attempt.StartedAt, CompletedAt: attempt.CompletedAt, TotalScore: attempt.TotalScore, CurrentRoundNumber: currentRound, GameID: attempt.GameID}
}

func toStreakSummary(streak *Streak, guestLimited bool) StreakSummary {
	if streak == nil {
		return EmptyStreakSummary(guestLimited)
	}
	var last *string
	if streak.LastCompletedChallengeDate != nil {
		v := streak.LastCompletedChallengeDate.UTC().Format("2006-01-02")
		last = &v
	}
	return StreakSummary{CurrentCount: streak.CurrentCount, BestCount: streak.BestCount, LastCompletedChallengeDate: last, Status: streak.Status, ProtectionState: streak.ProtectionState, GuestLimited: guestLimited}
}

type ownerIdentity struct {
	userID      *uuid.UUID
	guestHash   *string
	displayName string
}

func ownerFromSession(sess *session.Context) (ownerIdentity, error) {
	if sess == nil {
		return ownerIdentity{}, ErrForbidden
	}
	if sess.IsRegistered() {
		id, err := uuid.Parse(*sess.UserID)
		if err != nil {
			return ownerIdentity{}, ErrForbidden
		}
		return ownerIdentity{userID: &id, displayName: "Player"}, nil
	}
	if sess.IsGuest() {
		guest := *sess.GuestID
		return ownerIdentity{guestHash: &guest, displayName: "Guest"}, nil
	}
	return ownerIdentity{}, ErrForbidden
}
