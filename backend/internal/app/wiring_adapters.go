package app

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/games"
	"github.com/raven/geoguess/backend/internal/matchplay"
	"github.com/raven/geoguess/backend/internal/parties"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
	"github.com/raven/geoguess/backend/internal/realtime"
)

// partyIdemAdapter maps platform Redis command idempotency to parties.CommandIdempotencyStore.
type partyIdemAdapter struct {
	inner *redisplatform.CommandIdempotencyStore
}

// GameOutcomeEventAdapter maps game-owned post-commit transitions to the match
// realtime channel without introducing a games -> matchplay package dependency.
type GameOutcomeEventAdapter struct {
	resolver interface {
		FindMatchIDByGameID(context.Context, uuid.UUID) (uuid.UUID, error)
	}
	events   *matchplay.Publisher
	versions *redisplatform.RealtimeStore
}

// NewGameOutcomeEventAdapter constructs the game-to-match event bridge.
func NewGameOutcomeEventAdapter(resolver interface {
	FindMatchIDByGameID(context.Context, uuid.UUID) (uuid.UUID, error)
}, events *matchplay.Publisher, versions *redisplatform.RealtimeStore) *GameOutcomeEventAdapter {
	return &GameOutcomeEventAdapter{resolver: resolver, events: events, versions: versions}
}

// PublishMultiplayerOutcome implements games.MultiplayerEventSink.
func (a *GameOutcomeEventAdapter) PublishMultiplayerOutcome(ctx context.Context, gameID uuid.UUID, outcome games.MultiplayerGuessOutcome) error {
	if a == nil || a.resolver == nil || a.events == nil || !outcome.RoundCompleted {
		return nil
	}
	matchID, err := a.resolver.FindMatchIDByGameID(ctx, gameID)
	if err != nil || matchID == uuid.Nil {
		return err
	}
	nextVersion := func() int64 {
		if a.versions == nil {
			return 0
		}
		version, _ := a.versions.IncrChannelVersion(ctx, realtime.ChannelKindMatch, matchID.String())
		return version
	}
	gid, rid := gameID, outcome.RoundID
	payload := map[string]any{
		"round_id": outcome.RoundID.String(), "submitted_count": outcome.SubmittedCount,
		"eligible_count": outcome.EligibleCount,
	}
	if err := a.events.PublishMatchEvent(ctx, matchID, matchplay.EventRoundEnded, nextVersion(), &gid, &rid, payload); err != nil {
		return err
	}
	if err := a.events.PublishMatchEvent(ctx, matchID, matchplay.EventRoundResultsRevealed, nextVersion(), &gid, &rid, payload); err != nil {
		return err
	}
	if outcome.GameCompleted {
		return a.events.PublishMatchCompleted(ctx, matchID, gameID, nextVersion(), map[string]any{"result_ready": true})
	}
	if outcome.NextRoundID != nil {
		nextPayload := map[string]any{"round_id": outcome.NextRoundID.String()}
		if outcome.NextRoundNumber != nil {
			nextPayload["round_number"] = *outcome.NextRoundNumber
		}
		nextID := *outcome.NextRoundID
		return a.events.PublishMatchEvent(ctx, matchID, matchplay.EventRoundStarted, nextVersion(), &gid, &nextID, nextPayload)
	}
	return nil
}

// NewPartyIdempotencyAdapter wraps the Redis command-idempotency store for parties.
func NewPartyIdempotencyAdapter(store *redisplatform.CommandIdempotencyStore) parties.CommandIdempotencyStore {
	if store == nil {
		return nil
	}
	return partyIdemAdapter{inner: store}
}

func (a partyIdemAdapter) Begin(
	ctx context.Context,
	scope, callerID, idemKey, bodyHash string,
	ttl time.Duration,
) (parties.IdempotencyBeginResult, error) {
	res, err := a.inner.Begin(ctx, scope, callerID, idemKey, bodyHash, ttl)
	if err != nil {
		return parties.IdempotencyBeginResult{}, err
	}
	return parties.IdempotencyBeginResult{
		Hit:      res.Hit,
		Conflict: res.Conflict,
		InFlight: res.InFlight,
		Response: res.Response,
	}, nil
}

