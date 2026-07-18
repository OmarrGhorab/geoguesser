package parties

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
	"github.com/raven/geoguess/backend/internal/matchmaking"
	"github.com/raven/geoguess/backend/internal/session"
)

// DefaultPartyInviteTTL is the default invite lifetime when config is omitted.
const DefaultPartyInviteTTL = 15 * time.Minute

// DefaultIdempotencyTTL is how long successful command snapshots are retained.
const DefaultIdempotencyTTL = 24 * time.Hour

// store is the persistence contract the service depends on.
type store interface {
	FindActiveUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
	HasActiveMatchAssignment(ctx context.Context, userID uuid.UUID) (bool, error)
	CreateParty(ctx context.Context, leaderID uuid.UUID, format string, now time.Time) (*PartySnapshot, error)
	GetActivePartyForUser(ctx context.Context, userID uuid.UUID) (*PartySnapshot, error)
	LoadSnapshot(ctx context.Context, partyID uuid.UUID) (*PartySnapshot, error)
	CreateInvite(ctx context.Context, partyID, leaderID, inviteeID uuid.UUID, expiresAt, now time.Time) (*InviteSnapshot, error)
	ListPendingInvitesForUser(ctx context.Context, inviteeID uuid.UUID, now time.Time) ([]InviteSnapshot, error)
	AcceptInvite(ctx context.Context, inviteID, inviteeID uuid.UUID, now time.Time) (*PartySnapshot, error)
	DeclineInvite(ctx context.Context, inviteID, inviteeID uuid.UUID, now time.Time) error
	SetReadiness(ctx context.Context, partyID, userID uuid.UUID, ready bool, now time.Time) (*PartySnapshot, error)
	LeaveParty(ctx context.Context, partyID, userID uuid.UUID, now time.Time) error
	KickMember(ctx context.Context, partyID, leaderID, targetID uuid.UUID, now time.Time) error
	DisbandParty(ctx context.Context, partyID, leaderID uuid.UUID, now time.Time) error
	GetInvite(ctx context.Context, inviteID uuid.UUID) (*PartyInvite, error)
	RestoreAfterTerminalMatch(ctx context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error
}

// FriendPolicy is the accepted-friend / block check used for invites.
type FriendPolicy interface {
	CanInvite(ctx context.Context, inviter, invitee uuid.UUID) error
}

// CommandIdempotencyStore is the optional Redis-backed command replay cache.
// Nil is allowed; when nil, commands execute without cross-request idempotency.
type CommandIdempotencyStore interface {
	Begin(ctx context.Context, scope, callerID, idemKey, bodyHash string, ttl time.Duration) (IdempotencyBeginResult, error)
	Complete(ctx context.Context, scope, callerID, idemKey, bodyHash string, response []byte, ttl time.Duration) error
	Release(ctx context.Context, scope, callerID, idemKey string) error
}

// PartyEventSink publishes post-commit party snapshots and lifecycle changes.
type PartyEventSink interface {
	Publish(ctx context.Context, partyID uuid.UUID, eventType string, version int64, payload any) error
}

type partyAudienceEventSink interface {
	PublishToUsers(ctx context.Context, partyID uuid.UUID, eventType string, version int64, userIDs []uuid.UUID, payload any) error
}

// IdempotencyBeginResult mirrors platform/redis BeginResult without importing it.
type IdempotencyBeginResult struct {
	Hit      bool
	Conflict bool
	InFlight bool
	Response []byte
}

// Config holds service-level party settings.
type Config struct {
	// CasualEnabled when false rejects mutating party commands. Default true in tests.
	CasualEnabled bool
	// InviteTTL is the pending invite lifetime. Default 15 minutes.
	InviteTTL time.Duration
	// IdempotencyTTL bounds stored command responses. Default 24h.
	IdempotencyTTL time.Duration
}

// Service implements party business rules.
type Service struct {
	repo    store
	friends FriendPolicy
	idem    CommandIdempotencyStore
	events  PartyEventSink
	metrics *Metrics
	logger  *slog.Logger
	cfg     Config
	clock   func() time.Time
}

// WithEvents attaches realtime party fanout.
func (s *Service) WithEvents(events PartyEventSink) *Service {
	s.events = events
	return s
}

// NewService returns a parties service.
func NewService(repo store, friendPolicy FriendPolicy, metrics *Metrics) *Service {
	return NewServiceWithOptions(repo, friendPolicy, metrics, nil, Config{CasualEnabled: true}, nil)
}

// NewServiceWithOptions returns a parties service with full dependency injection.
func NewServiceWithOptions(
	repo store,
	friendPolicy FriendPolicy,
	metrics *Metrics,
	idem CommandIdempotencyStore,
	cfg Config,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.InviteTTL <= 0 {
		cfg.InviteTTL = DefaultPartyInviteTTL
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = DefaultIdempotencyTTL
	}
	// CasualEnabled defaults true when zero-value struct is partially filled via NewService.
	// Explicit false is honored when set through WithCasualEnabled or Config from wiring.
	return &Service{
		repo:    repo,
		friends: friendPolicy,
		idem:    idem,
		metrics: metrics,
		logger:  logger,
		cfg:     cfg,
		clock:   func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the wall clock (tests).
func (s *Service) WithClock(clock func() time.Time) *Service {
	if clock != nil {
		s.clock = clock
	}
	return s
}

// WithIdempotency attaches a command idempotency store.
func (s *Service) WithIdempotency(store CommandIdempotencyStore) *Service {
	s.idem = store
	return s
}

// WithCasualEnabled toggles the optional feature flag gate.
func (s *Service) WithCasualEnabled(enabled bool) *Service {
	s.cfg.CasualEnabled = enabled
	return s
}

func (s *Service) requireRegistered(sess session.Context) (uuid.UUID, error) {
	if !sess.IsRegistered() {
		return uuid.Nil, ErrUnauthorized
	}
	id, err := uuid.Parse(*sess.UserID)
	if err != nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func (s *Service) requireActiveRegistered(ctx context.Context, sess session.Context) (uuid.UUID, error) {
	id, err := s.requireRegistered(sess)
	if err != nil {
		return uuid.Nil, err
	}
	active, err := s.repo.FindActiveUser(ctx, id)
	if err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			return uuid.Nil, ErrDependencyFailure
		}
		return uuid.Nil, err
	}
	if active == nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func (s *Service) requireFeature() error {
	// When CasualEnabled is false, refuse mutations. Reads remain available for recovery.
	if !s.cfg.CasualEnabled {
		return ErrFeatureDisabled
	}
	return nil
}

// Create creates a forming Duo or Squad party led by the caller.
func (s *Service) Create(ctx context.Context, sess session.Context, req CreatePartyRequest, idemKey string) (*PartyResponse, error) {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("create", mapErrorOutcome(err), start)
		return nil, err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("create", mapErrorOutcome(err), start)
		return nil, err
	}
	format := strings.TrimSpace(strings.ToLower(req.Format))
	if _, ok := CapacityForFormat(format); !ok {
		s.observeCommand("create", "validation_failed", start)
		return nil, ErrUnsupportedFormat
	}
	if strings.TrimSpace(idemKey) == "" {
		s.observeCommand("create", "validation_failed", start)
		return nil, ErrIdempotencyRequired
	}

	bodyHash := hashBody(req)
	if cached, done, err := s.beginIdem(ctx, "party.create", actor.String(), idemKey, bodyHash); err != nil {
		s.observeCommand("create", mapErrorOutcome(err), start)
		return nil, err
	} else if done {
		var resp PartyResponse
		if err := json.Unmarshal(cached, &resp); err != nil {
			s.observeCommand("create", "error", start)
			return nil, ErrDependencyFailure
		}
		s.observeCommand("create", "success", start)
		return &resp, nil
	}

	snap, err := s.repo.CreateParty(ctx, actor, format, s.clock())
	if err != nil {
		s.releaseIdem(ctx, "party.create", actor.String(), idemKey)
		outcome := mapErrorOutcome(err)
		s.observeCommand("create", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	resp := &PartyResponse{Party: toPartyDTO(*snap)}
	s.publishParty(ctx, resp.Party, EventPartyRosterUpdated)
	s.completeIdem(ctx, "party.create", actor.String(), idemKey, bodyHash, resp)
	s.observeCommand("create", "success", start)
	s.logger.InfoContext(ctx, "party created",
		slog.String("user_id", actor.String()),
		slog.String("party_id", snap.Party.ID.String()),
		slog.String("format", format),
	)
	return resp, nil
}

// Current returns the caller's active party or a null party field.
func (s *Service) Current(ctx context.Context, sess session.Context) (*CurrentPartyResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeRead("current", mapErrorOutcome(err))
		return nil, err
	}
	snap, err := s.repo.GetActivePartyForUser(ctx, actor)
	if err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			s.observeRead("current", "error")
			return nil, ErrDependencyFailure
		}
		s.observeRead("current", "error")
		return nil, err
	}
	s.observeRead("current", "success")
	if snap == nil {
		return &CurrentPartyResponse{Party: nil}, nil
	}
	dto := toPartyDTO(*snap)
	return &CurrentPartyResponse{Party: &dto}, nil
}

// AuthorizeRealtime reports whether userID is an active member of the party.
// Returns false without distinguishing missing vs unauthorized parties.
func (s *Service) AuthorizeRealtime(ctx context.Context, userID uuid.UUID, partyID string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrDependencyFailure
	}
	id, err := uuid.Parse(partyID)
	if err != nil {
		return false, nil
	}
	snap, err := s.repo.LoadSnapshot(ctx, id)
	if err != nil {
		if isDependency(err) {
			return false, ErrDependencyFailure
		}
		return false, nil
	}
	if snap == nil {
		return false, nil
	}
	return isActiveMember(*snap, userID), nil
}

