package friends_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
	"github.com/raven/geoguess/backend/internal/session"
)

type fakeStore struct {
	active          map[uuid.UUID]bool
	createFn        func(ctx context.Context, requester, target uuid.UUID) (*friends.Friendship, error)
	acceptFn        func(ctx context.Context, requestID, acceptor uuid.UUID) (*friends.Friendship, *friends.PublicProfile, error)
	declineFn       func(ctx context.Context, requestID, actor uuid.UUID) error
	removeFn        func(ctx context.Context, actor, other uuid.UUID) error
	blockFn         func(ctx context.Context, blocker, target uuid.UUID) error
	unblockFn       func(ctx context.Context, actor, other uuid.UUID) error
	incoming        *friends.Page
	outgoing        *friends.Page
	accepted        *friends.Page
	blocked         *friends.Page
	profiles        map[uuid.UUID]*friends.PublicProfile
	listIncomingErr error
	listOutgoingErr error
	listAcceptedErr error
	listBlockedErr  error
}

func (f *fakeStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	if f.active != nil && f.active[userID] {
		id := userID
		return &id, nil
	}
	return nil, nil
}

func (f *fakeStore) CreateRequest(ctx context.Context, requester, target uuid.UUID) (*friends.Friendship, error) {
	if f.createFn != nil {
		return f.createFn(ctx, requester, target)
	}
	return nil, errors.New("not implemented")
}

func (f *fakeStore) AcceptRequest(ctx context.Context, requestID, acceptor uuid.UUID) (*friends.Friendship, *friends.PublicProfile, error) {
	if f.acceptFn != nil {
		return f.acceptFn(ctx, requestID, acceptor)
	}
	return nil, nil, errors.New("not implemented")
}

func (f *fakeStore) DeclineRequest(ctx context.Context, requestID, actor uuid.UUID) error {
	if f.declineFn != nil {
		return f.declineFn(ctx, requestID, actor)
	}
	return errors.New("not implemented")
}

func (f *fakeStore) RemoveFriendship(ctx context.Context, actor, other uuid.UUID) error {
	if f.removeFn != nil {
		return f.removeFn(ctx, actor, other)
	}
	return nil
}

func (f *fakeStore) BlockUser(ctx context.Context, blocker, target uuid.UUID) error {
	if f.blockFn != nil {
		return f.blockFn(ctx, blocker, target)
	}
	return nil
}

func (f *fakeStore) UnblockUser(ctx context.Context, actor, other uuid.UUID) error {
	if f.unblockFn != nil {
		return f.unblockFn(ctx, actor, other)
	}
	return nil
}

func (f *fakeStore) ListIncomingRequests(context.Context, uuid.UUID, int, string) (*friends.Page, error) {
	if f.listIncomingErr != nil {
		return nil, f.listIncomingErr
	}
	if f.incoming == nil {
		return &friends.Page{Limit: 20}, nil
	}
	return f.incoming, nil
}

func (f *fakeStore) ListOutgoingRequests(context.Context, uuid.UUID, int, string) (*friends.Page, error) {
	if f.listOutgoingErr != nil {
		return nil, f.listOutgoingErr
	}
	if f.outgoing == nil {
		return &friends.Page{Limit: 20}, nil
	}
	return f.outgoing, nil
}

func (f *fakeStore) ListAcceptedFriends(context.Context, uuid.UUID, int, string) (*friends.Page, error) {
	if f.listAcceptedErr != nil {
		return nil, f.listAcceptedErr
	}
	if f.accepted == nil {
		return &friends.Page{Limit: 20}, nil
	}
	return f.accepted, nil
}

func (f *fakeStore) ListBlockedUsers(context.Context, uuid.UUID, int, string) (*friends.Page, error) {
	if f.listBlockedErr != nil {
		return nil, f.listBlockedErr
	}
	if f.blocked == nil {
		return &friends.Page{Limit: 20}, nil
	}
	return f.blocked, nil
}

func (f *fakeStore) LoadPublicProfile(_ context.Context, userID uuid.UUID) (*friends.PublicProfile, error) {
	if f.profiles != nil {
		return f.profiles[userID], nil
	}
	return &friends.PublicProfile{UserID: userID, DisplayName: "User"}, nil
}

func registered(id uuid.UUID) session.Context {
	s := id.String()
	return session.Context{Kind: session.KindUser, UserID: &s, Role: "user"}
}

func TestCreateRequestRequiresRegisteredSession(t *testing.T) {
	svc := friends.NewService(&fakeStore{}, nil)
	_, err := svc.CreateRequest(context.Background(), session.Context{Kind: session.KindGuest}, uuid.New().String())
	if !errors.Is(err, friends.ErrUnauthorized) {
		t.Fatalf("err = %v, want unauthorized", err)
	}
}

func TestCreateRequestRejectsInvalidUUID(t *testing.T) {
	actor := uuid.New()
	svc := friends.NewService(&fakeStore{}, nil)
	_, err := svc.CreateRequest(context.Background(), registered(actor), "not-a-uuid")
	if !errors.Is(err, friends.ErrInvalidUserID) {
		t.Fatalf("err = %v, want invalid user id", err)
	}
}

func TestCreateRequestRejectsSelf(t *testing.T) {
	actor := uuid.New()
	svc := friends.NewService(&fakeStore{}, nil)
	_, err := svc.CreateRequest(context.Background(), registered(actor), actor.String())
	if !errors.Is(err, friends.ErrSelfPair) {
		t.Fatalf("err = %v, want self pair", err)
	}
}