func (a partyIdemAdapter) Complete(
	ctx context.Context,
	scope, callerID, idemKey, bodyHash string,
	response []byte,
	ttl time.Duration,
) error {
	return a.inner.Complete(ctx, scope, callerID, idemKey, bodyHash, response, ttl)
}

func (a partyIdemAdapter) Release(ctx context.Context, scope, callerID, idemKey string) error {
	return a.inner.Release(ctx, scope, callerID, idemKey)
}

// matchVersionAdapter maps RealtimeStore channel versions to matchplay.VersionStore.
type matchVersionAdapter struct {
	inner *redisplatform.RealtimeStore
}

// NewMatchVersionAdapter wraps the Redis realtime store for match channel versions.
func NewMatchVersionAdapter(store *redisplatform.RealtimeStore) matchplay.VersionStore {
	if store == nil {
		return nil
	}
	return matchVersionAdapter{inner: store}
}

func (a matchVersionAdapter) NextVersion(ctx context.Context, matchID uuid.UUID) (int64, error) {
	return a.inner.IncrChannelVersion(ctx, "match", matchID.String())
}

func (a matchVersionAdapter) CurrentVersion(ctx context.Context, matchID uuid.UUID) (int64, error) {
	return a.inner.GetChannelVersion(ctx, "match", matchID.String())
}

// matchPresenceAdapter maps RealtimeStore presence/reconnect keys to matchplay.PresenceGraceStore.
type matchPresenceAdapter struct {
	inner *redisplatform.RealtimeStore
}

// NewMatchPresenceAdapter wraps the Redis realtime store for match reconnect grace.
func NewMatchPresenceAdapter(store *redisplatform.RealtimeStore) matchplay.PresenceGraceStore {
	if store == nil {
		return nil
	}
	return matchPresenceAdapter{inner: store}
}

func (a matchPresenceAdapter) HasReconnectWindow(ctx context.Context, matchID, userID uuid.UUID) (bool, error) {
	_, ok, err := a.inner.GetReconnectWindow(ctx, "match", matchID.String(), userID)
	return ok, err
}

func (a matchPresenceAdapter) ClearPresence(ctx context.Context, matchID, userID uuid.UUID) error {
	if err := a.inner.DeletePresence(ctx, "match", matchID.String(), userID); err != nil {
		return err
	}
	return a.inner.DeleteReconnectWindow(ctx, "match", matchID.String(), userID)
}

// --- Realtime ticket store adapter ---

// NewRealtimeTicketStore adapts Redis RealtimeStore to realtime.TicketStore.
func NewRealtimeTicketStore(store *redisplatform.RealtimeStore) realtime.TicketStore {
	if store == nil {
		return nil
	}
	return realtime.RealtimeTicketAdapter{
		IssueFn: store.IssueTicket,
		ConsumeFn: func(ctx context.Context, token string) (realtime.TicketClaims, error) {
			claims, err := store.ConsumeTicket(ctx, token)
			if err != nil {
				switch {
				case errors.Is(err, redisplatform.ErrTicketUsed):
					return realtime.TicketClaims{}, realtime.ErrTicketUsed
				case errors.Is(err, redisplatform.ErrTicketExpired):
					return realtime.TicketClaims{}, realtime.ErrTicketExpired
				case errors.Is(err, redisplatform.ErrTicketInvalid):
					return realtime.TicketClaims{}, realtime.ErrTicketInvalid
				default:
					return realtime.TicketClaims{}, err
				}
			}
			return realtime.TicketClaims{
				UserID:      claims.UserID,
				ChannelKind: claims.ChannelKind,
				ChannelID:   claims.ChannelID,
				ExpiresAt:   claims.ExpiresAt,
			}, nil
		},
	}
}