// SnapshotForRealtime returns an authorized party DTO and version for WS connect.
func (s *Service) SnapshotForRealtime(ctx context.Context, userID uuid.UUID, partyID string) (any, int64, error) {
	ok, err := s.AuthorizeRealtime(ctx, userID, partyID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, ErrNotFound
	}
	id, err := uuid.Parse(partyID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	snap, err := s.repo.LoadSnapshot(ctx, id)
	if err != nil || snap == nil {
		return nil, 0, ErrNotFound
	}
	dto := toPartyDTO(*snap)
	return dto, int64(dto.Version), nil
}

// Get returns an authorized party snapshot for an active member.
func (s *Service) Get(ctx context.Context, sess session.Context, partyID string) (*PartyResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeRead("get", mapErrorOutcome(err))
		return nil, err
	}
	id, err := uuid.Parse(partyID)
	if err != nil {
		s.observeRead("get", "validation_failed")
		return nil, ErrInvalidPartyID
	}
	snap, err := s.repo.LoadSnapshot(ctx, id)
	if err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			s.observeRead("get", "error")
			return nil, ErrDependencyFailure
		}
		s.observeRead("get", mapErrorOutcome(err))
		return nil, err
	}
	if !isActiveMember(*snap, actor) {
		s.observeRead("get", "not_found")
		return nil, ErrNotFound
	}
	s.observeRead("get", "success")
	return &PartyResponse{Party: toPartyDTO(*snap)}, nil
}