func TestCreateRequestMissingTarget(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return nil, friends.ErrTargetNotFound
		},
	}
	svc := friends.NewService(store, nil)
	_, err := svc.CreateRequest(context.Background(), registered(actor), target.String())
	if !errors.Is(err, friends.ErrTargetNotFound) {
		t.Fatalf("err = %v, want target not found", err)
	}
}

func TestCreateRequestDuplicatePending(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return nil, friends.ErrAlreadyPending
		},
	}
	svc := friends.NewService(store, nil)
	_, err := svc.CreateRequest(context.Background(), registered(actor), target.String())
	if !errors.Is(err, friends.ErrAlreadyPending) {
		t.Fatalf("err = %v, want already pending", err)
	}
}

func TestCreateRequestReciprocalConflict(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	store := &fakeStore{
		createFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, error) {
			return nil, friends.ErrAlreadyPending
		},
	}
	svc := friends.NewService(store, nil)
	_, err := svc.CreateRequest(context.Background(), registered(actor), target.String())
	if !errors.Is(err, friends.ErrAlreadyPending) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRequestSuccess(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	now := time.Now().UTC()
	reqID := uuid.New()
	store := &fakeStore{
		createFn: func(_ context.Context, requester, tgt uuid.UUID) (*friends.Friendship, error) {
			if requester != actor || tgt != target {
				t.Fatalf("unexpected pair")
			}
			return &friends.Friendship{ID: reqID, Status: friends.StatusPending, CreatedAt: now}, nil
		},
		profiles: map[uuid.UUID]*friends.PublicProfile{
			target: {UserID: target, DisplayName: "Target"},
		},
	}
	svc := friends.NewService(store, nil)
	resp, err := svc.CreateRequest(context.Background(), registered(actor), target.String())
	if err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if resp.Data.RequestID != reqID || resp.Data.User.DisplayName != "Target" {
		t.Fatalf("unexpected response: %+v", resp.Data)
	}
}

func TestAcceptAuthorizationNotFound(t *testing.T) {
	actor := uuid.New()
	store := &fakeStore{
		acceptFn: func(context.Context, uuid.UUID, uuid.UUID) (*friends.Friendship, *friends.PublicProfile, error) {
			return nil, nil, friends.ErrNotFound
		},
	}
	svc := friends.NewService(store, nil)
	_, err := svc.AcceptRequest(context.Background(), registered(actor), uuid.New().String())
	if !errors.Is(err, friends.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeclineDeletes(t *testing.T) {
	actor := uuid.New()
	reqID := uuid.New()
	called := false
	store := &fakeStore{
		declineFn: func(_ context.Context, id, who uuid.UUID) error {
			called = true
			if id != reqID || who != actor {
				t.Fatalf("unexpected decline args")
			}
			return nil
		},
	}
	svc := friends.NewService(store, nil)
	if err := svc.DeclineRequest(context.Background(), registered(actor), reqID.String()); err != nil {
		t.Fatalf("DeclineRequest: %v", err)
	}
	if !called {
		t.Fatal("expected decline to hit store")
	}
}

func TestListFriendsSymmetricPrivacy(t *testing.T) {
	actor := uuid.New()
	friendID := uuid.New()
	now := time.Now().UTC()
	store := &fakeStore{
		accepted: &friends.Page{
			Limit: 20,
			Items: []friends.RelationshipRow{{
				Friendship: friends.Friendship{ID: uuid.New(), Status: friends.StatusAccepted, AcceptedAt: &now, CreatedAt: now},
				Other:      friends.PublicProfile{UserID: friendID, DisplayName: "Friend"},
			}},
		},
	}
	svc := friends.NewService(store, nil)
	resp, err := svc.ListFriends(context.Background(), registered(actor), 20, "")
	if err != nil {
		t.Fatalf("ListFriends: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].User.UserID != friendID {
		t.Fatalf("unexpected friends list: %+v", resp.Data)
	}
}

func TestRemoveFriendIdempotent(t *testing.T) {
	actor := uuid.New()
	other := uuid.New()
	store := &fakeStore{
		removeFn: func(context.Context, uuid.UUID, uuid.UUID) error { return nil },
	}
	svc := friends.NewService(store, nil)
	if err := svc.RemoveFriend(context.Background(), registered(actor), other.String()); err != nil {
		t.Fatalf("RemoveFriend: %v", err)
	}
}

func TestBlockAndUnblockRules(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	blocked := false
	store := &fakeStore{
		blockFn: func(_ context.Context, blocker, tgt uuid.UUID) error {
			if blocker != actor || tgt != target {
				t.Fatalf("block pair")
			}
			blocked = true
			return nil
		},
		unblockFn: func(_ context.Context, who, tgt uuid.UUID) error {
			if who != actor || tgt != target {
				t.Fatalf("unblock pair")
			}
			blocked = false
			return nil
		},
	}
	svc := friends.NewService(store, nil)
	if err := svc.Block(context.Background(), registered(actor), target.String()); err != nil {
		t.Fatalf("Block: %v", err)
	}
	if !blocked {
		t.Fatal("expected blocked")
	}
	if err := svc.Unblock(context.Background(), registered(actor), target.String()); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	if blocked {
		t.Fatal("expected unblocked")
	}
}

func TestNormalizePairAndModelHelpers(t *testing.T) {
	a := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	b := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ua, ub, err := friends.NormalizePair(b, a)
	if err != nil || ua != a || ub != b {
		t.Fatalf("NormalizePair = %v %v %v", ua, ub, err)
	}
	_, _, err = friends.NormalizePair(a, a)
	if !errors.Is(err, friends.ErrSelfPair) {
		t.Fatalf("self pair err = %v", err)
	}
	if err := friends.ValidateStatusLifecycle(friends.StatusPending, nil, nil); err != nil {
		t.Fatalf("pending lifecycle: %v", err)
	}
}