// --- Channel authorizer / snapshot / command adapters ---

// RealtimeChannelServices bundles matchplay + parties lookups for ticket/WS auth.
type RealtimeChannelServices struct {
	Matchplay *matchplay.Service
	Parties   *parties.Service
	Live      *redisplatform.MatchLiveStore
	Metrics   matchplay.MetricsRecorder
}

// RealtimeConnectionLifecycle coordinates global Redis socket counts with the
// durable match participant connection marker used by reconnect/forfeit workers.
type RealtimeConnectionLifecycle struct {
	store *redisplatform.RealtimeStore
	match *matchplay.Service
	grace time.Duration
}

// NewRealtimeConnectionLifecycle constructs the socket transition adapter.
func NewRealtimeConnectionLifecycle(store *redisplatform.RealtimeStore, match *matchplay.Service, grace time.Duration) *RealtimeConnectionLifecycle {
	if grace <= 0 {
		grace = matchplay.DefaultReconnectGrace
	}
	return &RealtimeConnectionLifecycle{store: store, match: match, grace: grace}
}

// Connected implements realtime.ConnectionLifecycle.
func (a *RealtimeConnectionLifecycle) Connected(ctx context.Context, channel realtime.ChannelRef, userID uuid.UUID, _ int64) error {
	if a == nil || a.store == nil {
		return errors.New("realtime presence unavailable")
	}
	count, err := a.store.AcquirePresence(ctx, channel.Kind, channel.ID, userID, a.grace)
	if err != nil || count != 1 || channel.Kind != realtime.ChannelKindMatch || a.match == nil {
		return err
	}
	matchID, err := uuid.Parse(channel.ID)
	if err != nil {
		return err
	}
	return a.match.MarkRealtimeConnected(ctx, matchID, userID)
}

// Heartbeat implements realtime.ConnectionLifecycle and recovers disposable
// counts after a Redis restart without dropping the authorized socket.
func (a *RealtimeConnectionLifecycle) Heartbeat(ctx context.Context, channel realtime.ChannelRef, userID uuid.UUID) error {
	if a == nil || a.store == nil {
		return errors.New("realtime presence unavailable")
	}
	count, err := a.store.RefreshPresence(ctx, channel.Kind, channel.ID, userID, a.grace)
	if err != nil || count > 0 {
		return err
	}
	return a.Connected(ctx, channel, userID, 0)
}

// Disconnected implements realtime.ConnectionLifecycle.
func (a *RealtimeConnectionLifecycle) Disconnected(ctx context.Context, channel realtime.ChannelRef, userID uuid.UUID, lastVersion int64) error {
	if a == nil || a.store == nil {
		return errors.New("realtime presence unavailable")
	}
	remaining, err := a.store.ReleasePresence(ctx, channel.Kind, channel.ID, userID, lastVersion, a.grace)
	if err != nil || remaining != 0 || channel.Kind != realtime.ChannelKindMatch || a.match == nil {
		return err
	}
	matchID, err := uuid.Parse(channel.ID)
	if err != nil {
		return err
	}
	if err := a.match.MarkRealtimeDisconnected(ctx, matchID, userID); err != nil {
		return err
	}
	// A reconnect can win Redis between the final-socket release and the DB
	// update. Re-check the global count and repair the durable marker so the
	// grace worker never forfeits an already reconnected participant.
	count, err := a.store.RefreshPresence(ctx, channel.Kind, channel.ID, userID, a.grace)
	if err != nil || count == 0 {
		return err
	}
	return a.match.MarkRealtimeConnected(ctx, matchID, userID)
}

