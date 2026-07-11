package friends

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
	"github.com/raven/geoguess/backend/internal/session"
)

// store is the persistence contract the service depends on.
type store interface {
	FindActiveUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
	CreateRequest(ctx context.Context, requester, target uuid.UUID) (*Friendship, *PublicProfile, error)
	AcceptRequest(ctx context.Context, requestID, acceptor uuid.UUID) (*Friendship, *PublicProfile, error)
	DeclineRequest(ctx context.Context, requestID, actor uuid.UUID) error
	RemoveFriendship(ctx context.Context, actor, other uuid.UUID) error
	BlockUser(ctx context.Context, blocker, target uuid.UUID) error
	UnblockUser(ctx context.Context, actor, other uuid.UUID) error
	ListIncomingRequests(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error)
	ListOutgoingRequests(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error)
	ListAcceptedFriends(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error)
	ListBlockedUsers(ctx context.Context, viewer uuid.UUID, limit int, cursor string) (*Page, error)
	LoadPublicProfile(ctx context.Context, userID uuid.UUID) (*PublicProfile, error)
}

// Service implements friends business rules.
type Service struct {
	repo    store
	metrics *Metrics
	logger  *slog.Logger
}

// NewService returns a friends service.
func NewService(repo store, metrics *Metrics) *Service {
	return NewServiceWithLogger(repo, metrics, nil)
}

// NewServiceWithLogger returns a friends service with structured logging.
func NewServiceWithLogger(repo store, metrics *Metrics, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, metrics: metrics, logger: logger}
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

// requireActiveRegistered enforces FR-001: registered JWT is not enough; the
// account must still be active in PostgreSQL for every social command and read.
func (s *Service) requireActiveRegistered(ctx context.Context, sess session.Context) (uuid.UUID, error) {
	id, err := s.requireRegistered(sess)
	if err != nil {
		return uuid.Nil, err
	}
	active, err := s.repo.FindActiveUser(ctx, id)
	if err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			return uuid.Nil, ErrDependencyFailure
		}
		return uuid.Nil, err
	}
	if active == nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

// CreateRequest sends a friend request to targetUserID.
func (s *Service) CreateRequest(ctx context.Context, sess session.Context, targetUserID string) (*RequestResponse, error) {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("request", mapErrorOutcome(err), start)
		return nil, err
	}
	target, err := uuid.Parse(targetUserID)
	if err != nil {
		s.observeCommand("request", "validation_failed", start)
		return nil, ErrInvalidUserID
	}
	if actor == target {
		s.observeCommand("request", "validation_failed", start)
		return nil, ErrSelfPair
	}

	f, profile, err := s.repo.CreateRequest(ctx, actor, target)
	if err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("request", outcome, start)
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, ErrDependencyFailure
		}
		s.logger.InfoContext(ctx, "friend request failed",
			slog.String("user_id", actor.String()),
			slog.String("outcome", outcome),
		)
		return nil, err
	}

	s.observeCommand("request", "success", start)
	s.logger.InfoContext(ctx, "friend request created",
		slog.String("user_id", actor.String()),
		slog.String("request_id", f.ID.String()),
	)
	return &RequestResponse{Data: FriendRequestDTO{
		RequestID: f.ID,
		User:      toPublicUserDTO(*profile),
		CreatedAt: f.CreatedAt.UTC(),
		Direction: "outgoing",
	}}, nil
}

// ListIncoming returns pending incoming requests.
func (s *Service) ListIncoming(ctx context.Context, sess session.Context, limit int, cursor string) (*RequestListResponse, error) {
	return s.listRequests(ctx, sess, limit, cursor, "incoming", s.repo.ListIncomingRequests)
}

// ListOutgoing returns pending outgoing requests.
func (s *Service) ListOutgoing(ctx context.Context, sess session.Context, limit int, cursor string) (*RequestListResponse, error) {
	return s.listRequests(ctx, sess, limit, cursor, "outgoing", s.repo.ListOutgoingRequests)
}

