package challenges

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	SelectLocationsBySeed(ctx context.Context, mapID uuid.UUID, count int, seed string) ([]maps.SelectedLocation, error)
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
	idempotency  IdempotencyStore
}

func NewService(repo *Repository, selector LocationSelector, clk clock.Clock, logger *slog.Logger, resetHourUTC int, defaultMapID uuid.UUID, metrics MetricsRecorder) *Service {
	return NewServiceWithIdempotency(repo, selector, clk, logger, resetHourUTC, defaultMapID, metrics, nil)
}

func NewServiceWithIdempotency(repo *Repository, selector LocationSelector, clk clock.Clock, logger *slog.Logger, resetHourUTC int, defaultMapID uuid.UUID, metrics MetricsRecorder, idempotency IdempotencyStore) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, selector: selector, clock: clk, logger: logger, resetHourUTC: resetHourUTC, defaultMapID: defaultMapID, metrics: metrics, idempotency: idempotency}
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

func (s *Service) StartDailyAttempt(ctx context.Context, sess *session.Context, idempotencyKey string) (*ChallengeAttemptResponse, error) {
	replay, op, handled, err := beginIdempotency[ChallengeAttemptResponse](ctx, s.idempotency, idempotencyKey, "start_daily_attempt", sess, nil)
	if handled || err != nil {
		return replay, err
	}
	defer releaseIdempotency(ctx, op)
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
	resp, err := s.startAttempt(ctx, sess, *challenge)
	if err != nil {
		return nil, err
	}
	if err := storeIdempotencyResponse(ctx, op, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) CreateShared(ctx context.Context, sess *session.Context, idempotencyKey string, req CreateSharedChallengeRequest) (*ChallengeMetadataResponse, error) {
	replay, op, handled, err := beginIdempotency[ChallengeMetadataResponse](ctx, s.idempotency, idempotencyKey, "create_shared_challenge", sess, req)
	if handled || err != nil {
		return replay, err
	}
	defer releaseIdempotency(ctx, op)
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
	seed, err := SharedSeed()
	if err != nil {
		return nil, err
	}
	selected, err := s.selectUniqueBySeed(ctx, req.MapID, settings.RoundCount, seed)
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
	resp, err := s.metadata(ctx, sess, *challenge)
	if err != nil {
		return nil, err
	}
	if err := storeIdempotencyResponse(ctx, op, resp); err != nil {
		return nil, err
	}
	return resp, nil
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

func (s *Service) StartChallengeAttempt(ctx context.Context, sess *session.Context, idempotencyKey string, challengeID string) (*ChallengeAttemptResponse, error) {
	replay, op, handled, err := beginIdempotency[ChallengeAttemptResponse](ctx, s.idempotency, idempotencyKey, "start_challenge_attempt:"+challengeID, sess, nil)
	if handled || err != nil {
		return replay, err
	}
	defer releaseIdempotency(ctx, op)
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
	resp, err := s.startAttempt(ctx, sess, *challenge)
	if err != nil {
		return nil, err
	}
	if err := storeIdempotencyResponse(ctx, op, resp); err != nil {
		return nil, err
	}
	return resp, nil
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
	settings, err := decodeSettings(challenge.SettingsSnapshot)
	if err != nil {
		return nil, err
	}
	if attempt.Status != AttemptStatusCompleted {
		s.logger.InfoContext(ctx, "challenge result spoiler protected", slog.String("challenge_id", id.String()), slog.String("attempt_id", attempt.ID.String()))
		return &ResultResponse{
			Challenge: toChallengeSummary(*challenge, settings),
			Attempt:   toAttemptSummary(*attempt, nil),
			Visible:   false,
			Message:   "Results are available after completing the challenge.",
		}, nil
	}
	result, err := s.repo.GetResultByAttempt(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrResultsNotReady
	}
	var rounds []RoundResultDTO
	if err := json.Unmarshal(result.RoundResultsSnapshot, &rounds); err != nil {
		return nil, fmt.Errorf("decode challenge result snapshot: %w", err)
	}
	totalScore := result.TotalScore
	totalDistance := result.TotalDistanceMeters
	resp := &ResultResponse{
		Challenge:     toChallengeSummary(*challenge, settings),
		Attempt:       toAttemptSummary(*attempt, nil),
		Visible:       true,
		TotalScore:    &totalScore,
		TotalDistance: &totalDistance,
		RoundResults:  rounds,
	}
	if len(result.RankSnapshot) > 0 {
		rankContext := result.RankSnapshot
		resp.RankContext = &rankContext
	}
	streak, err := s.repo.GetStreakForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	streakSummary := toStreakSummary(streak, owner.guestHash != nil)
	resp.Streak = &streakSummary
	missions, err := s.GetMissions(ctx, sess)
	if err != nil {
		return nil, err
	}
	resp.MissionsSummary = missions
	return resp, nil
}

func (s *Service) GetLeaderboard(ctx context.Context, sess *session.Context, challengeID string, limit int, cursor string) (*LeaderboardResponse, error) {
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
	entries, err := s.repo.ListLeaderboardEntries(ctx, id, limit, cursor)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var nextCursor *string
	if len(entries) == limit {
		cursor := encodeLeaderboardCursor(entries[len(entries)-1])
		nextCursor = &cursor
	}
	var currentUserID *uuid.UUID
	if sess != nil && sess.IsRegistered() {
		if uid, parseErr := uuid.Parse(*sess.UserID); parseErr == nil {
			currentUserID = &uid
		}
	}
	dtos := make([]LeaderboardEntryDTO, len(entries))
	for i, entry := range entries {
		dto := LeaderboardEntryDTO{Rank: entry.Rank, DisplayName: entry.DisplayNameSnapshot, Score: entry.Score, CompletionDurationMS: entry.CompletionDurationMS, CompletedAt: entry.CompletedAt}
		if currentUserID != nil && entry.UserID == *currentUserID {
			dto.CurrentPlayer = true
		}
		dtos[i] = dto
	}
	s.logger.InfoContext(ctx, "challenge leaderboard read", slog.String("challenge_id", id.String()), slog.Int("entries", len(entries)))
	return &LeaderboardResponse{Challenge: toChallengeSummary(*challenge, settings), Entries: dtos, Page: PageInfo{Limit: limit, NextCursor: nextCursor}}, nil
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
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	missions, err := s.repo.ListActiveMissions(ctx, now)
	if err != nil {
		return nil, err
	}
	progressList, err := s.repo.ListMissionProgressForOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	progressByMission := make(map[uuid.UUID]MissionProgress, len(progressList))
	for _, p := range progressList {
		progressByMission[p.MissionID] = p
	}
	if len(missions) == 0 {
		defaults := DefaultMissionSummaries(now)
		for _, m := range defaults {
			if existing, lookupErr := s.repo.GetMissionByCode(ctx, m.Code); lookupErr == nil && existing == nil {
				dbMission := Mission{
					Code:           m.Code,
					TitleKey:       m.TitleKey,
					DescriptionKey: m.DescriptionKey,
					MissionType:    m.MissionType,
					TargetValue:    m.TargetValue,
					ActiveStartsAt: now,
					ActiveEndsAt:   m.ActiveEndsAt,
					RewardSnapshot: json.RawMessage("{}"),
					Status:         "active",
				}
				_ = s.repo.UpsertMission(ctx, &dbMission)
			}
		}
		missions, err = s.repo.ListActiveMissions(ctx, now)
		if err != nil {
			return nil, err
		}
	}
	summaries := make([]MissionSummary, len(missions))
	for i, m := range missions {
		summaries[i] = MissionSummary{
			ID:             m.ID,
			Code:           m.Code,
			TitleKey:       m.TitleKey,
			DescriptionKey: m.DescriptionKey,
			MissionType:    m.MissionType,
			TargetValue:    m.TargetValue,
			ActiveEndsAt:   m.ActiveEndsAt,
			Status:         "not_started",
		}
		if progress, ok := progressByMission[m.ID]; ok {
			summaries[i].CurrentValue = progress.CurrentValue
			if progress.ClaimedAt != nil {
				summaries[i].Status = "claimed"
			} else if progress.Status == "completed" {
				summaries[i].Status = "completed"
			} else {
				summaries[i].Status = progress.Status
			}
		}
	}
	return summaries, nil
}

func (s *Service) ClaimMission(ctx context.Context, sess *session.Context, idempotencyKey string, missionID string) (*MissionSummary, error) {
	replay, op, handled, err := beginIdempotency[MissionSummary](ctx, s.idempotency, idempotencyKey, "claim_mission:"+missionID, sess, nil)
	if handled || err != nil {
		return replay, err
	}
	defer releaseIdempotency(ctx, op)
	owner, err := ownerFromSession(sess)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(missionID)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	progress, err := s.repo.GetMissionProgress(ctx, id, owner)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		return nil, ErrChallengeNotFound
	}
	if progress.Status != "completed" {
		return nil, ErrResultsNotReady
	}
	if progress.ClaimedAt != nil {
		resp := &MissionSummary{ID: progress.MissionID, CurrentValue: progress.CurrentValue, TargetValue: progress.TargetValue, Status: "claimed"}
		if err := storeIdempotencyResponse(ctx, op, resp); err != nil {
			return nil, err
		}
		return resp, nil
	}
	if err := s.repo.ClaimMissionProgress(ctx, id, owner, s.clock.Now()); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "challenge mission claimed", slog.String("mission_id", missionID))
	now := s.clock.Now()
	resp := &MissionSummary{ID: progress.MissionID, CurrentValue: progress.CurrentValue, TargetValue: progress.TargetValue, Status: "claimed", ActiveEndsAt: &now}
	if err := storeIdempotencyResponse(ctx, op, resp); err != nil {
		return nil, err
	}
	return resp, nil
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
	challenge, err := s.repo.GetChallengeByID(ctx, attempt.ChallengeID)
	if err != nil || challenge == nil {
		return err
	}
	owner, err := s.attemptOwner(attempt)
	if err != nil {
		return err
	}
	if challenge.Type == TypeDaily && challenge.ChallengeDate != nil {
		s.updateStreak(ctx, owner, *challenge.ChallengeDate)
	}
	s.fireMissionProgress(ctx, attempt, challenge, roundResults, owner)
	return nil
}