// Invite creates a pending friend invite to join the caller's party.
func (s *Service) Invite(ctx context.Context, sess session.Context, partyID string, req InviteRequest, idemKey string) (*PartyInviteResponse, error) {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("invite", mapErrorOutcome(err), start)
		return nil, err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("invite", mapErrorOutcome(err), start)
		return nil, err
	}
	pid, err := uuid.Parse(partyID)
	if err != nil {
		s.observeCommand("invite", "validation_failed", start)
		return nil, ErrInvalidPartyID
	}
	invitee, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		s.observeCommand("invite", "validation_failed", start)
		return nil, ErrInvalidUserID
	}
	if invitee == actor {
		s.observeCommand("invite", "validation_failed", start)
		return nil, ErrSelfInvite
	}
	if strings.TrimSpace(idemKey) == "" {
		s.observeCommand("invite", "validation_failed", start)
		return nil, ErrIdempotencyRequired
	}

	bodyHash := hashBody(req)
	if cached, done, err := s.beginIdem(ctx, "party.invite", actor.String()+"|"+pid.String(), idemKey, bodyHash); err != nil {
		s.observeCommand("invite", mapErrorOutcome(err), start)
		return nil, err
	} else if done {
		var resp PartyInviteResponse
		if err := json.Unmarshal(cached, &resp); err != nil {
			s.observeCommand("invite", "error", start)
			return nil, ErrDependencyFailure
		}
		s.observeCommand("invite", "success", start)
		return &resp, nil
	}

	// Friendship / block policy before durable write (privacy-safe).
	if s.friends == nil {
		s.releaseIdem(ctx, "party.invite", actor.String()+"|"+pid.String(), idemKey)
		s.observeCommand("invite", "error", start)
		return nil, ErrDependencyFailure
	}
	if err := s.friends.CanInvite(ctx, actor, invitee); err != nil {
		s.releaseIdem(ctx, "party.invite", actor.String()+"|"+pid.String(), idemKey)
		if errors.Is(err, friends.ErrFriendUnavailable) {
			s.observeCommand("invite", "not_found", start)
			return nil, ErrFriendUnavailable
		}
		if isDependency(err) {
			s.observeDependency("postgres")
			s.observeCommand("invite", "error", start)
			return nil, ErrDependencyFailure
		}
		s.observeCommand("invite", "not_found", start)
		return nil, ErrFriendUnavailable
	}

	now := s.clock()
	expires := now.Add(s.cfg.InviteTTL)
	inviteSnap, err := s.repo.CreateInvite(ctx, pid, actor, invitee, expires, now)
	if err != nil {
		s.releaseIdem(ctx, "party.invite", actor.String()+"|"+pid.String(), idemKey)
		outcome := mapErrorOutcome(err)
		s.observeCommand("invite", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	resp := &PartyInviteResponse{Invite: toInviteDTO(*inviteSnap)}
	s.completeIdem(ctx, "party.invite", actor.String()+"|"+pid.String(), idemKey, bodyHash, resp)
	s.observeCommand("invite", "success", start)
	s.logger.InfoContext(ctx, "party invite created",
		slog.String("user_id", actor.String()),
		slog.String("party_id", pid.String()),
		slog.String("invite_id", inviteSnap.Invite.ID.String()),
	)
	return resp, nil
}

// ListInvites returns the caller's pending incoming invitations.
func (s *Service) ListInvites(ctx context.Context, sess session.Context) (*PartyInviteListResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeRead("invites", mapErrorOutcome(err))
		return nil, err
	}
	rows, err := s.repo.ListPendingInvitesForUser(ctx, actor, s.clock())
	if err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			s.observeRead("invites", "error")
			return nil, ErrDependencyFailure
		}
		s.observeRead("invites", "error")
		return nil, err
	}
	items := make([]PartyInviteDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toInviteDTO(row))
	}
	s.observeRead("invites", "success")
	return &PartyInviteListResponse{Invites: items}, nil
}

