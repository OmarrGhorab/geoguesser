package friends

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Friendship status constants.
const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusBlocked  = "blocked"
)

// Friendship is the durable undirected social edge between two users.
// user_a_id and user_b_id are always stored in sorted UUID order.
type Friendship struct {
	ID                uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserAID           uuid.UUID  `gorm:"column:user_a_id;type:uuid;not null"`
	UserBID           uuid.UUID  `gorm:"column:user_b_id;type:uuid;not null"`
	RequestedByUserID uuid.UUID  `gorm:"column:requested_by_user_id;type:uuid;not null"`
	Status            string     `gorm:"column:status;not null"`
	BlockedByUserID   *uuid.UUID `gorm:"column:blocked_by_user_id;type:uuid"`
	AcceptedAt        *time.Time `gorm:"column:accepted_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null"`
}

// TableName returns the friendships table name.
func (Friendship) TableName() string { return "friendships" }

// OtherUserID returns the counterpart user for the given participant.
func (f Friendship) OtherUserID(viewer uuid.UUID) (uuid.UUID, error) {
	switch viewer {
	case f.UserAID:
		return f.UserBID, nil
	case f.UserBID:
		return f.UserAID, nil
	default:
		return uuid.Nil, fmt.Errorf("viewer is not a friendship participant")
	}
}

// NormalizePair returns sorted user UUIDs or an error when the pair is invalid.
func NormalizePair(a, b uuid.UUID) (userA, userB uuid.UUID, err error) {
	if a == uuid.Nil || b == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalidUserID
	}
	if a == b {
		return uuid.Nil, uuid.Nil, ErrSelfPair
	}
	if bytes.Compare(a[:], b[:]) < 0 {
		return a, b, nil
	}
	return b, a, nil
}

// ValidateStatusLifecycle checks status-dependent field invariants.
func ValidateStatusLifecycle(status string, acceptedAt *time.Time, blockedBy *uuid.UUID) error {
	switch status {
	case StatusPending:
		if acceptedAt != nil || blockedBy != nil {
			return ErrInvalidState
		}
	case StatusAccepted:
		if acceptedAt == nil || blockedBy != nil {
			return ErrInvalidState
		}
	case StatusBlocked:
		if blockedBy == nil || acceptedAt != nil {
			return ErrInvalidState
		}
	default:
		return ErrInvalidState
	}
	return nil
}

// PublicProfile is a privacy-safe profile projection joined for list endpoints.
type PublicProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CountryCode *string
}

// RelationshipRow is a friendship plus the counterpart public profile.
type RelationshipRow struct {
	Friendship Friendship
	Other      PublicProfile
}

// Page is a cursor page of relationship rows.
type Page struct {
	Items      []RelationshipRow
	Limit      int
	NextCursor *string
}