// Authorize implements realtime.ChannelAuthorizer.
func (s RealtimeChannelServices) Authorize(ctx context.Context, userID uuid.UUID, channelKind, channelID string) (realtime.ChannelMembership, error) {
	var zero realtime.ChannelMembership
	switch channelKind {
	case realtime.ChannelKindMatch:
		if s.Matchplay == nil {
			return zero, realtime.ErrNotParticipant
		}
		matchID, err := uuid.Parse(channelID)
		if err != nil {
			return zero, realtime.ErrNotParticipant
		}
		// Use privacy-safe snapshot path: non-participants get ErrNotFound.
		// We only need team slot — load via GetSnapshot with a synthetic session.
		// Prefer a lightweight participant lookup when available through service store.
		part, err := s.Matchplay.AuthorizeRealtime(ctx, userID, matchID)
		if err != nil {
			return zero, realtime.ErrNotParticipant
		}
		slot := part.TeamSlot
		return realtime.ChannelMembership{UserID: userID, TeamSlot: &slot}, nil
	case realtime.ChannelKindParty:
		if s.Parties == nil {
			return zero, realtime.ErrNotParticipant
		}
		ok, err := s.Parties.AuthorizeRealtime(ctx, userID, channelID)
		if err != nil || !ok {
			return zero, realtime.ErrNotParticipant
		}
		return realtime.ChannelMembership{UserID: userID}, nil
	default:
		return zero, realtime.ErrNotParticipant
	}
}

// Snapshot implements realtime.SnapshotProvider.
func (s RealtimeChannelServices) Snapshot(ctx context.Context, userID uuid.UUID, channelKind, channelID string) (any, int64, error) {
	switch channelKind {
	case realtime.ChannelKindMatch:
		if s.Matchplay == nil {
			return map[string]any{}, 0, nil
		}
		matchID, err := uuid.Parse(channelID)
		if err != nil {
			return nil, 0, err
		}
		resp, err := s.Matchplay.SnapshotForRealtime(ctx, userID, matchID)
		if err != nil {
			return nil, 0, err
		}
		if resp == nil {
			return map[string]any{}, 0, nil
		}
		return resp.Match, resp.Match.RealtimeVersion, nil
	case realtime.ChannelKindParty:
		if s.Parties == nil {
			return map[string]any{}, 0, nil
		}
		snap, ver, err := s.Parties.SnapshotForRealtime(ctx, userID, channelID)
		if err != nil {
			return nil, 0, err
		}
		return snap, ver, nil
	default:
		return map[string]any{}, 0, nil
	}
}

// LiveCommandAdapter implements realtime.MatchCommandService.
type LiveCommandAdapter struct {
	Live      *redisplatform.MatchLiveStore
	Matchplay *matchplay.Service
	Metrics   matchplay.MetricsRecorder
}

// NewLiveCommandAdapter constructs the Redis-backed match command service.
func NewLiveCommandAdapter(live *redisplatform.MatchLiveStore, metrics matchplay.MetricsRecorder) *LiveCommandAdapter {
	if metrics == nil {
		metrics = matchplay.NoopMetrics{}
	}
	return &LiveCommandAdapter{Live: live, Metrics: metrics}
}

// WithMatchplay attaches snapshot/spectate policy for select and view fanout authorization.
func (a *LiveCommandAdapter) WithMatchplay(svc *matchplay.Service) *LiveCommandAdapter {
	if a == nil {
		return a
	}
	a.Matchplay = svc
	return a
}

func (a *LiveCommandAdapter) SetMarker(ctx context.Context, matchID, roundID, userID uuid.UUID, teamSlot int, lat, lng float64) (int64, error) {
	if a == nil || a.Live == nil {
		return 0, errors.New("live state unavailable")
	}
	rec, err := a.Live.SetMarker(ctx, matchID.String(), roundID.String(), teamSlot, userID, lat, lng, redisplatform.DefaultRoundLiveTTL)
	if err != nil {
		if errors.Is(err, redisplatform.ErrMarkerThrottled) {
			a.Metrics.ObserveMarker("throttled")
			return 0, realtime.WrapThrottleError(err)
		}
		if errors.Is(err, redisplatform.ErrPlayerLocked) {
			a.Metrics.ObserveMarker("locked")
			return 0, realtime.WrapLockedError(err)
		}
		a.Metrics.ObserveMarker("error")
		return 0, err
	}
	a.Metrics.ObserveMarker("ok")
	return rec.Version, nil
}