func (s *Service) listRequests(
	ctx context.Context,
	sess session.Context,
	limit int,
	cursor string,
	direction string,
	listFn func(context.Context, uuid.UUID, int, string) (*Page, error),
) (*RequestListResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.metrics.ObserveList(direction, mapErrorOutcome(err))
		return nil, err
	}
	limit, err = normalizeLimit(limit)
	if err != nil {
		s.metrics.ObserveList(direction, "validation_failed")
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		s.metrics.ObserveList(direction, "validation_failed")
		return nil, err
	}
	page, err := listFn(ctx, actor, limit, cursor)
	if err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.metrics.ObserveList(direction, "error")
			return nil, ErrDependencyFailure
		}
		s.metrics.ObserveList(direction, "error")
		return nil, err
	}
	items := make([]FriendRequestDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toRequestDTO(row, direction))
	}
	s.metrics.ObserveList(direction, "success")
	return &RequestListResponse{
		Data: items,
		Page: apphttp.PageInfo{Limit: page.Limit, NextCursor: page.NextCursor},
	}, nil
}

// AcceptRequest accepts a pending friend request.
func (s *Service) AcceptRequest(ctx context.Context, sess session.Context, requestID string) (*FriendshipResponse, error) {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("accept", mapErrorOutcome(err), start)
		return nil, err
	}
	id, err := uuid.Parse(requestID)
	if err != nil {
		s.observeCommand("accept", "validation_failed", start)
		return nil, ErrInvalidRequestID
	}
	f, other, err := s.repo.AcceptRequest(ctx, id, actor)
	if err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("accept", outcome, start)
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			return nil, ErrDependencyFailure
		}
		return nil, err
	}
	s.observeCommand("accept", "success", start)
	s.logger.InfoContext(ctx, "friend request accepted",
		slog.String("user_id", actor.String()),
		slog.String("request_id", f.ID.String()),
	)
	return &FriendshipResponse{Data: toFriendDTO(RelationshipRow{Friendship: *f, Other: *other})}, nil
}

// DeclineRequest declines (deletes) a pending friend request.
func (s *Service) DeclineRequest(ctx context.Context, sess session.Context, requestID string) error {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("decline", mapErrorOutcome(err), start)
		return err
	}
	id, err := uuid.Parse(requestID)
	if err != nil {
		s.observeCommand("decline", "validation_failed", start)
		return ErrInvalidRequestID
	}
	if err := s.repo.DeclineRequest(ctx, id, actor); err != nil {
		outcome := mapErrorOutcome(err)
		s.observeCommand("decline", outcome, start)
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			return ErrDependencyFailure
		}
		return err
	}
	s.observeCommand("decline", "success", start)
	s.logger.InfoContext(ctx, "friend request declined",
		slog.String("user_id", actor.String()),
		slog.String("request_id", id.String()),
	)
	return nil
}

// ListFriends returns accepted friends.
func (s *Service) ListFriends(ctx context.Context, sess session.Context, limit int, cursor string) (*FriendListResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.metrics.ObserveList("friends", mapErrorOutcome(err))
		return nil, err
	}
	limit, err = normalizeLimit(limit)
	if err != nil {
		s.metrics.ObserveList("friends", "validation_failed")
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		s.metrics.ObserveList("friends", "validation_failed")
		return nil, err
	}
	page, err := s.repo.ListAcceptedFriends(ctx, actor, limit, cursor)
	if err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.metrics.ObserveList("friends", "error")
			return nil, ErrDependencyFailure
		}
		s.metrics.ObserveList("friends", "error")
		return nil, err
	}
	items := make([]FriendDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toFriendDTO(row))
	}
	s.metrics.ObserveList("friends", "success")
	return &FriendListResponse{
		Data: items,
		Page: apphttp.PageInfo{Limit: page.Limit, NextCursor: page.NextCursor},
	}, nil
}

// RemoveFriend removes an accepted friendship (idempotent).
func (s *Service) RemoveFriend(ctx context.Context, sess session.Context, otherUserID string) error {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("remove", mapErrorOutcome(err), start)
		return err
	}
	other, err := uuid.Parse(otherUserID)
	if err != nil {
		s.observeCommand("remove", "validation_failed", start)
		return ErrInvalidUserID
	}
	if actor == other {
		s.observeCommand("remove", "validation_failed", start)
		return ErrSelfPair
	}
	if err := s.repo.RemoveFriendship(ctx, actor, other); err != nil {
		if errors.Is(err, ErrSelfPair) || errors.Is(err, ErrInvalidUserID) {
			s.observeCommand("remove", "validation_failed", start)
			return err
		}
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.observeCommand("remove", "error", start)
			return ErrDependencyFailure
		}
		s.observeCommand("remove", "error", start)
		return err
	}
	s.observeCommand("remove", "success", start)
	s.logger.InfoContext(ctx, "friend removed", slog.String("user_id", actor.String()))
	return nil
}