func (s *Service) OnGameCompleted(ctx context.Context, gameID uuid.UUID, completedAt time.Time) error {
	attempt, err := s.repo.GetAttemptByGameID(ctx, gameID)
	if err != nil {
		return err
	}
	if attempt == nil {
		return nil
	}
	if attempt.Status == AttemptStatusCompleted {
		return nil
	}
	gameData, err := s.repo.GetGameCompletionData(ctx, gameID)
	if err != nil {
		return err
	}
	roundResults, err := s.repo.LoadGameRoundResults(ctx, gameID)
	if err != nil {
		return err
	}
	var durationMS *int64
	if gameData.StartedAt != nil {
		d := completedAt.Sub(*gameData.StartedAt).Milliseconds()
		durationMS = &d
	}
	owner, err := s.attemptOwner(*attempt)
	if err != nil {
		return err
	}
	resultsDTO := make([]RoundResultDTO, len(roundResults))
	totalDistance := 0
	for i, r := range roundResults {
		resultsDTO[i].RoundNumber = r.RoundNumber
		resultsDTO[i].Score = r.Score
		resultsDTO[i].DistanceMeters = r.DistanceMeters
		totalDistance += r.DistanceMeters
	}
	if err := s.repo.UpdateAttemptCompletion(ctx, attempt.ID, gameData.TotalScore, totalDistance, durationMS, completedAt); err != nil {
		return err
	}
	attempt.TotalScore = gameData.TotalScore
	attempt.CompletionDurationMS = durationMS
	attempt.CompletedAt = &completedAt
	attempt.TotalDistanceMeters = totalDistance
	return s.FinalizeAttemptResult(ctx, *attempt, resultsDTO, completedAt, owner.displayName)
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
	seed := DailySeed(date)
	selected, err := s.selectUniqueBySeed(ctx, mapID, settings.RoundCount, seed)
	if err != nil {
		return nil, err
	}
	rawSettings, err := encodeSettings(settings)
	if err != nil {
		return nil, err
	}
	challenge := &Challenge{
		Type:             TypeDaily,
		Seed:             seed,
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

func (s *Service) selectUniqueBySeed(ctx context.Context, mapID uuid.UUID, count int, seed string) ([]maps.SelectedLocation, error) {
	selected, err := s.selector.SelectLocationsBySeed(ctx, mapID, count, seed)
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
		remaining := int64(s.clock.Until(*challenge.ResetEndsAt).Seconds())
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

const (
	idempotencyReplayTTL = 24 * time.Hour
	idempotencyLockTTL   = 2 * time.Minute
)

type idempotencyOperation struct {
	store       IdempotencyStore
	key         string
	fingerprint string
	claimed     bool
}

func beginIdempotency[T any](ctx context.Context, store IdempotencyStore, rawKey, operation string, sess *session.Context, body any) (*T, *idempotencyOperation, bool, error) {
	rawKey = strings.TrimSpace(rawKey)
	if store == nil || rawKey == "" {
		return nil, nil, false, nil
	}
	storageKey := idempotencyStorageKey(operation, sess, rawKey)
	fingerprint, err := idempotencyFingerprint(operation, sess, body)
	if err != nil {
		return nil, nil, false, err
	}
	record, err := store.Get(ctx, storageKey)
	if err != nil {
		return nil, nil, false, err
	}
	if record != nil {
		if record.Fingerprint != fingerprint {
			return nil, nil, true, ErrIdempotencyConflict
		}
		var replay T
		if err := json.Unmarshal(record.Payload, &replay); err != nil {
			return nil, nil, false, fmt.Errorf("decode challenge idempotency replay: %w", err)
		}
		return &replay, nil, true, nil
	}
	claimed, err := store.Claim(ctx, storageKey, idempotencyLockTTL)
	if err != nil {
		return nil, nil, false, err
	}
	if !claimed {
		return nil, nil, true, ErrIdempotencyConflict
	}
	return nil, &idempotencyOperation{store: store, key: storageKey, fingerprint: fingerprint, claimed: true}, false, nil
}

func storeIdempotencyResponse[T any](ctx context.Context, op *idempotencyOperation, resp *T) error {
	if op == nil || op.store == nil {
		return nil
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("encode challenge idempotency replay: %w", err)
	}
	return op.store.Store(ctx, op.key, IdempotencyRecord{Fingerprint: op.fingerprint, Payload: payload}, idempotencyReplayTTL)
}

func releaseIdempotency(ctx context.Context, op *idempotencyOperation) {
	if op == nil || op.store == nil || !op.claimed {
		return
	}
	_ = op.store.Release(ctx, op.key)
	op.claimed = false
}

func idempotencyStorageKey(operation string, sess *session.Context, rawKey string) string {
	sum := sha256.Sum256([]byte(operation + "|" + idempotencyActor(sess) + "|" + rawKey))
	return "challenges:idempotency:" + hex.EncodeToString(sum[:])
}

func idempotencyFingerprint(operation string, sess *session.Context, body any) (string, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("encode challenge idempotency fingerprint: %w", err)
	}
	sum := sha256.Sum256([]byte(operation + "|" + idempotencyActor(sess) + "|" + string(bodyJSON)))
	return hex.EncodeToString(sum[:]), nil
}

func idempotencyActor(sess *session.Context) string {
	if sess == nil {
		return "none"
	}
	if sess.IsRegistered() && sess.UserID != nil {
		return "user:" + *sess.UserID
	}
	if sess.IsGuest() && sess.GuestID != nil {
		return "guest:" + *sess.GuestID
	}
	return "anonymous"
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

func (s *Service) attemptOwner(attempt ChallengeAttempt) (ownerIdentity, error) {
	owner := ownerIdentity{userID: attempt.UserID, guestHash: attempt.GuestIdentityHash, displayName: "Player"}
	if owner.guestHash != nil {
		owner.displayName = "Guest"
	}
	if owner.userID == nil && owner.guestHash == nil {
		return ownerIdentity{}, ErrForbidden
	}
	return owner, nil
}

func (s *Service) updateStreak(ctx context.Context, owner ownerIdentity, challengeDate time.Time) {
	now := s.clock.Now()
	existing, err := s.repo.GetStreakForOwner(ctx, owner)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to load streak for update", slog.Any("error", err))
		return
	}
	next := ApplyDailyCompletion(existing, challengeDate, now)
	event := StreakEvent{
		OwnerUserID:       owner.userID,
		GuestIdentityHash: owner.guestHash,
		ChallengeDate:     &challengeDate,
		EventType:         "daily_completion",
		PreviousCount:     0,
		NewCount:          next.CurrentCount,
	}
	if existing != nil {
		event.PreviousCount = existing.CurrentCount
	}
	if existing != nil && next.CurrentCount == existing.CurrentCount {
		return
	}
	if err := s.repo.UpsertStreak(ctx, owner, next); err != nil {
		s.logger.ErrorContext(ctx, "failed to upsert streak", slog.Any("error", err))
		return
	}
	_ = s.repo.CreateStreakEvent(ctx, event)
	s.logger.InfoContext(ctx, "streak updated", slog.Int("current_count", next.CurrentCount), slog.Int("best_count", next.BestCount))
}

func (s *Service) fireMissionProgress(ctx context.Context, attempt ChallengeAttempt, challenge *Challenge, roundResults any, owner ownerIdentity) {
	now := s.clock.Now()
	resultJSON, _ := json.Marshal(roundResults)
	var results []struct {
		RoundNumber    int `json:"round_number"`
		Score          int `json:"score"`
		DistanceMeters int `json:"distance_meters"`
	}
	_ = json.Unmarshal(resultJSON, &results)
	eventSource := &attempt.ID
	challengeSource := &attempt.ChallengeID
	if challenge.Type == TypeDaily {
		s.applyMissionEvent(ctx, owner, now, "daily_completion", 1, eventSource, challengeSource)
		if attempt.UserID != nil {
			s.applyMissionEvent(ctx, owner, now, "streak_milestone", 1, eventSource, challengeSource)
		}
	}
	if challenge.Type == TypeShared {
		s.applyMissionEvent(ctx, owner, now, "shared_participation", 1, eventSource, challengeSource)
	}
	highAccuracy := true
	for _, r := range results {
		if r.Score < 4000 {
			highAccuracy = false
			break
		}
	}
	if highAccuracy && len(results) > 0 {
		s.applyMissionEvent(ctx, owner, now, "round_accuracy", 1, eventSource, challengeSource)
	}
	s.applyMissionEvent(ctx, owner, now, "score_threshold", attempt.TotalScore, eventSource, challengeSource)
}

func (s *Service) applyMissionEvent(ctx context.Context, owner ownerIdentity, now time.Time, missionCode string, delta int, sourceAttemptID, sourceChallengeID *uuid.UUID) {
	if delta <= 0 {
		return
	}
	mission, err := s.repo.GetMissionByCode(ctx, missionCode)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to lookup mission for progress event", slog.String("mission_code", missionCode), slog.Any("error", err))
		return
	}
	if mission == nil {
		return
	}
	event := MissionProgressEvent{
		SourceAttemptID:   sourceAttemptID,
		SourceChallengeID: sourceChallengeID,
		EventType:         missionCode,
		Delta:             delta,
	}
	if err := s.repo.ApplyMissionProgressEvent(ctx, *mission, owner, event, now); err != nil {
		s.logger.ErrorContext(ctx, "failed to apply mission progress", slog.String("mission_code", missionCode), slog.Any("error", err))
	}
}