func (a *LiveCommandAdapter) SetView(ctx context.Context, matchID, roundID, userID uuid.UUID, panoramaID string, heading, pitch, zoom float64) (int64, error) {
	if a == nil || a.Live == nil {
		return 0, errors.New("live state unavailable")
	}
	rec, err := a.Live.SetView(ctx, matchID.String(), roundID.String(), userID, redisplatform.ViewRecord{
		PanoramaID: panoramaID,
		Heading:    heading,
		Pitch:      pitch,
		Zoom:       zoom,
	}, redisplatform.DefaultRoundLiveTTL)
	if err != nil {
		if errors.Is(err, redisplatform.ErrViewThrottled) {
			a.Metrics.ObserveViewUpdate("throttled", "unknown")
			return 0, realtime.WrapThrottleError(err)
		}
		if errors.Is(err, redisplatform.ErrViewUnchanged) {
			a.Metrics.ObserveViewUpdate("unchanged", "unknown")
			return rec.Version, realtime.WrapViewUnchanged(err)
		}
		if errors.Is(err, redisplatform.ErrViewForbiddenFields) {
			a.Metrics.ObserveViewUpdate("rejected", "unknown")
			return 0, err
		}
		a.Metrics.ObserveViewUpdate("error", "unknown")
		return 0, err
	}
	a.Metrics.ObserveViewUpdate("ok", "unknown")
	return rec.Version, nil
}

func (a *LiveCommandAdapter) SelectSpectate(ctx context.Context, matchID, viewerUserID, preferredTargetGamePlayerID uuid.UUID) (realtime.SpectateSelectResult, error) {
	var zero realtime.SpectateSelectResult
	if a == nil || a.Matchplay == nil {
		return zero, realtime.WrapSpectateForbidden(errors.New("spectate policy unavailable"))
	}
	outcome, err := a.Matchplay.SelectSpectate(ctx, matchID, viewerUserID, preferredTargetGamePlayerID)
	if err != nil {
		if errors.Is(err, matchplay.ErrSpectateForbidden) {
			return realtime.SpectateSelectResult{
				RoundID:          outcome.RoundID,
				AllowedTargetIDs: outcome.AllowedTargetIDs,
				Fallback:         outcome.Fallback,
				Format:           outcome.Format,
			}, realtime.WrapSpectateForbidden(err)
		}
		return zero, err
	}
	// Bump live version so clients can order spectate_target_changed with other events.
	version := int64(0)
	if a.Live != nil {
		if v, vErr := a.Live.IncrVersion(ctx, matchID.String()); vErr == nil {
			version = v
		}
	}
	return realtime.SpectateSelectResult{
		RoundID:          outcome.RoundID,
		SelectedTargetID: outcome.SelectedTargetID,
		AllowedTargetIDs: outcome.AllowedTargetIDs,
		Fallback:         outcome.Fallback,
		Format:           outcome.Format,
		Version:          version,
	}, nil
}

func (a *LiveCommandAdapter) ViewRecipients(ctx context.Context, matchID, roundID, sourceUserID uuid.UUID) ([]uuid.UUID, error) {
	if a == nil || a.Matchplay == nil {
		return nil, nil
	}
	_ = roundID // policy uses current active round from durable snapshot
	return a.Matchplay.ViewRecipients(ctx, matchID, sourceUserID)
}

func (a *LiveCommandAdapter) Heartbeat(ctx context.Context, matchID, userID uuid.UUID) error {
	if a == nil || a.Live == nil {
		return nil
	}
	return a.Live.SetPresence(ctx, matchID.String(), userID, "connected", redisplatform.DefaultMatchPresenceTTL)
}