// Block blocks a target user (idempotent for same blocker).
func (s *Service) Block(ctx context.Context, sess session.Context, targetUserID string) error {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("block", mapErrorOutcome(err), start)
		return err
	}
	target, err := uuid.Parse(targetUserID)
	if err != nil {
		s.observeCommand("block", "validation_failed", start)
		return ErrInvalidUserID
	}
	if actor == target {
		s.observeCommand("block", "validation_failed", start)
		return ErrSelfPair
	}
	if err := s.repo.BlockUser(ctx, actor, target); err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.observeCommand("block", "error", start)
			return ErrDependencyFailure
		}
		s.observeCommand("block", mapErrorOutcome(err), start)
		return err
	}
	s.observeCommand("block", "success", start)
	s.logger.InfoContext(ctx, "user blocked", slog.String("user_id", actor.String()))
	return nil
}

// Unblock removes a block owned by the caller (privacy-safe no-op otherwise).
func (s *Service) Unblock(ctx context.Context, sess session.Context, targetUserID string) error {
	start := time.Now()
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.observeCommand("unblock", mapErrorOutcome(err), start)
		return err
	}
	target, err := uuid.Parse(targetUserID)
	if err != nil {
		s.observeCommand("unblock", "validation_failed", start)
		return ErrInvalidUserID
	}
	if actor == target {
		s.observeCommand("unblock", "validation_failed", start)
		return ErrSelfPair
	}
	if err := s.repo.UnblockUser(ctx, actor, target); err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.observeCommand("unblock", "error", start)
			return ErrDependencyFailure
		}
		s.observeCommand("unblock", mapErrorOutcome(err), start)
		return err
	}
	s.observeCommand("unblock", "success", start)
	s.logger.InfoContext(ctx, "user unblocked", slog.String("user_id", actor.String()))
	return nil
}

// ListBlocked returns users blocked by the caller.
func (s *Service) ListBlocked(ctx context.Context, sess session.Context, limit int, cursor string) (*BlockedListResponse, error) {
	actor, err := s.requireActiveRegistered(ctx, sess)
	if err != nil {
		s.metrics.ObserveList("blocked", mapErrorOutcome(err))
		return nil, err
	}
	limit, err = normalizeLimit(limit)
	if err != nil {
		s.metrics.ObserveList("blocked", "validation_failed")
		return nil, err
	}
	if err := validateCursor(cursor); err != nil {
		s.metrics.ObserveList("blocked", "validation_failed")
		return nil, err
	}
	page, err := s.repo.ListBlockedUsers(ctx, actor, limit, cursor)
	if err != nil {
		if isDependency(err) {
			s.metrics.ObserveDependencyFailure("postgres")
			s.metrics.ObserveList("blocked", "error")
			return nil, ErrDependencyFailure
		}
		s.metrics.ObserveList("blocked", "error")
		return nil, err
	}
	items := make([]BlockedUserDTO, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toBlockedDTO(row))
	}
	s.metrics.ObserveList("blocked", "success")
	return &BlockedListResponse{
		Data: items,
		Page: apphttp.PageInfo{Limit: page.Limit, NextCursor: page.NextCursor},
	}, nil
}

func (s *Service) observeCommand(command, outcome string, start time.Time) {
	if s.metrics == nil {
		return
	}
	s.metrics.ObserveCommand(command, outcome, time.Since(start))
}

func mapErrorOutcome(err error) string {
	switch {
	case errors.Is(err, ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrTargetNotFound):
		return "not_found"
	case errors.Is(err, ErrAlreadyFriends), errors.Is(err, ErrAlreadyPending), errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrInvalidRequestID),
		errors.Is(err, ErrSelfPair), errors.Is(err, ErrInvalidCursor), errors.Is(err, ErrInvalidLimit):
		return "validation_failed"
	case errors.Is(err, ErrInvalidState):
		return "unprocessable"
	case errors.Is(err, ErrDependencyFailure):
		return "unavailable"
	default:
		return "error"
	}
}

func isDependency(err error) bool {
	if err == nil {
		return false
	}
	for _, known := range []error{
		ErrUnauthorized, ErrInvalidUserID, ErrInvalidRequestID, ErrSelfPair, ErrInvalidState,
		ErrInvalidCursor, ErrInvalidLimit, ErrNotFound, ErrTargetNotFound, ErrConflict,
		ErrAlreadyFriends, ErrAlreadyPending, ErrRateLimited, ErrInvalidJSON, ErrInvalidRequest,
		ErrDependencyFailure,
	} {
		if errors.Is(err, known) {
			return false
		}
	}
	return true
}
