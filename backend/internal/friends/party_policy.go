package friends

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// PartyInvitePolicy is the narrow friends surface consumed by parties.
// Implementations MUST NOT leak which side blocked the other.
type PartyInvitePolicy interface {
	// CanInvite reports whether inviter may send a party invite to invitee.
	// Missing/inactive targets, non-friends, and either-direction blocks all
	// return ErrFriendUnavailable without revealing the reason.
	CanInvite(ctx context.Context, inviter, invitee uuid.UUID) error
}

// ErrFriendUnavailable is the privacy-safe rejection for party invites.
// Parties map this to friend_unavailable without consulting block ownership.
var ErrFriendUnavailable = errors.New("friend unavailable for party invite")

// partyPolicyStore is the persistence surface needed by PartyPolicy.
type partyPolicyStore interface {
	FindActiveUser(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
	GetByPair(ctx context.Context, userA, userB uuid.UUID) (*Friendship, error)
}

// PartyPolicy evaluates accepted-friend and either-direction block rules for party invites.
type PartyPolicy struct {
	store partyPolicyStore
}

// NewPartyPolicy returns a party invite policy backed by the friends repository.
func NewPartyPolicy(store partyPolicyStore) *PartyPolicy {
	return &PartyPolicy{store: store}
}

// CanInvite enforces accepted friendship with no block in either direction.
func (p *PartyPolicy) CanInvite(ctx context.Context, inviter, invitee uuid.UUID) error {
	if p == nil || p.store == nil {
		return fmt.Errorf("party policy: store required")
	}
	if inviter == uuid.Nil || invitee == uuid.Nil {
		return ErrFriendUnavailable
	}
	if inviter == invitee {
		return ErrFriendUnavailable
	}

	activeInvitee, err := p.store.FindActiveUser(ctx, invitee)
	if err != nil {
		return err
	}
	if activeInvitee == nil {
		return ErrFriendUnavailable
	}

	userA, userB, err := NormalizePair(inviter, invitee)
	if err != nil {
		return ErrFriendUnavailable
	}

	f, err := p.store.GetByPair(ctx, userA, userB)
	if err != nil {
		return err
	}
	if f == nil {
		return ErrFriendUnavailable
	}
	switch f.Status {
	case StatusAccepted:
		return nil
	case StatusBlocked, StatusPending:
		return ErrFriendUnavailable
	default:
		return ErrFriendUnavailable
	}
}

// Ensure Repository satisfies partyPolicyStore at compile time.
var _ partyPolicyStore = (*Repository)(nil)

// Ensure PartyPolicy satisfies PartyInvitePolicy at compile time.
var _ PartyInvitePolicy = (*PartyPolicy)(nil)