func (a *LiveCommandAdapter) CurrentVersion(ctx context.Context, matchID uuid.UUID) (int64, error) {
	if a == nil || a.Live == nil {
		return 0, nil
	}
	return a.Live.GetVersion(ctx, matchID.String())
}

func (a *LiveCommandAdapter) LoadCommandAck(ctx context.Context, matchID, userID uuid.UUID, commandID string) ([]byte, bool, error) {
	if a == nil || a.Live == nil {
		return nil, false, nil
	}
	ack, err := a.Live.GetCommandAck(ctx, matchID.String(), userID, commandID)
	if err != nil {
		return nil, false, err
	}
	if ack == nil {
		return nil, false, nil
	}
	if len(ack.Payload) > 0 {
		return []byte(ack.Payload), true, nil
	}
	raw, err := json.Marshal(ack)
	return raw, true, err
}

func (a *LiveCommandAdapter) SaveCommandAck(ctx context.Context, matchID, userID uuid.UUID, commandID string, ackJSON []byte) error {
	if a == nil || a.Live == nil {
		return nil
	}
	_, _, err := a.Live.StoreCommandAck(ctx, matchID.String(), userID, redisplatform.CommandAck{
		CommandID: commandID,
		OK:        true,
		Payload:   json.RawMessage(ackJSON),
	}, redisplatform.DefaultCommandAckTTL)
	return err
}

var _ realtime.MatchCommandService = (*LiveCommandAdapter)(nil)
var _ realtime.ChannelAuthorizer = RealtimeChannelServices{}
var _ realtime.SnapshotProvider = RealtimeChannelServices{}

// --- Storage / upload adapters for chat wiring ---

// storagePresigner is satisfied by platform/storage.Provider.
type storagePresigner interface {
	PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	DeleteObject(ctx context.Context, key string) error
}

// AttachmentSignerAdapter wraps storage for matchplay chat signed downloads.
type AttachmentSignerAdapter struct {
	inner storagePresigner
}

// NewAttachmentSignerAdapter constructs a matchplay.AttachmentSigner.
func NewAttachmentSignerAdapter(inner storagePresigner) matchplay.AttachmentSigner {
	if inner == nil {
		return nil
	}
	return AttachmentSignerAdapter{inner: inner}
}

func (a AttachmentSignerAdapter) PresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	return a.inner.PresignedDownloadURL(ctx, key, expiresIn)
}

// ObjectDeleterAdapter wraps storage deletes for chat retention cleanup.
type ObjectDeleterAdapter struct {
	inner storagePresigner
}

// NewObjectDeleterAdapter constructs a matchplay.ObjectDeleter.
func NewObjectDeleterAdapter(inner storagePresigner) matchplay.ObjectDeleter {
	if inner == nil {
		return nil
	}
	return ObjectDeleterAdapter{inner: inner}
}

func (a ObjectDeleterAdapter) DeleteObject(ctx context.Context, key string) error {
	return a.inner.DeleteObject(ctx, key)
}

// MatchUploadAccessAdapter authorizes team-chat uploads via matchplay participation.
type MatchUploadAccessAdapter struct {
	svc *matchplay.Service
}

// NewMatchUploadAccessAdapter constructs an uploads.MatchAccessChecker.
func NewMatchUploadAccessAdapter(svc *matchplay.Service) *MatchUploadAccessAdapter {
	return &MatchUploadAccessAdapter{svc: svc}
}

// AuthorizeTeamChatUpload returns nil when the user is a match participant.
func (a *MatchUploadAccessAdapter) AuthorizeTeamChatUpload(ctx context.Context, matchID, userID uuid.UUID) error {
	if a == nil || a.svc == nil {
		return matchplay.ErrUnavailable
	}
	_, err := a.svc.AuthorizeRealtime(ctx, userID, matchID)
	return err
}
