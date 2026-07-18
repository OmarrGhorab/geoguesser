package matchplay

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/session"
)

// ServiceConfig holds matchplay timing knobs.
type ServiceConfig struct {
	ReconnectGrace   time.Duration
	CasualInactivity time.Duration
	SweepBatchSize   int
}

// Service owns authorized match snapshot, result, leave, chat, and lifecycle operations.
type Service struct {
	store    Store
	parties  PartyRestorer
	versions VersionStore
	presence PresenceGraceStore
	events   EventSink
	media    MediaResolver
	clock    func() time.Time
	logger   *slog.Logger
	metrics  MetricsRecorder
	cfg      ServiceConfig
	chat     chatDeps
}

// NewService constructs a matchplay service.
func NewService(store Store, logger *slog.Logger, metrics MetricsRecorder, cfg ServiceConfig) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	if cfg.ReconnectGrace <= 0 {
		cfg.ReconnectGrace = DefaultReconnectGrace
	}
	if cfg.CasualInactivity <= 0 {
		cfg.CasualInactivity = DefaultCasualInactivity
	}
	if cfg.SweepBatchSize <= 0 {
		cfg.SweepBatchSize = DefaultSweepBatchSize
	}
	return &Service{
		store:   store,
		clock:   func() time.Time { return time.Now().UTC() },
		logger:  logger,
		metrics: metrics,
		cfg:     cfg,
	}
}

// WithPartyRestorer attaches post-terminal party restoration.
func (s *Service) WithPartyRestorer(r PartyRestorer) *Service {
	s.parties = r
	return s
}

// WithVersionStore attaches match channel version counters.
func (s *Service) WithVersionStore(v VersionStore) *Service {
	s.versions = v
	return s
}

// WithPresence attaches reconnect grace lookups for disconnect sweeps.
func (s *Service) WithPresence(p PresenceGraceStore) *Service {
	s.presence = p
	return s
}

// WithEvents attaches a post-commit event sink.
func (s *Service) WithEvents(e EventSink) *Service {
	s.events = e
	return s
}

// WithMedia attaches round media resolution.
func (s *Service) WithMedia(m MediaResolver) *Service {
	s.media = m
	return s
}

// WithClock overrides wall time (tests).
func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.clock = now
	}
	return s
}

// AuthorizeRealtime returns the participant row for a realtime ticket/WS check.
// Non-participants receive privacy-safe ErrNotFound.
func (s *Service) AuthorizeRealtime(ctx context.Context, userID, matchID uuid.UUID) (*MatchParticipant, error) {
	if s == nil || s.store == nil {
		return nil, ErrUnavailable
	}
	part, err := s.store.FindParticipant(ctx, matchID, userID)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if part == nil {
		return nil, ErrNotFound
	}
	return part, nil
}

// SnapshotForRealtime builds an authorized match snapshot for the initial WS message.
func (s *Service) SnapshotForRealtime(ctx context.Context, userID, matchID uuid.UUID) (*MatchSnapshotResponse, error) {
	uid := userID.String()
	sess := &session.Context{Kind: session.KindUser, UserID: &uid}
	return s.GetSnapshot(ctx, sess, matchID)
}

// MarkRealtimeConnected clears a durable disconnect marker after the first
// active socket for this participant is established.
func (s *Service) MarkRealtimeConnected(ctx context.Context, matchID, userID uuid.UUID) error {
	return s.markRealtimeConnection(ctx, matchID, userID, true)
}

// MarkRealtimeDisconnected records when the participant's final active socket closes.
func (s *Service) MarkRealtimeDisconnected(ctx context.Context, matchID, userID uuid.UUID) error {
	return s.markRealtimeConnection(ctx, matchID, userID, false)
}