// AcceptInvite accepts a pending invitation and returns the joined party.
func (s *Service) AcceptInvite(ctx context.Context, sess session.Context, inviteID string, idemKey string) (*PartyResponse, error) {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("accept", mapErrorOutcome(err), start)
		return nil, err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("accept", mapErrorOutcome(err), start)
		return nil, err
	}
	id, err := uuid.Parse(inviteID)
	if err != nil {
		s.observeCommand("accept", "validation_failed", start)
		return nil, ErrInvalidInviteID
	}
	if strings.TrimSpace(idemKey) == "" {
		s.observeCommand("accept", "validation_failed", start)
		return nil, ErrIdempotencyRequired
	}

	bodyHash := hashBody(map[string]string{"invite_id": id.String()})
	if cached, done, err := s.beginIdem(ctx, "party.accept", actor.String(), idemKey, bodyHash); err != nil {
		s.observeCommand("accept", mapErrorOutcome(err), start)
		return nil, err
	} else if done {
		var resp PartyResponse
		if err := json.Unmarshal(cached, &resp); err != nil {
			s.observeCommand("accept", "error", start)
			return nil, ErrDependencyFailure
		}
		s.observeCommand("accept", "success", start)
		return &resp, nil
	}

	// Revalidate friendship/block before durable accept.
	invite, err := s.repo.GetInvite(ctx, id)
	if err != nil {
		s.releaseIdem(ctx, "party.accept", actor.String(), idemKey)
		if isDependency(err) {
			s.observeDependency("postgres")
			s.observeCommand("accept", "error", start)
			return nil, ErrDependencyFailure
		}
		s.observeCommand("accept", "error", start)
		return nil, err
	}
	if invite == nil || invite.InviteeUserID != actor {
		s.releaseIdem(ctx, "party.accept", actor.String(), idemKey)
		s.observeCommand("accept", "not_found", start)
		return nil, ErrInviteNotFound
	}
	if s.friends != nil {
		if err := s.friends.CanInvite(ctx, invite.InviterUserID, actor); err != nil {
			s.releaseIdem(ctx, "party.accept", actor.String(), idemKey)
			if errors.Is(err, friends.ErrFriendUnavailable) {
				s.observeCommand("accept", "not_found", start)
				return nil, ErrFriendUnavailable
			}
			if isDependency(err) {
				s.observeDependency("postgres")
				s.observeCommand("accept", "error", start)
				return nil, ErrDependencyFailure
			}
			s.observeCommand("accept", "not_found", start)
			return nil, ErrFriendUnavailable
		}
	}

	snap, err := s.repo.AcceptInvite(ctx, id, actor, s.clock())
	if err != nil {
		s.releaseIdem(ctx, "party.accept", actor.String(), idemKey)
		outcome := mapErrorOutcome(err)
		s.observeCommand("accept", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	resp := &PartyResponse{Party: toPartyDTO(*snap)}
	s.publishParty(ctx, resp.Party, EventPartyRosterUpdated)
	s.completeIdem(ctx, "party.accept", actor.String(), idemKey, bodyHash, resp)
	s.observeCommand("accept", "success", start)
	s.logger.InfoContext(ctx, "party invite accepted",
		slog.String("user_id", actor.String()),
		slog.String("invite_id", id.String()),
		slog.String("party_id", snap.Party.ID.String()),
	)
	return resp, nil
}

// DeclineInvite idempotently declines a pending invitation.
func (s *Service) DeclineInvite(ctx context.Context, sess session.Context, inviteID string) error {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("decline", mapErrorOutcome(err), start)
		return err
	}
	id, err := uuid.Parse(inviteID)
	if err != nil {
		s.observeCommand("decline", "validation_failed", start)
		return ErrInvalidInviteID
	}
	if err := s.repo.DeclineInvite(ctx, id, actor, s.clock()); err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("decline", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	s.observeCommand("decline", "success", start)
	return nil
}

// SetReadiness updates the caller's ready flag while forming.
func (s *Service) SetReadiness(ctx context.Context, sess session.Context, partyID string, req ReadinessRequest) (*PartyResponse, error) {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("readiness", mapErrorOutcome(err), start)
		return nil, err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("readiness", mapErrorOutcome(err), start)
		return nil, err
	}
	pid, err := uuid.Parse(partyID)
	if err != nil {
		s.observeCommand("readiness", "validation_failed", start)
		return nil, ErrInvalidPartyID
	}
	snap, err := s.repo.SetReadiness(ctx, pid, actor, req.Ready, s.clock())
	if err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("readiness", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	resp := &PartyResponse{Party: toPartyDTO(*snap)}
	s.publishParty(ctx, resp.Party, EventPartyReadinessChanged)
	s.observeCommand("readiness", "success", start)
	return resp, nil
}

// Leave removes the caller from a forming party.
func (s *Service) Leave(ctx context.Context, sess session.Context, partyID string) error {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("leave", mapErrorOutcome(err), start)
		return err
	}
	pid, err := uuid.Parse(partyID)
	if err != nil {
		s.observeCommand("leave", "validation_failed", start)
		return ErrInvalidPartyID
	}
	if err := s.repo.LeaveParty(ctx, pid, actor, s.clock()); err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("leave", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	s.publishLoadedParty(ctx, pid, EventPartyRosterUpdated, actor)
	s.observeCommand("leave", "success", start)
	s.logger.InfoContext(ctx, "party left",
		slog.String("user_id", actor.String()),
		slog.String("party_id", pid.String()),
	)
	return nil
}

// Kick removes another member; only the leader may kick.
func (s *Service) Kick(ctx context.Context, sess session.Context, partyID, targetUserID string) error {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("kick", mapErrorOutcome(err), start)
		return err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("kick", mapErrorOutcome(err), start)
		return err
	}
	pid, err := uuid.Parse(partyID)
	if err != nil {
		s.observeCommand("kick", "validation_failed", start)
		return ErrInvalidPartyID
	}
	target, err := uuid.Parse(targetUserID)
	if err != nil {
		s.observeCommand("kick", "validation_failed", start)
		return ErrInvalidUserID
	}
	if target == actor {
		s.observeCommand("kick", "validation_failed", start)
		return ErrInvalidRequest
	}
	if err := s.repo.KickMember(ctx, pid, actor, target, s.clock()); err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("kick", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	s.publishLoadedParty(ctx, pid, EventPartyRosterUpdated, target)
	s.observeCommand("kick", "success", start)
	s.logger.InfoContext(ctx, "party member kicked",
		slog.String("user_id", actor.String()),
		slog.String("party_id", pid.String()),
	)
	return nil
}

// Disband closes a forming party; only the leader may disband. Idempotent when already closed by the same leader.
func (s *Service) Disband(ctx context.Context, sess session.Context, partyID string) error {
	start := s.clock()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("disband", mapErrorOutcome(err), start)
		return err
	}
	if err := s.requireFeature(); err != nil {
		s.observeCommand("disband", mapErrorOutcome(err), start)
		return err
	}
	pid, err := uuid.Parse(partyID)
	if err != nil {
		s.observeCommand("disband", "validation_failed", start)
		return ErrInvalidPartyID
	}
	closedRecipients := s.partyRecipientIDs(ctx, pid)
	if err := s.repo.DisbandParty(ctx, pid, actor, s.clock()); err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("disband", outcome, start)
		if isDependency(err) {
			s.observeDependency("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	s.publishLoadedParty(ctx, pid, EventPartyClosed, closedRecipients...)
	s.observeCommand("disband", "success", start)
	s.logger.InfoContext(ctx, "party disbanded",
		slog.String("user_id", actor.String()),
		slog.String("party_id", pid.String()),
	)
	return nil
}

// ReadyPartyForQueue implements matchmaking.PartySnapshotReader.
// It returns the caller's complete, all-ready, leader-owned party when format and version match.
func (s *Service) ReadyPartyForQueue(ctx context.Context, leaderUserID uuid.UUID, format string, expectedVersion int64) (*matchmaking.PartyQueueSnapshot, error) {
	if leaderUserID == uuid.Nil {
		return nil, ErrUnauthorized
	}
	format = strings.TrimSpace(strings.ToLower(format))
	if _, ok := CapacityForFormat(format); !ok {
		return nil, ErrUnsupportedFormat
	}
	snap, err := s.repo.GetActivePartyForUser(ctx, leaderUserID)
	if err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	if snap == nil {
		return nil, nil
	}
	if snap.Party.LeaderUserID != leaderUserID {
		return nil, ErrLeaderRequired
	}
	if snap.Party.Status != StatusForming {
		return nil, ErrPartyLocked
	}
	if snap.Party.Format != format {
		return nil, ErrUnsupportedFormat
	}
	if int64(snap.Party.Version) != expectedVersion {
		return nil, ErrVersionMismatch
	}
	if !snap.IsComplete() {
		return nil, ErrPartyIncomplete
	}
	if !snap.AllReady() {
		return nil, ErrPartyNotReady
	}
	return &matchmaking.PartyQueueSnapshot{
		PartyID:   snap.Party.ID,
		Format:    snap.Party.Format,
		Capacity:  int(snap.Party.Capacity),
		Version:   int64(snap.Party.Version),
		LeaderID:  snap.Party.LeaderUserID,
		Status:    snap.Party.Status,
		MemberIDs: snap.ActiveUserIDs(),
		AllReady:  true,
	}, nil
}

// RestoreAfterTerminalMatch restores parties after a terminal match so members can requeue.
// Implements matchplay.PartyRestorer without importing matchplay (avoids package cycles
// at the service call sites; the method signature matches the restorer contract).
func (s *Service) RestoreAfterTerminalMatch(ctx context.Context, matchID uuid.UUID, partyIDs []uuid.UUID) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if err := s.repo.RestoreAfterTerminalMatch(ctx, matchID, partyIDs); err != nil {
		if isDependency(err) {
			s.observeDependency("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	for _, partyID := range partyIDs {
		s.publishLoadedParty(ctx, partyID, EventPartyQueueChanged)
	}
	return nil
}

func (s *Service) publishLoadedParty(ctx context.Context, partyID uuid.UUID, eventType string, additionalRecipients ...uuid.UUID) {
	if s == nil || s.events == nil || s.repo == nil {
		return
	}
	snap, err := s.repo.LoadSnapshot(ctx, partyID)
	if err != nil || snap == nil {
		return
	}
	s.publishParty(ctx, toPartyDTO(*snap), eventType, additionalRecipients...)
}

func (s *Service) publishParty(ctx context.Context, party PartyDTO, eventType string, additionalRecipients ...uuid.UUID) {
	if s == nil || s.events == nil {
		return
	}
	recipientSet := make(map[uuid.UUID]struct{}, len(party.Members)+len(additionalRecipients))
	for _, member := range party.Members {
		recipientSet[member.UserID] = struct{}{}
	}
	for _, userID := range additionalRecipients {
		if userID != uuid.Nil {
			recipientSet[userID] = struct{}{}
		}
	}
	recipients := make([]uuid.UUID, 0, len(recipientSet))
	for userID := range recipientSet {
		recipients = append(recipients, userID)
	}
	payload := map[string]any{"party": party}
	var err error
	if targeted, ok := s.events.(partyAudienceEventSink); ok {
		err = targeted.PublishToUsers(ctx, party.ID, eventType, int64(party.Version), recipients, payload)
	} else {
		err = s.events.Publish(ctx, party.ID, eventType, int64(party.Version), payload)
	}
	if err != nil {
		s.logger.WarnContext(ctx, "party event publish failed", slog.String("event_type", eventType), slog.Any("error", err))
	}
}

func (s *Service) partyRecipientIDs(ctx context.Context, partyID uuid.UUID) []uuid.UUID {
	if s == nil || s.repo == nil {
		return nil
	}
	snap, err := s.repo.LoadSnapshot(ctx, partyID)
	if err != nil || snap == nil {
		return nil
	}
	return snap.ActiveUserIDs()
}

// Ensure Service satisfies matchmaking.PartySnapshotReader.
var _ matchmaking.PartySnapshotReader = (*Service)(nil)

func isActiveMember(snap PartySnapshot, userID uuid.UUID) bool {
	for _, m := range snap.Members {
		if m.Member.UserID == userID && m.Member.Status == MemberStatusActive {
			return true
		}
	}
	return false
}

func (s *Service) beginIdem(ctx context.Context, scope, caller, key, bodyHash string) (cached []byte, hit bool, err error) {
	if s.idem == nil {
		return nil, false, nil
	}
	res, err := s.idem.Begin(ctx, scope, caller, key, bodyHash, s.cfg.IdempotencyTTL)
	if err != nil {
		if isDependency(err) {
			s.observeDependency("redis")
			return nil, false, ErrDependencyFailure
		}
		return nil, false, err
	}
	if res.Conflict || res.InFlight {
		return nil, false, ErrIdempotencyConflict
	}
	if res.Hit {
		return res.Response, true, nil
	}
	return nil, false, nil
}

func (s *Service) completeIdem(ctx context.Context, scope, caller, key, bodyHash string, resp any) {
	if s.idem == nil {
		return
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		_ = s.idem.Release(ctx, scope, caller, key)
		return
	}
	if err := s.idem.Complete(ctx, scope, caller, key, bodyHash, payload, s.cfg.IdempotencyTTL); err != nil {
		s.logger.InfoContext(ctx, "party idempotency complete failed", slog.String("scope", scope))
	}
}

func (s *Service) releaseIdem(ctx context.Context, scope, caller, key string) {
	if s.idem == nil {
		return
	}
	_ = s.idem.Release(ctx, scope, caller, key)
}

func hashBody(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "invalid"
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *Service) observeCommand(command, outcome string, start time.Time) {
	if s.metrics == nil {
		return
	}
	s.metrics.ObserveCommand(command, outcome, time.Since(start))
}

func (s *Service) observeRead(name, outcome string) {
	if s.metrics == nil {
		return
	}
	s.metrics.ObserveRead(name, outcome)
}

func (s *Service) observeDependency(dep string) {
	if s.metrics == nil {
		return
	}
	s.metrics.ObserveDependencyFailure(dep)
}

func mapErrorOutcome(err error) string {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrInviteNotFound), errors.Is(err, ErrFriendUnavailable):
		return "not_found"
	case errors.Is(err, ErrActivePartyConflict), errors.Is(err, ErrActiveQueueOrMatch),
		errors.Is(err, ErrPartyFull), errors.Is(err, ErrAlreadyInvited), errors.Is(err, ErrTargetBusy),
		errors.Is(err, ErrPartyLocked), errors.Is(err, ErrIdempotencyConflict):
		return "conflict"
	case errors.Is(err, ErrLeaderRequired):
		return "forbidden"
	case errors.Is(err, ErrUnsupportedFormat), errors.Is(err, ErrPartyIncomplete),
		errors.Is(err, ErrPartyNotReady), errors.Is(err, ErrVersionMismatch):
		return "unprocessable"
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidPartyID),
		errors.Is(err, ErrInvalidInviteID), errors.Is(err, ErrInvalidUserID),
		errors.Is(err, ErrSelfInvite), errors.Is(err, ErrIdempotencyRequired),
		errors.Is(err, ErrInvalidJSON):
		return "validation_failed"
	case errors.Is(err, ErrDependencyFailure), errors.Is(err, ErrFeatureDisabled):
		return "unavailable"
	default:
		return "error"
	}
}

func isDependency(err error) bool {
	if err == nil {
		return false
	}
	known := []error{
		ErrUnauthorized, ErrNotFound, ErrInviteNotFound, ErrActivePartyConflict, ErrActiveQueueOrMatch,
		ErrUnsupportedFormat, ErrLeaderRequired, ErrFriendUnavailable, ErrPartyFull, ErrAlreadyInvited,
		ErrTargetBusy, ErrPartyLocked, ErrPartyIncomplete, ErrPartyNotReady, ErrInvalidRequest,
		ErrInvalidJSON, ErrInvalidPartyID, ErrInvalidInviteID, ErrInvalidUserID, ErrSelfInvite,
		ErrIdempotencyConflict, ErrIdempotencyRequired, ErrDependencyFailure, ErrRateLimited,
		ErrVersionMismatch, ErrFeatureDisabled, friends.ErrFriendUnavailable,
	}
	for _, k := range known {
		if errors.Is(err, k) {
			return false
		}
	}
	return true
}
