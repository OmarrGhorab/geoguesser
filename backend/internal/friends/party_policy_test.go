package friends_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/friends"
)

type policyStore struct {
	active map[uuid.UUID]bool
	edges  map[string]*friends.Friendship
	err    error
}

func pairKey(a, b uuid.UUID) string {
	ua, ub, err := friends.NormalizePair(a, b)
	if err != nil {
		return a.String() + "|" + b.String()
	}
	return ua.String() + "|" + ub.String()
}

func (s *policyStore) FindActiveUser(_ context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.active != nil && !s.active[userID] {
		return nil, nil
	}
	id := userID
	return &id, nil
}

func (s *policyStore) GetByPair(_ context.Context, userA, userB uuid.UUID) (*friends.Friendship, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.edges == nil {
		return nil, nil
	}
	return s.edges[pairKey(userA, userB)], nil
}

func TestPartyPolicyAcceptedFriendsOnly(t *testing.T) {
	inviter := uuid.New()
	invitee := uuid.New()
	now := time.Now().UTC()
	store := &policyStore{
		active: map[uuid.UUID]bool{inviter: true, invitee: true},
		edges: map[string]*friends.Friendship{
			pairKey(inviter, invitee): {
				ID:         uuid.New(),
				Status:     friends.StatusAccepted,
				AcceptedAt: &now,
			},
		},
	}
	policy := friends.NewPartyPolicy(store)
	if err := policy.CanInvite(context.Background(), inviter, invitee); err != nil {
		t.Fatalf("accepted friends should allow invite: %v", err)
	}
}

func TestPartyPolicyRejectsNonFriends(t *testing.T) {
	inviter := uuid.New()
	invitee := uuid.New()
	store := &policyStore{active: map[uuid.UUID]bool{inviter: true, invitee: true}}
	policy := friends.NewPartyPolicy(store)
	if err := policy.CanInvite(context.Background(), inviter, invitee); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("err = %v, want friend unavailable", err)
	}
}

func TestPartyPolicyRejectsPending(t *testing.T) {
	inviter := uuid.New()
	invitee := uuid.New()
	store := &policyStore{
		active: map[uuid.UUID]bool{inviter: true, invitee: true},
		edges: map[string]*friends.Friendship{
			pairKey(inviter, invitee): {ID: uuid.New(), Status: friends.StatusPending},
		},
	}
	policy := friends.NewPartyPolicy(store)
	if err := policy.CanInvite(context.Background(), inviter, invitee); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("err = %v, want friend unavailable", err)
	}
}

func TestPartyPolicyEitherDirectionBlock(t *testing.T) {
	inviter := uuid.New()
	invitee := uuid.New()
	blocker := inviter
	store := &policyStore{
		active: map[uuid.UUID]bool{inviter: true, invitee: true},
		edges: map[string]*friends.Friendship{
			pairKey(inviter, invitee): {
				ID:              uuid.New(),
				Status:          friends.StatusBlocked,
				BlockedByUserID: &blocker,
			},
		},
	}
	policy := friends.NewPartyPolicy(store)
	if err := policy.CanInvite(context.Background(), inviter, invitee); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("inviter-blocked: err = %v", err)
	}

	// Invitee blocked inviter — same privacy-safe error.
	blocker = invitee
	store.edges[pairKey(inviter, invitee)].BlockedByUserID = &blocker
	if err := policy.CanInvite(context.Background(), inviter, invitee); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("invitee-blocked: err = %v", err)
	}
}

func TestPartyPolicyInactiveOrMissingInvitee(t *testing.T) {
	inviter := uuid.New()
	invitee := uuid.New()
	store := &policyStore{
		active: map[uuid.UUID]bool{inviter: true}, // invitee missing
	}
	policy := friends.NewPartyPolicy(store)
	if err := policy.CanInvite(context.Background(), inviter, invitee); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("inactive: err = %v", err)
	}
}

func TestPartyPolicySelfRejected(t *testing.T) {
	id := uuid.New()
	policy := friends.NewPartyPolicy(&policyStore{active: map[uuid.UUID]bool{id: true}})
	if err := policy.CanInvite(context.Background(), id, id); !errors.Is(err, friends.ErrFriendUnavailable) {
		t.Fatalf("self: err = %v", err)
	}
}

func TestPartyPolicyDoesNotLeakBlockOwnership(t *testing.T) {
	// All negative outcomes must be the same sentinel; never ErrNotFound with block details.
	inviter := uuid.New()
	invitee := uuid.New()
	cases := []struct {
		name  string
		store *policyStore
	}{
		{"missing edge", &policyStore{active: map[uuid.UUID]bool{inviter: true, invitee: true}}},
		{"blocked by inviter", &policyStore{
			active: map[uuid.UUID]bool{inviter: true, invitee: true},
			edges: map[string]*friends.Friendship{
				pairKey(inviter, invitee): {Status: friends.StatusBlocked, BlockedByUserID: &inviter},
			},
		}},
		{"blocked by invitee", &policyStore{
			active: map[uuid.UUID]bool{inviter: true, invitee: true},
			edges: map[string]*friends.Friendship{
				pairKey(inviter, invitee): {Status: friends.StatusBlocked, BlockedByUserID: &invitee},
			},
		}},
		{"inactive invitee", &policyStore{active: map[uuid.UUID]bool{inviter: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := friends.NewPartyPolicy(tc.store).CanInvite(context.Background(), inviter, invitee)
			if !errors.Is(err, friends.ErrFriendUnavailable) {
				t.Fatalf("err = %v, want ErrFriendUnavailable", err)
			}
			if errors.Is(err, friends.ErrNotFound) || errors.Is(err, friends.ErrTargetNotFound) {
				t.Fatalf("must not surface not_found variants: %v", err)
			}
		})
	}
}