func (s *Service) markRealtimeConnection(ctx context.Context, matchID, userID uuid.UUID, connected bool) error {
	if s == nil || s.store == nil {
		return ErrUnavailable
	}
	connectionStore, ok := s.store.(connectionStateStore)
	if !ok {
		return nil
	}
	changed, err := connectionStore.MarkConnectionState(ctx, matchID, userID, connected, s.clock())
	if err != nil {
		return mapStoreErr(err)
	}
	if !changed || s.events == nil {
		return nil
	}
	version := int64(0)
	if s.versions != nil {
		version, _ = s.versions.NextVersion(ctx, matchID)
	}
	eventType := EventMatchPlayerDisconnected
	if connected {
		eventType = EventMatchPlayerReconnected
	}
	return s.events.PublishMatchEvent(ctx, matchID, eventType, version, nil, nil, map[string]any{
		"user_id": userID.String(),
	})
}

// GetSnapshot returns the caller-authorized match snapshot.
// Non-participants receive privacy-safe ErrNotFound.
func (s *Service) GetSnapshot(ctx context.Context, sess *session.Context, matchID uuid.UUID) (*MatchSnapshotResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveCommand("snapshot", "unauthorized", s.clock().Sub(start))
		return nil, err
	}

	bundle, err := s.store.LoadSnapshotBundle(ctx, matchID)
	if err != nil {
		s.metrics.ObserveCommand("snapshot", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if bundle == nil {
		s.metrics.ObserveCommand("snapshot", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	viewerPart, ok := findParticipant(bundle.Participants, userID)
	if !ok {
		// Privacy-safe: do not distinguish missing match vs non-participant.
		s.metrics.ObserveCommand("snapshot", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	if s.versions != nil {
		if v, vErr := s.versions.CurrentVersion(ctx, matchID); vErr == nil {
			bundle.RealtimeVersion = v
		}
	}

	dto := s.projectSnapshot(userID, viewerPart, bundle)
	s.metrics.ObserveCommand("snapshot", "ok", s.clock().Sub(start))
	return &MatchSnapshotResponse{Match: dto}, nil
}

// GetRoundResults returns fully revealed round results after the round is completed.
func (s *Service) GetRoundResults(ctx context.Context, sess *session.Context, matchID, roundID uuid.UUID) (*RoundResultsResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveCommand("round_results", "unauthorized", s.clock().Sub(start))
		return nil, err
	}

	// Authorization first via participant check so missing vs forbidden stay privacy-safe.
	part, err := s.store.FindParticipant(ctx, matchID, userID)
	if err != nil {
		s.metrics.ObserveCommand("round_results", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if part == nil {
		s.metrics.ObserveCommand("round_results", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	bundle, err := s.store.LoadRoundResultBundle(ctx, matchID, roundID)
	if err != nil {
		if err == ErrRoundNotRevealed {
			s.metrics.ObserveCommand("round_results", "not_revealed", s.clock().Sub(start))
			return nil, ErrRoundNotRevealed
		}
		s.metrics.ObserveCommand("round_results", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if bundle == nil {
		s.metrics.ObserveCommand("round_results", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	result := projectRevealedRound(bundle)
	s.metrics.ObserveCommand("round_results", "ok", s.clock().Sub(start))
	return &RoundResultsResponse{Result: result}, nil
}

// GetTerminalResult returns terminal match results with progression projection.
// Casual always returns progression.applied=false reason=casual (no Elo mutation).
// Ranked may return ErrProgressionPending while finalization is still in flight.
func (s *Service) GetTerminalResult(ctx context.Context, sess *session.Context, matchID uuid.UUID) (*TerminalResultResponse, error) {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveCommand("terminal_result", "unauthorized", s.clock().Sub(start))
		return nil, err
	}

	part, err := s.store.FindParticipant(ctx, matchID, userID)
	if err != nil {
		s.metrics.ObserveCommand("terminal_result", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if part == nil {
		s.metrics.ObserveCommand("terminal_result", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	bundle, err := s.store.LoadTerminalResultBundle(ctx, matchID)
	if err != nil {
		if err == ErrMatchNotActive {
			// Still active — not a terminal result yet.
			s.metrics.ObserveCommand("terminal_result", "not_terminal", s.clock().Sub(start))
			return nil, ErrMatchNotActive
		}
		s.metrics.ObserveCommand("terminal_result", "error", s.clock().Sub(start))
		return nil, mapStoreErr(err)
	}
	if bundle == nil {
		s.metrics.ObserveCommand("terminal_result", "not_found", s.clock().Sub(start))
		return nil, ErrNotFound
	}

	if IsRankedMatch(bundle.Match) && bundle.Match.ProgressionFinalizedAt == nil {
		s.metrics.ObserveCommand("terminal_result", "progression_pending", s.clock().Sub(start))
		return nil, ErrProgressionPending
	}

	resp := projectTerminalResult(bundle)
	s.metrics.ObserveCommand("terminal_result", "ok", s.clock().Sub(start))
	return resp, nil
}

// Leave explicitly abandons the match for the caller (team forfeit).
// Terminal replay is idempotent (nil error, no mutation of winner/abandon facts).
// Casual is progression-neutral: no competitive rating rows are written.
// Ranked: team forfeit with normal opponent win facts; progression stays pending
// until competitive finalization applies base Elo + quitter-only -15.
func (s *Service) Leave(ctx context.Context, sess *session.Context, matchID uuid.UUID, _ LeaveRequest) error {
	start := s.clock()
	userID, err := requireRegistered(sess)
	if err != nil {
		s.metrics.ObserveCommand("leave", "unauthorized", s.clock().Sub(start))
		return err
	}

	now := s.clock()
	outcome, err := s.store.ExplicitLeaveTx(ctx, matchID, userID, now)
	if err != nil {
		if err == ErrNotFound {
			s.metrics.ObserveCommand("leave", "not_found", s.clock().Sub(start))
			return ErrNotFound
		}
		s.metrics.ObserveCommand("leave", "error", s.clock().Sub(start))
		return mapStoreErr(err)
	}

	if !outcome.AlreadyTerminal {
		s.publishLeaveEvents(ctx, outcome, AbandonReasonExplicitLeave)
		s.restoreParties(ctx, outcome)
		if IsRankedMatch(outcome.Match) {
			s.metrics.ObserveAbandon("explicit", "forfeited")
		}
	} else {
		// Still attempt party restore for terminal replays (idempotent).
		s.restoreParties(ctx, outcome)
		if IsRankedMatch(outcome.Match) {
			s.metrics.ObserveAbandon("explicit", "replay")
		}
	}

	s.metrics.ObserveCommand("leave", "ok", s.clock().Sub(start))
	return nil
}

// SweepDisconnectGrace forfeits participants past reconnect grace (bounded).
// Progression-neutral: no Elo writes.
func (s *Service) SweepDisconnectGrace(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	now := s.clock()
	candidates, err := s.store.ListDisconnectCandidates(ctx, now, s.cfg.ReconnectGrace, s.cfg.SweepBatchSize)
	if err != nil {
		s.metrics.ObserveSweep("disconnect", "error")
		return err
	}
	for _, c := range candidates {
		if s.presence != nil {
			// If a reconnect window still exists, grace has not elapsed for this instance.
			if ok, pErr := s.presence.HasReconnectWindow(ctx, c.MatchID, c.UserID); pErr == nil && ok {
				continue
			}
		}
		// Durable left_at/updated_at already past grace from the list query.
		outcome, fErr := s.store.ForfeitDisconnectTx(ctx, c.MatchID, c.UserID, now)
		if fErr != nil {
			s.logger.WarnContext(ctx, "disconnect forfeit failed",
				slog.String("match_id", c.MatchID.String()),
				slog.Any("error", fErr),
			)
			continue
		}
		if outcome != nil && !outcome.AlreadyTerminal {
			s.publishLeaveEvents(ctx, outcome, AbandonReasonDisconnectTimeout)
			s.restoreParties(ctx, outcome)
			if s.presence != nil {
				_ = s.presence.ClearPresence(ctx, c.MatchID, c.UserID)
			}
			s.metrics.ObserveSweep("disconnect", "forfeited")
			if IsRankedMatch(outcome.Match) {
				s.metrics.ObserveAbandon("disconnect", "forfeited")
			}
		}
	}
	s.metrics.ObserveSweep("disconnect", "ok")
	return nil
}

// SweepCasualInactivity closes casual matches idle past the inactivity window.
// Progression-neutral: no Elo writes.
func (s *Service) SweepCasualInactivity(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	now := s.clock()
	cutoff := now.Add(-s.cfg.CasualInactivity)
	candidates, err := s.store.ListInactiveCasualMatches(ctx, cutoff, s.cfg.SweepBatchSize)
	if err != nil {
		s.metrics.ObserveSweep("inactivity", "error")
		return err
	}
	for _, c := range candidates {
		outcome, cErr := s.store.CloseInactiveCasualTx(ctx, c.MatchID, now)
		if cErr != nil {
			s.logger.WarnContext(ctx, "casual inactivity close failed",
				slog.String("match_id", c.MatchID.String()),
				slog.Any("error", cErr),
			)
			continue
		}
		if outcome != nil && !outcome.AlreadyTerminal {
			s.publishInactivityEvents(ctx, outcome)
			s.restoreParties(ctx, outcome)
			s.metrics.ObserveSweep("inactivity", "closed")
		}
	}
	s.metrics.ObserveSweep("inactivity", "ok")
	return nil
}

// RunLifecycleSweep executes disconnect + inactivity sweeps once (worker tick).
func (s *Service) RunLifecycleSweep(ctx context.Context) error {
	var first error
	if err := s.SweepDisconnectGrace(ctx); err != nil && first == nil {
		first = err
	}
	if err := s.SweepCasualInactivity(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

func (s *Service) projectSnapshot(viewerID uuid.UUID, viewer MatchParticipant, bundle *SnapshotBundle) MatchSnapshotDTO {
	// Pre-reveal: redact opponent private fields and individual totals.
	// Reveal is allowed when the last completed round exists and there is no active
	// round waiting, or when the current round is completed (terminal mid-snapshot).
	revealed := roundIsRevealed(bundle)

	redacted := map[uuid.UUID]bool{}
	if !revealed {
		for _, p := range bundle.Participants {
			if p.TeamSlot != viewer.TeamSlot {
				redacted[p.UserID] = true
			}
		}
	}

	submitted := bundle.SubmittedIDs
	if submitted == nil {
		submitted = map[uuid.UUID]bool{}
	}

	teams := NewTeamProjections(
		bundle.Participants,
		bundle.Players,
		submitted,
		bundle.Match.TeamOneScore,
		bundle.Match.TeamTwoScore,
		redacted,
	)

	viewerSubmitted := submitted[viewer.GamePlayerID]
	dto := MatchSnapshotDTO{
		ID:       bundle.Match.ID,
		GameID:   bundle.Match.GameID,
		Playlist: playlistOf(bundle.Match),
		Format:   formatOf(bundle.Match),
		Status:   bundle.Match.Status,
		Result:   bundle.Match.Result,
		TeamSize: teamSizeOf(bundle.Match),
		Viewer: ViewerProjection{
			UserID:                   viewerID,
			GamePlayerID:             viewer.GamePlayerID,
			TeamSlot:                 viewer.TeamSlot,
			Submitted:                viewerSubmitted,
			CanChat:                  canChat(bundle.Match, s.clock()),
			AllowedSpectatePlayerIDs: allowedSpectate(viewer, bundle, viewerSubmitted),
		},
		Teams:           teams,
		TeamMarkers:     []TeamMarkerDTO{},
		RealtimeVersion: bundle.RealtimeVersion,
		FormedAt:        bundle.Match.MatchedAt.UTC(),
	}
	if IsRankedMatch(bundle.Match) {
		// Ranked authoritative round timer (60s); exposed for client countdown sync.
		timer := 60
		dto.TimerSeconds = &timer
	}

	if bundle.CurrentRound != nil {
		dto.Round = &RoundProjection{
			ID:             bundle.CurrentRound.ID,
			Number:         bundle.CurrentRound.RoundNumber,
			Status:         bundle.CurrentRound.Status,
			StartsAt:       bundle.CurrentRound.StartsAt,
			EndsAt:         bundle.CurrentRound.EndsAt,
			SubmittedCount: len(submitted),
			EligibleCount:  bundle.EligibleCount,
		}
		// Media is safe (panorama ref only); never include answer coordinates.
		// Current-round media is left nil when the media provider is not wired or
		// only answer metadata is available — answer coordinates stay out of snapshots.
		_ = s.media
	}

	if revealed && bundle.LastCompletedRound != nil && bundle.LastRoundAnswer != nil {
		rb := &RoundResultBundle{
			Match:        bundle.Match,
			Participants: bundle.Participants,
			Players:      bundle.Players,
			Round:        *bundle.LastCompletedRound,
			Guesses:      bundle.LastRoundGuesses,
			Answer:       *bundle.LastRoundAnswer,
			TeamOneScore: bundle.Match.TeamOneScore,
			TeamTwoScore: bundle.Match.TeamTwoScore,
		}
		last := projectRevealedRound(rb)
		dto.LastRoundResult = &last
	}

	return dto
}

func (s *Service) publishLeaveEvents(ctx context.Context, outcome *LeaveOutcome, reason string) {
	if s.events == nil || outcome == nil {
		return
	}
	if reason == "" {
		reason = AbandonReasonExplicitLeave
	}
	version := int64(0)
	if s.versions != nil {
		if v, err := s.versions.NextVersion(ctx, outcome.Match.ID); err == nil {
			version = v
			outcome.Version = v
		}
	}
	gameID := outcome.Match.GameID
	for _, uid := range outcome.AbandonedUserIDs {
		_ = s.events.PublishMatchEvent(ctx, outcome.Match.ID, EventMatchPlayerForfeited, version, &gameID, nil, map[string]any{
			"user_id": uid.String(),
			"reason":  reason,
		})
	}
	if outcome.Match.Result != nil {
		_ = s.events.PublishMatchEvent(ctx, outcome.Match.ID, EventMatchCompleted, version, &gameID, nil, map[string]any{
			"result":           *outcome.Match.Result,
			"winner_team_slot": outcome.Match.WinnerTeamSlot,
			"playlist":         playlistOf(outcome.Match),
		})
	}
}

func (s *Service) publishInactivityEvents(ctx context.Context, outcome *LeaveOutcome) {
	if s.events == nil || outcome == nil {
		return
	}
	version := int64(0)
	if s.versions != nil {
		if v, err := s.versions.NextVersion(ctx, outcome.Match.ID); err == nil {
			version = v
		}
	}
	gameID := outcome.Match.GameID
	_ = s.events.PublishMatchEvent(ctx, outcome.Match.ID, EventMatchCompleted, version, &gameID, nil, map[string]any{
		"result":   MatchResultAbandoned,
		"reason":   "inactivity",
		"playlist": playlistOf(outcome.Match),
	})
}

func (s *Service) restoreParties(ctx context.Context, outcome *LeaveOutcome) {
	if s.parties == nil || outcome == nil || len(outcome.RestoredPartyIDs) == 0 {
		return
	}
	if err := s.parties.RestoreAfterTerminalMatch(ctx, outcome.Match.ID, outcome.RestoredPartyIDs); err != nil {
		s.logger.WarnContext(ctx, "party restore after terminal match failed",
			slog.String("match_id", outcome.Match.ID.String()),
			slog.Any("error", err),
		)
		s.metrics.ObserveDependencyFailure("parties")
	}
}

func projectRevealedRound(bundle *RoundResultBundle) RevealedRoundResult {
	playerByID := make(map[uuid.UUID]GamePlayerRow, len(bundle.Players))
	for _, p := range bundle.Players {
		playerByID[p.ID] = p
	}
	partByPlayer := make(map[uuid.UUID]MatchParticipant, len(bundle.Participants))
	for _, p := range bundle.Participants {
		partByPlayer[p.GamePlayerID] = p
	}

	guesses := make([]RevealedGuessDTO, 0, len(bundle.Guesses))
	for _, g := range bundle.Guesses {
		dto := RevealedGuessDTO{
			GamePlayerID:  g.GamePlayerID,
			AccuracyScore: g.AccuracyScore,
			SpeedBonus:    g.SpeedBonus,
			Score:         g.Score,
			TimedOut:      g.TimedOut,
		}
		if !g.TimedOut {
			lat, lng, dist := g.Latitude, g.Longitude, g.DistanceMeters
			dto.Latitude = &lat
			dto.Longitude = &lng
			dto.DistanceMeters = &dist
			submitted := g.SubmittedAt
			dto.SubmittedAt = &submitted
		}
		if part, ok := partByPlayer[g.GamePlayerID]; ok {
			dto.UserID = part.UserID
			dto.TeamSlot = part.TeamSlot
		} else if gp, ok := playerByID[g.GamePlayerID]; ok && gp.UserID != nil {
			dto.UserID = *gp.UserID
			if gp.TeamSlot != nil {
				dto.TeamSlot = *gp.TeamSlot
			}
		}
		guesses = append(guesses, dto)
	}

	return RevealedRoundResult{
		RoundID:     bundle.Round.ID,
		RoundNumber: bundle.Round.RoundNumber,
		ActualLocation: RevealedLocationDTO{
			Latitude:    bundle.Answer.Latitude,
			Longitude:   bundle.Answer.Longitude,
			CountryCode: bundle.Answer.CountryCode,
			Region:      bundle.Answer.Region,
			Locality:    bundle.Answer.Locality,
		},
		Guesses: guesses,
		Teams: []TeamScoreProjection{
			{Slot: 1, Score: bundle.TeamOneScore},
			{Slot: 2, Score: bundle.TeamTwoScore},
		},
	}
}

func projectTerminalResult(bundle *TerminalResultBundle) *TerminalResultResponse {
	result := ""
	if bundle.Match.Result != nil {
		result = *bundle.Match.Result
	}
	teams := NewTeamProjections(
		bundle.Participants,
		bundle.Players,
		map[uuid.UUID]bool{},
		bundle.Match.TeamOneScore,
		bundle.Match.TeamTwoScore,
		nil, // fully revealed
	)
	rounds := make([]RevealedRoundResult, 0, len(bundle.Rounds))
	for i := range bundle.Rounds {
		rounds = append(rounds, projectRevealedRound(&bundle.Rounds[i]))
	}

	var progression ProgressionDTO
	if IsCasualMatch(bundle.Match) {
		progression = CasualNoProgression()
	} else if bundle.Match.ProgressionFinalizedAt == nil {
		// Should be gated by GetTerminalResult; defensive pending projection.
		progression = ProgressionPending()
	} else if bundle.ViewerProgression != nil {
		progression = *bundle.ViewerProgression
	} else {
		// Ranked finalized without loaded rating change rows (competitive not wired).
		progression = ProgressionDTO{Applied: true}
	}

	return &TerminalResultResponse{
		Result:         result,
		WinnerTeamSlot: bundle.Match.WinnerTeamSlot,
		Teams:          teams,
		Rounds:         rounds,
		Progression:    progression,
	}
}

func roundIsRevealed(bundle *SnapshotBundle) bool {
	if bundle.LastCompletedRound == nil || bundle.LastRoundAnswer == nil {
		return false
	}
	// Always allow last completed round reveal on snapshot (contract: after round closes).
	return true
}

func allowedSpectate(viewer MatchParticipant, bundle *SnapshotBundle, viewerSubmitted bool) []uuid.UUID {
	ids := AllowedSpectateTargets(BuildSpectatePolicyInput(viewer, bundle, viewerSubmitted))
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

// SpectateSelectOutcome is the service-level result of round.spectate.select.
type SpectateSelectOutcome struct {
	RoundID          uuid.UUID
	SelectedTargetID uuid.UUID
	AllowedTargetIDs []uuid.UUID
	Fallback         bool
	Format           string
}

// SelectSpectate evaluates preferred target against current policy and applies fallback.
func (s *Service) SelectSpectate(
	ctx context.Context,
	matchID, viewerUserID, preferredTargetGamePlayerID uuid.UUID,
) (SpectateSelectOutcome, error) {
	var zero SpectateSelectOutcome
	if s == nil || s.store == nil {
		return zero, ErrUnavailable
	}
	bundle, err := s.store.LoadSnapshotBundle(ctx, matchID)
	if err != nil {
		return zero, mapStoreErr(err)
	}
	if bundle == nil {
		return zero, ErrNotFound
	}
	var viewer *MatchParticipant
	for i := range bundle.Participants {
		if bundle.Participants[i].UserID == viewerUserID {
			viewer = &bundle.Participants[i]
			break
		}
	}
	if viewer == nil {
		return zero, ErrNotFound
	}
	submitted := bundle.SubmittedIDs != nil && bundle.SubmittedIDs[viewer.GamePlayerID]
	in := BuildSpectatePolicyInput(*viewer, bundle, submitted)
	format := in.Format
	if format == "" {
		format = formatOf(bundle.Match)
	}
	out := SpectateSelectOutcome{Format: format, AllowedTargetIDs: []uuid.UUID{}}
	if bundle.CurrentRound != nil {
		out.RoundID = bundle.CurrentRound.ID
	}
	if !CanViewerSpectate(submitted, in.RoundActive) {
		s.metrics.ObserveSpectate("denied", format)
		return out, ErrSpectateForbidden
	}
	allowed := AllowedSpectateTargets(in)
	if allowed == nil {
		allowed = []uuid.UUID{}
	}
	out.AllowedTargetIDs = allowed
	selected, fallback := SelectSpectateTarget(allowed, preferredTargetGamePlayerID)
	out.SelectedTargetID = selected
	out.Fallback = fallback
	if selected == uuid.Nil {
		s.metrics.ObserveSpectate("denied", format)
		return out, ErrSpectateForbidden
	}
	if fallback {
		s.metrics.ObserveSpectate("fallback", format)
	} else {
		s.metrics.ObserveSpectate("selected", format)
	}
	return out, nil
}

// ViewRecipients returns user IDs authorized to receive sourceUserID's provider-safe view updates.
func (s *Service) ViewRecipients(ctx context.Context, matchID, sourceUserID uuid.UUID) ([]uuid.UUID, error) {
	if s == nil || s.store == nil {
		return nil, ErrUnavailable
	}
	bundle, err := s.store.LoadSnapshotBundle(ctx, matchID)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	if bundle == nil {
		return nil, ErrNotFound
	}
	// Use a neutral viewer slot; SpectatorsOf re-evaluates per submitted player.
	in := BuildSpectatePolicyInput(MatchParticipant{}, bundle, false)
	return SpectatorsOf(in, sourceUserID), nil
}

func canChat(m Match, now time.Time) bool {
	if m.Format == FormatSolo || teamSizeOf(m) == 1 {
		return false
	}
	if !IsTerminalMatch(m.Status) {
		return true
	}
	if m.ChatAccessUntil != nil && now.Before(*m.ChatAccessUntil) {
		return true
	}
	return false
}

func findParticipant(parts []MatchParticipant, userID uuid.UUID) (MatchParticipant, bool) {
	for _, p := range parts {
		if p.UserID == userID {
			return p, true
		}
	}
	return MatchParticipant{}, false
}

func playlistOf(m Match) string {
	if m.Playlist != "" {
		return m.Playlist
	}
	if IsCasualMatch(m) {
		return PlaylistCasual
	}
	return PlaylistRanked
}

func formatOf(m Match) string {
	if m.Format != "" {
		return m.Format
	}
	return FormatSolo
}

func teamSizeOf(m Match) int {
	if m.TeamSize > 0 {
		return m.TeamSize
	}
	return 1
}

func requireRegistered(sess *session.Context) (uuid.UUID, error) {
	if sess == nil || !sess.IsRegistered() || sess.UserID == nil {
		return uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func mapStoreErr(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case ErrNotFound, ErrRoundNotRevealed, ErrMatchNotActive, ErrInvalidLeave, ErrUnavailable, ErrProgressionPending:
		return err
	default:
		return err
	}
}
